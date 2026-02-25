package recipe

import (
	"path/filepath"
	"context"
	"fmt"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// LibunwindRecipe builds libunwind from a pre-downloaded source tarball.
// The Concourse github-releases depwatcher has already placed the tarball in source/.
// Only the include/ and lib/ directories are packed into the artifact.
type LibunwindRecipe struct{}

func (l *LibunwindRecipe) Name() string { return "libunwind" }
func (l *LibunwindRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (l *LibunwindRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version

	// Derive the directory name from the URL filename by stripping .tar.gz.
	parts := strings.Split(src.URL, "/")
	filename := parts[len(parts)-1]
	dirName := strings.TrimSuffix(strings.TrimSuffix(filename, ".tar.gz"), ".tgz")

	srcTarball := fmt.Sprintf("source/%s", filename)
	srcDir := fmt.Sprintf("/tmp/%s", dirName)
	builtPath := fmt.Sprintf("/tmp/libunwind-built-%s", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("%s.tgz", dirName))

	// Extract the pre-downloaded tarball.
	if err := run.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
		return fmt.Errorf("libunwind: extracting source: %w", err)
	}

	if err := run.Run("mkdir", "-p", builtPath); err != nil {
		return err
	}

	// Configure, make, install.
	if err := run.RunInDir(srcDir, "./configure", fmt.Sprintf("--prefix=%s", builtPath)); err != nil {
		return fmt.Errorf("libunwind: configure: %w", err)
	}
	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("libunwind: make: %w", err)
	}
	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("libunwind: make install: %w", err)
	}

	// Pack only include/ and lib/ directories.
	if err := run.RunInDir(builtPath, "tar", "czf", artifactPath, "include", "lib"); err != nil {
		return fmt.Errorf("libunwind: packing artifact: %w", err)
	}

	return nil
}
