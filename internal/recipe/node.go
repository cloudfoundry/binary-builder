package recipe

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/compiler"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/portile"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// NodeRecipe builds Node.js, matching the Ruby NodeRecipe exactly:
//   - configure with --prefix=/ --openssl-use-def-ca-store
//   - install with DESTDIR=/tmp/node-v{version}-linux-x64 PORTABLE=1
//   - copy LICENSE from work_path into dest_dir
//   - pack dest_dir as top-level dir, then strip top-level from the artifact
//
// The tarball node-v{version}.tar.gz extracts to node-v{version}/ (not node-{version}/),
// so ExtractedDirName is set to "node-v{version}" to match.
type NodeRecipe struct {
	Fetcher fetch.Fetcher
}

func (n *NodeRecipe) Name() string { return "node" }
func (n *NodeRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (n *NodeRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	// Step 1: Install python3/pip3 build tools (node configure requires python3).
	if err := setupPythonAndPip(run); err != nil {
		return fmt.Errorf("node: setup python/pip: %w", err)
	}

	// Step 2: Set up GCC (stack-driven; cflinuxfs5 skips the PPA).
	a := apt.New(run)
	gcc := compiler.NewGCC(s.Compilers.GCC, a, run)
	if err := gcc.Setup(ctx); err != nil {
		return fmt.Errorf("node: GCC setup: %w", err)
	}

	// Step 3: Strip `v` prefix from version (e.g. "v22.14.0" → "22.14.0").
	version := strings.TrimPrefix(src.Version, "v")

	// Ruby recipe's dest_dir: /tmp/node-v{version}-linux-x64
	destDir := fmt.Sprintf("/tmp/node-v%s-linux-x64", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("node-%s-linux-x64.tgz", version))

	// Step 4: Install optional node build packages from stack config.
	if pkgs, ok := s.AptPackages["node_build"]; ok && len(pkgs) > 0 {
		if err := a.Install(ctx, pkgs...); err != nil {
			return fmt.Errorf("node: apt install node_build: %w", err)
		}
	}

	// Step 5: Build via portile.
	// Ruby recipe: --prefix=/ --openssl-use-def-ca-store
	// Install: make install DESTDIR={destDir} PORTABLE=1
	// Tarball node-v{version}.tar.gz extracts to node-v{version}/ directory.
	p := &portile.Portile{
		Name:             "node",
		Version:          version,
		URL:              src.URL,
		Checksum:         src.PrimaryChecksum(),
		Prefix:           "/",
		Options:          []string{"--openssl-use-def-ca-store"},
		ExtractedDirName: fmt.Sprintf("node-v%s", version),
		InstallArgs:      []string{fmt.Sprintf("DESTDIR=%s", destDir), "PORTABLE=1"},
		Runner:           run,
		Fetcher:          n.Fetcher,
	}

	if err := p.Cook(ctx); err != nil {
		return fmt.Errorf("node: portile cook: %w", err)
	}

	// Step 6: Copy LICENSE into dest_dir (mirrors Ruby recipe's setup_tar).
	// The portile work_path is TmpPath()/port.
	licenseSource := fmt.Sprintf("%s/port/LICENSE", p.TmpPath())
	if err := run.Run("cp", licenseSource, destDir); err != nil {
		return fmt.Errorf("node: copying LICENSE: %w", err)
	}

	// Step 7: Pack dest_dir as the top-level directory, then strip it.
	// Ruby: cp -r destDir tmpdir/ → tar czf artifact -C tmpdir → strip_top_level_directory_from_tar
	// We recreate this by packing destDir's parent with the dirname, then stripping.
	destParent := filepath.Dir(destDir)
	destBase := filepath.Base(destDir)
	if err := run.Run("tar", "czf", artifactPath, "-C", destParent, destBase); err != nil {
		return fmt.Errorf("node: packing artifact: %w", err)
	}

	return archive.StripTopLevelDir(artifactPath)
}
