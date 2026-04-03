package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// PythonRecipe builds Python, matching the Ruby builder's build_python method
// in builder.rb exactly.
//
// Critical ordering: configure runs BEFORE libdb-dev/libgdbm-dev are installed,
// so _dbm and _gdbm modules are NOT detected/built (matching Ruby output).
// Then packages are installed, tcl/tk debs are downloaded and extracted, and
// finally make && make install are run.
type PythonRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PythonRecipe) Name() string { return "python" }
func (p *PythonRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (p *PythonRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	a := apt.New(run)

	tclVersion := s.Python.TCLVersion // e.g. "8.6"
	aptFlag := "-y"
	if s.Python.UseForceYes {
		aptFlag = "--force-yes"
	}
	// debPkgs are .deb packages that are downloaded and extracted (not installed)
	// into the build prefix to bundle tcl/tk and its X11 dependencies.
	// libtcl/libtk versions come from s.Python.TCLVersion (stack config).
	// Additional packages (e.g. libxss1) live in s.AptPackages["python_deb_extras"]
	// so they can be adjusted per stack without modifying Go source.
	debPkgs := append(
		[]string{
			fmt.Sprintf("libtcl%s", tclVersion),
			fmt.Sprintf("libtk%s", tclVersion),
		},
		s.AptPackages["python_deb_extras"]...,
	)

	builtPath := fmt.Sprintf("/app/vendor/python-%s", src.Version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("python-%s-linux-x64.tgz", src.Version))

	// Download Python source tarball.
	srcTarball := fmt.Sprintf("/tmp/Python-%s.tgz", src.Version)
	if err := p.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("python: downloading source: %w", err)
	}

	// Extract source.
	srcDir := fmt.Sprintf("/tmp/Python-%s", src.Version)
	if err := run.Run("mkdir", "-p", srcDir); err != nil {
		return err
	}
	if err := run.Run("tar", "xf", srcTarball, "-C", "/tmp"); err != nil {
		return fmt.Errorf("python: extracting source: %w", err)
	}

	// Create install prefix dir.
	if err := run.Run("mkdir", "-p", builtPath); err != nil {
		return err
	}

	// Configure flags reference the tcl/tk version from stack config.
	tclInclude := fmt.Sprintf("-I/usr/include/tcl%s", tclVersion)
	tclLib := fmt.Sprintf("-L/usr/lib/x86_64-linux-gnu -ltcl%s -ltk%s", tclVersion, tclVersion)

	// Step 1: Run ./configure BEFORE installing libdb-dev/libgdbm-dev.
	// This matches the Ruby builder which also runs configure first, then
	// installs packages. Because db/gdbm headers are absent at configure time,
	// Python will NOT build _dbm or _gdbm extension modules.
	configureArgs := []string{
		fmt.Sprintf("--prefix=%s", builtPath),
		"--enable-shared",
		"--with-ensurepip=yes",
		"--with-dbmliborder=bdb:gdbm",
		fmt.Sprintf("--with-tcltk-includes=%s", tclInclude),
		fmt.Sprintf("--with-tcltk-libs=%s", tclLib),
		"--enable-unicode=ucs4",
	}
	if err := run.RunInDir(srcDir, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("python: configure: %w", err)
	}

	// Step 2: Install build packages AFTER configure (matching Ruby builder order).
	if err := a.Install(ctx, s.AptPackages["python_build"]...); err != nil {
		return fmt.Errorf("python: apt install python_build: %w", err)
	}

	// Step 3: Download tcl/tk .deb packages (without installing).
	aptArgs := append([]string{aptFlag, "-y", "-d", "install", "--reinstall"}, debPkgs...)
	if err := run.RunWithEnv(map[string]string{"DEBIAN_FRONTEND": "noninteractive"}, "apt-get", aptArgs...); err != nil {
		return fmt.Errorf("python: downloading tcl/tk debs: %w", err)
	}

	// Step 4: Extract each tcl/tk .deb into the install prefix to bundle them.
	aptCacheDir := "/var/cache/apt/archives"
	for _, pkg := range debPkgs {
		if err := run.Run("sh", "-c", fmt.Sprintf("dpkg -x %s/%s_*.deb %s", aptCacheDir, pkg, builtPath)); err != nil {
			return fmt.Errorf("python: dpkg -x %s: %w", pkg, err)
		}
	}

	// Step 5: Compile and install.
	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("python: make: %w", err)
	}
	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("python: make install: %w", err)
	}

	// Step 6: Create bin/python symlink → ./python3 (relative, matching Ruby recipe's
	// File.symlink('./python3', "#{destdir}/bin/python")).
	pythonLink := fmt.Sprintf("%s/bin/python", builtPath)
	if err := run.Run("ln", "-sf", "./python3", pythonLink); err != nil {
		return fmt.Errorf("python: creating bin/python symlink: %w", err)
	}

	// Step 7: Pack with --hard-dereference to resolve all symlinks into the artifact.
	if err := archive.PackWithDereference(run, artifactPath, builtPath); err != nil {
		return fmt.Errorf("python: packing artifact: %w", err)
	}

	return nil
}
