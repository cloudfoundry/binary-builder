package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// DotnetSDKRecipe builds dotnet-sdk: download, prune ./shared/*, inject RuntimeVersion.txt, xz compress.
type DotnetSDKRecipe struct{}

func (d *DotnetSDKRecipe) Name() string { return "dotnet-sdk" }
func (d *DotnetSDKRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (d *DotnetSDKRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	return pruneDotnetFiles(r, src, []string{"./shared/*"}, true)
}

// DotnetRuntimeRecipe builds dotnet-runtime: download, prune ./dotnet, xz compress.
type DotnetRuntimeRecipe struct{}

func (d *DotnetRuntimeRecipe) Name() string { return "dotnet-runtime" }
func (d *DotnetRuntimeRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (d *DotnetRuntimeRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	return pruneDotnetFiles(r, src, []string{"./dotnet"}, false)
}

// DotnetAspnetcoreRecipe builds dotnet-aspnetcore: download, prune ./dotnet + ./shared/Microsoft.NETCore.App, xz compress.
type DotnetAspnetcoreRecipe struct{}

func (d *DotnetAspnetcoreRecipe) Name() string { return "dotnet-aspnetcore" }
func (d *DotnetAspnetcoreRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (d *DotnetAspnetcoreRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	return pruneDotnetFiles(r, src, []string{"./dotnet", "./shared/Microsoft.NETCore.App"}, false)
}

// pruneDotnetFiles extracts a dotnet tarball excluding specified paths,
// optionally writes RuntimeVersion.txt, and re-compresses with xz.
//
// The dotnet source tarball is pre-downloaded by Concourse into source/*.tar.gz.
// We use filepath.Glob to resolve the actual path before passing it to tar,
// since the runner does not invoke a shell (no glob expansion).
//
// The output artifact is written to the CWD using dash-separated naming
// (e.g. dotnet-runtime-8.0.21-linux-x64.tar.xz) so that findIntermediateArtifact
// can locate it via the standard glob patterns.
func pruneDotnetFiles(r runner.Runner, src *source.Input, excludes []string, writeRuntime bool) error {
	adjustedFile := filepath.Join(mustCwd(), fmt.Sprintf("%s-%s-linux-x64.tar.xz", src.Name, src.Version))
	tmpDir := fmt.Sprintf("/tmp/dotnet-prune-%s-%s", src.Name, src.Version)

	if err := r.Run("mkdir", "-p", tmpDir); err != nil {
		return err
	}

	// Resolve source/*.tar.gz via glob — the runner does not use a shell so
	// glob patterns are NOT expanded by the OS.
	matches, err := filepath.Glob("source/*.tar.gz")
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("dotnet: no source tarball found matching source/*.tar.gz")
	}
	sourceTarball := matches[0]

	// Build exclude args.
	extractArgs := []string{"-xf", sourceTarball, "-C", tmpDir}
	for _, exc := range excludes {
		extractArgs = append(extractArgs, fmt.Sprintf("--exclude=%s", exc))
	}

	if err := r.Run("tar", extractArgs...); err != nil {
		return fmt.Errorf("extracting dotnet: %w", err)
	}

	if writeRuntime {
		// Extract runtime version from the original archive.
		// List entries under ./shared/Microsoft.NETCore.App/ and take the last directory,
		// mirroring the Ruby recipe's write_runtime_version_file.
		runtimeOutput, err := r.Output("tar", "tf", sourceTarball, "./shared/Microsoft.NETCore.App/")
		if err != nil {
			return fmt.Errorf("listing runtime version: %w", err)
		}

		// Parse output: keep only directory entries (ending with '/'), take the last one.
		lines := strings.Split(strings.TrimSpace(runtimeOutput), "\n")
		var lastDir string
		for _, line := range lines {
			if strings.HasSuffix(line, "/") {
				lastDir = line
			}
		}
		if lastDir == "" {
			return fmt.Errorf("dotnet: no directory found under ./shared/Microsoft.NETCore.App/")
		}
		runtimeVersion := filepath.Base(strings.TrimSuffix(lastDir, "/"))

		runtimeVersionFile := filepath.Join(tmpDir, "RuntimeVersion.txt")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("mkdir tmpDir for RuntimeVersion.txt: %w", err)
		}
		if err := os.WriteFile(runtimeVersionFile, []byte(runtimeVersion), 0644); err != nil {
			return fmt.Errorf("writing RuntimeVersion.txt: %w", err)
		}
	}

	// Re-compress with xz.
	if err := r.RunInDir(tmpDir, "tar", "-Jcf", adjustedFile, "."); err != nil {
		return fmt.Errorf("creating xz archive: %w", err)
	}

	return nil
}
