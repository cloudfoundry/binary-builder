// Package autoconf provides a hook-based build engine for software that uses
// the standard autoconf configure/make/make-install cycle.
//
// Recipe is a pure build engine — it does not implement the recipe.Recipe
// interface directly. Thin wrappers in internal/recipe/ embed or delegate to
// Recipe.Build and expose the Name/Artifact methods required by recipe.Recipe.
// This avoids an import cycle between internal/recipe and internal/autoconf.
//
// Hook fields are all func types; nil means "use default behaviour".
package autoconf

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// Hooks holds all optional customisation points for Recipe. A nil hook means
// "use the default behaviour described in the field comment".
type Hooks struct {
	// AptPackages returns the list of apt packages to install before building.
	// Default: s.AptPackages["{name}_build"]
	AptPackages func(s *stack.Stack) []string

	// SourceProvider downloads or otherwise prepares the source tree and returns
	// the path to the extracted source directory.
	// Default: fetch tarball from src.URL, extract to /tmp/<name>-<version>
	SourceProvider func(ctx context.Context, src *source.Input, f fetch.Fetcher, r runner.Runner) (srcDir string, err error)

	// BeforeDownload runs before the source tarball is downloaded (or before
	// SourceProvider is called). Typical use: GPG signature verification.
	// Default: no-op
	BeforeDownload func(ctx context.Context, src *source.Input, r runner.Runner) error

	// AfterExtract runs inside srcDir immediately after extraction.
	// Typical use: autoreconf -i, autogen.sh.
	// Default: no-op
	AfterExtract func(ctx context.Context, srcDir string, prefix string, r runner.Runner) error

	// ConfigureArgs returns the full list of arguments for ./configure.
	// Default: ["--prefix=<prefix>"]
	ConfigureArgs func(srcDir, prefix string) []string

	// ConfigureEnv provides additional environment variables for ./configure and make.
	// Default: nil (no extra env)
	ConfigureEnv func() map[string]string

	// MakeArgs returns extra arguments for the make step.
	// Default: nil (plain "make")
	MakeArgs func() []string

	// InstallEnv provides extra environment variables for make install.
	// Default: nil (no extra env, same as ConfigureEnv result)
	InstallEnv func(prefix string) map[string]string

	// AfterInstall runs after make install, inside the prefix directory.
	// Typical use: remove runtime dirs, move/rename files.
	// Default: no-op
	AfterInstall func(ctx context.Context, prefix string, r runner.Runner) error

	// PackDirs lists the sub-directories of prefix to pack into the artifact tarball.
	// Default: ["."] (pack the entire prefix)
	PackDirs func() []string

	// AfterPack runs after the artifact tarball is created.
	// Typical use: archive.StripTopLevelDir for nginx.
	// Default: no-op
	AfterPack func(artifactPath string) error
}

// Recipe is a hook-based build engine for autoconf-based dependencies.
// It does NOT implement recipe.Recipe; use a thin wrapper in internal/recipe/.
type Recipe struct {
	DepName string
	Fetcher fetch.Fetcher
	Hooks   Hooks
}

func (r *Recipe) Name() string { return r.DepName }

// Build runs the full configure/make/make-install cycle with hook customisation.
func (r *Recipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	name := r.DepName
	version := src.Version

	// ── Step 1: apt install build dependencies ────────────────────────────────
	var pkgs []string
	if r.Hooks.AptPackages != nil {
		pkgs = r.Hooks.AptPackages(s)
	} else {
		pkgs = s.AptPackages[fmt.Sprintf("%s_build", name)]
	}
	if len(pkgs) > 0 {
		a := apt.New(run)
		if err := a.Install(ctx, pkgs...); err != nil {
			return fmt.Errorf("%s: apt install %s_build: %w", name, name, err)
		}
	}

	// ── Step 2: before-download hook (e.g. GPG verification) ─────────────────
	if r.Hooks.BeforeDownload != nil {
		if err := r.Hooks.BeforeDownload(ctx, src, run); err != nil {
			return fmt.Errorf("%s: before_download: %w", name, err)
		}
	}

	// ── Step 3: provide source ────────────────────────────────────────────────
	builtPath := fmt.Sprintf("/tmp/%s-built-%s", name, version)
	prefix := builtPath

	var srcDir string
	if r.Hooks.SourceProvider != nil {
		var err error
		srcDir, err = r.Hooks.SourceProvider(ctx, src, r.Fetcher, run)
		if err != nil {
			return fmt.Errorf("%s: source provider: %w", name, err)
		}
	} else {
		srcDir = fmt.Sprintf("/tmp/%s-%s", name, version)
		srcTarball := fmt.Sprintf("/tmp/%s-%s.tar.gz", name, version)
		if err := r.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
			return fmt.Errorf("%s: downloading source: %w", name, err)
		}
		if err := run.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
			return fmt.Errorf("%s: extracting source: %w", name, err)
		}
	}

	if err := run.Run("mkdir", "-p", builtPath); err != nil {
		return fmt.Errorf("%s: mkdir prefix: %w", name, err)
	}

	// ── Step 4: after-extract hook ────────────────────────────────────────────
	if r.Hooks.AfterExtract != nil {
		if err := r.Hooks.AfterExtract(ctx, srcDir, prefix, run); err != nil {
			return fmt.Errorf("%s: after_extract: %w", name, err)
		}
	}

	// ── Step 5: configure ─────────────────────────────────────────────────────
	var configureArgs []string
	if r.Hooks.ConfigureArgs != nil {
		configureArgs = r.Hooks.ConfigureArgs(srcDir, prefix)
	} else {
		configureArgs = []string{fmt.Sprintf("--prefix=%s", prefix)}
	}

	var configureEnv map[string]string
	if r.Hooks.ConfigureEnv != nil {
		configureEnv = r.Hooks.ConfigureEnv()
	}

	if configureEnv != nil {
		if err := run.RunInDirWithEnv(srcDir, configureEnv, "./configure", configureArgs...); err != nil {
			return fmt.Errorf("%s: configure: %w", name, err)
		}
	} else {
		if err := run.RunInDir(srcDir, "./configure", configureArgs...); err != nil {
			return fmt.Errorf("%s: configure: %w", name, err)
		}
	}

	// ── Step 6: make ──────────────────────────────────────────────────────────
	makeArgs := []string{}
	if r.Hooks.MakeArgs != nil {
		makeArgs = r.Hooks.MakeArgs()
	}

	if configureEnv != nil {
		if err := run.RunInDirWithEnv(srcDir, configureEnv, "make", makeArgs...); err != nil {
			return fmt.Errorf("%s: make: %w", name, err)
		}
	} else {
		if err := run.RunInDir(srcDir, "make", makeArgs...); err != nil {
			return fmt.Errorf("%s: make: %w", name, err)
		}
	}

	// ── Step 7: make install ──────────────────────────────────────────────────
	var installEnv map[string]string
	if r.Hooks.InstallEnv != nil {
		installEnv = r.Hooks.InstallEnv(prefix)
	} else if configureEnv != nil {
		installEnv = configureEnv
	}

	if installEnv != nil {
		if err := run.RunInDirWithEnv(srcDir, installEnv, "make", "install"); err != nil {
			return fmt.Errorf("%s: make install: %w", name, err)
		}
	} else {
		if err := run.RunInDir(srcDir, "make", "install"); err != nil {
			return fmt.Errorf("%s: make install: %w", name, err)
		}
	}

	// ── Step 8: after-install hook ────────────────────────────────────────────
	if r.Hooks.AfterInstall != nil {
		if err := r.Hooks.AfterInstall(ctx, prefix, run); err != nil {
			return fmt.Errorf("%s: after_install: %w", name, err)
		}
	}

	// ── Step 9: pack artifact ─────────────────────────────────────────────────
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("%s-%s-linux-x64.tgz", name, version))

	var packDirs []string
	if r.Hooks.PackDirs != nil {
		packDirs = r.Hooks.PackDirs()
	} else {
		packDirs = []string{"."}
	}

	tarArgs := append([]string{"czf", artifactPath}, packDirs...)
	if err := run.RunInDir(prefix, "tar", tarArgs...); err != nil {
		return fmt.Errorf("%s: packing artifact: %w", name, err)
	}

	// ── Step 10: after-pack hook (e.g. StripTopLevelDir) ─────────────────────
	if r.Hooks.AfterPack != nil {
		if err := r.Hooks.AfterPack(artifactPath); err != nil {
			return fmt.Errorf("%s: after_pack: %w", name, err)
		}
	}

	return nil
}

// mustCwd returns the current working directory, panicking on error.
func mustCwd() string {
	cwd, err := filepath.Abs(".")
	if err != nil {
		panic(fmt.Sprintf("autoconf: getting cwd: %v", err))
	}
	return cwd
}
