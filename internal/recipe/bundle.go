package recipe

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// BundleRecipe downloads multiple pip packages and bundles them into a tarball.
// It covers pip and pipenv, which both follow the pattern:
//
//  1. setupPythonAndPip (fixed first step — both always need it)
//  2. mkdir -p tmpDir
//  3. pip3 download main package into tmpDir
//  4. [optional] ExtraSteps — e.g. pip's source-tarball strip + extract
//  5. pip3 download each ExtraDeps into tmpDir
//  6. tar zcvf OutputPath(version) from tmpDir
type BundleRecipe struct {
	DepName string
	Meta    ArtifactMeta
	Fetcher fetch.Fetcher
	// MainPackage returns the pip package specifier for the main dep, e.g. "pip==24.0".
	MainPackage func(version string) string
	// DownloadArgs are extra args passed to the pip3 download command for the main package.
	// e.g. ["--no-binary", ":all:"] or ["--no-cache-dir", "--no-binary", ":all:"]
	DownloadArgs []string
	// ExtraDeps is the list of additional packages to bundle (each gets its own pip3 download).
	ExtraDeps []string
	// ExtraSteps runs inside tmpDir after the main package download and before ExtraDeps.
	// Used for pip's source-tarball-strip-and-extract step.
	// May be nil.
	ExtraSteps func(ctx context.Context, tmpDir string, src *source.Input, f fetch.Fetcher, r runner.Runner) error
	// OutputPath returns the artifact path from version.
	// Default: "/tmp/<depname>-<version>.tgz"
	OutputPath func(version string) string
}

func (b *BundleRecipe) Name() string           { return b.DepName }
func (b *BundleRecipe) Artifact() ArtifactMeta { return b.Meta }

func (b *BundleRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	name := b.DepName
	version := src.Version

	// Fixed first step: both pip and pipenv always need python + pip setup.
	if err := setupPythonAndPip(ctx, s, r); err != nil {
		return fmt.Errorf("%s: setup python: %w", name, err)
	}

	outputPath := fmt.Sprintf("/tmp/%s-%s.tgz", name, version)
	if b.OutputPath != nil {
		outputPath = b.OutputPath(version)
	}

	tmpDir := fmt.Sprintf("/tmp/%s-build-%s", name, version)
	if err := r.Run("mkdir", "-p", tmpDir); err != nil {
		return fmt.Errorf("%s: mkdir: %w", name, err)
	}

	// Download the main package.
	mainArgs := append(append([]string{"download"}, b.DownloadArgs...), b.MainPackage(version))
	if err := r.RunInDir(tmpDir, "/usr/bin/pip3", mainArgs...); err != nil {
		return fmt.Errorf("%s: pip3 download %s: %w", name, b.MainPackage(version), err)
	}

	// Run recipe-specific extra steps (e.g. pip's source tarball strip + extract).
	if b.ExtraSteps != nil {
		if err := b.ExtraSteps(ctx, tmpDir, src, b.Fetcher, r); err != nil {
			return fmt.Errorf("%s: extra steps: %w", name, err)
		}
	}

	// Download each extra dependency.
	// Do NOT pass --no-binary :all: here: many pure-Python packages publish sdists
	// with name="unknown" in metadata, which pip 22.x (cflinuxfs4) rejects. Wheels
	// are fine for these deps since they are pure Python.
	for _, dep := range b.ExtraDeps {
		if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", dep); err != nil {
			return fmt.Errorf("%s: pip3 download %s: %w", name, dep, err)
		}
	}

	// Bundle everything.
	if err := r.RunInDir(tmpDir, "tar", "zcvf", outputPath, "."); err != nil {
		return fmt.Errorf("%s: creating tarball: %w", name, err)
	}

	return nil
}
