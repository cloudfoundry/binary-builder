package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// LibgdiplusRecipe builds libgdiplus from source via git clone + autogen + make.
type LibgdiplusRecipe struct{}

func (l *LibgdiplusRecipe) Name() string { return "libgdiplus" }
func (l *LibgdiplusRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (l *LibgdiplusRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	a := apt.New(run)

	// Install libgdiplus build dependencies from stack config.
	if err := a.Install(ctx, s.AptPackages["libgdiplus_build"]...); err != nil {
		return fmt.Errorf("libgdiplus: apt install libgdiplus_build: %w", err)
	}

	version := src.Version
	repo := src.Repo // e.g. "mono/libgdiplus"
	cloneDir := fmt.Sprintf("libgdiplus-%s", version)
	builtPath := fmt.Sprintf("/tmp/libgdiplus-built-%s", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("libgdiplus-%s.tgz", version))

	// Clone the repository at the given version tag/branch.
	repoURL := fmt.Sprintf("https://github.com/%s", repo)
	if err := run.Run("git", "clone", "--single-branch", "--branch", version, repoURL, cloneDir); err != nil {
		return fmt.Errorf("libgdiplus: git clone: %w", err)
	}

	if err := run.Run("mkdir", "-p", builtPath); err != nil {
		return err
	}

	// Set warning-suppression flags to avoid -Werror failures.
	buildEnv := map[string]string{
		"CFLAGS":   "-g -Wno-maybe-uninitialized",
		"CXXFLAGS": "-g -Wno-maybe-uninitialized",
	}

	if err := run.RunWithEnv(buildEnv, "sh", "-c",
		fmt.Sprintf("cd %s && ./autogen.sh --prefix=%s", cloneDir, builtPath)); err != nil {
		return fmt.Errorf("libgdiplus: autogen: %w", err)
	}

	if err := run.RunInDirWithEnv(cloneDir, buildEnv, "make"); err != nil {
		return fmt.Errorf("libgdiplus: make: %w", err)
	}
	if err := run.RunInDirWithEnv(cloneDir, buildEnv, "make", "install"); err != nil {
		return fmt.Errorf("libgdiplus: make install: %w", err)
	}

	// Pack only lib/ directory.
	if err := run.RunInDir(builtPath, "tar", "czf", artifactPath, "lib"); err != nil {
		return fmt.Errorf("libgdiplus: packing artifact: %w", err)
	}

	return nil
}
