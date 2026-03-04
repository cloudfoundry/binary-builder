package recipe

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/apt"
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

func (l *LibunwindRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	// Install autotools needed to regenerate ./configure from configure.ac.
	// GitHub source archives only contain autotools sources, not the generated script.
	a := apt.New(run)
	if err := a.Install(ctx, s.AptPackages["libunwind_build"]...); err != nil {
		return fmt.Errorf("libunwind: apt install libunwind_build: %w", err)
	}

	version := src.Version

	// Derive the directory name from the URL filename by stripping .tar.gz.
	parts := strings.Split(src.URL, "/")
	filename := parts[len(parts)-1]
	tag := strings.TrimSuffix(strings.TrimSuffix(filename, ".tar.gz"), ".tgz")
	// Two URL styles are in use:
	//   refs/tags/v1.6.2.tar.gz   → tag="v1.6.2"         → extracts to libunwind-1.6.2/
	//   libunwind-1.6.2.tar.gz    → tag="libunwind-1.6.2" → extracts to libunwind-1.6.2/
	// Avoid double-prefixing when the filename already starts with "libunwind-".
	var dirName string
	if strings.HasPrefix(tag, "libunwind-") {
		dirName = tag
	} else {
		dirName = "libunwind-" + strings.TrimPrefix(tag, "v")
	}

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

	// Regenerate ./configure from configure.ac (GitHub archives ship only autotools sources).
	if err := run.RunInDir(srcDir, "autoreconf", "-i"); err != nil {
		return fmt.Errorf("libunwind: autoreconf: %w", err)
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
