package recipe

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// GoRecipe builds Go from source using a pre-downloaded bootstrap binary and make.bash.
// The source tarball extracts into a `go/` subdirectory; we pack that directory then
// strip the top-level `go/` prefix so the final artifact has `./`-prefixed paths,
// matching what Ruby's builder.rb build_go produces via strip_top_level_directory_from_tar.
type GoRecipe struct {
	Fetcher fetch.Fetcher
}

func (g *GoRecipe) Name() string { return "go" }
func (g *GoRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (g *GoRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	// Strip the `go` prefix from version (e.g. "go1.24.2" → "1.24.2").
	version := strings.TrimPrefix(src.Version, "go")

	srcTarball := fmt.Sprintf("/tmp/go%s.src.tar.gz", version)
	bootstrapDir := fmt.Sprintf("/tmp/go-bootstrap-%s", version)
	srcDir := fmt.Sprintf("/tmp/go-src-%s", version)
	// Use a dash between name and version so findIntermediateArtifact can locate this file.
	// Pattern: go-1.22.0.linux-amd64.tar.gz  (matches glob "go-1.22.0*.tar.gz")
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("go-%s.linux-amd64.tar.gz", version))

	// Download Go source tarball.
	if err := g.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("go: downloading source: %w", err)
	}

	// Extract source. The Go source tarball extracts into a `go/` subdirectory.
	// We do NOT use --strip-components so srcDir/go/ contains bin/, src/, etc.
	if err := run.Run("mkdir", "-p", srcDir); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", srcDir); err != nil {
		return fmt.Errorf("go: extracting source: %w", err)
	}

	// Download and extract the bootstrap Go binary.
	// Match the Ruby go.rb recipe which uses go1.24.2 as bootstrap.
	bootstrapURL := "https://go.dev/dl/go1.24.2.linux-amd64.tar.gz"
	bootstrapTarball := "/tmp/go-bootstrap.tar.gz"

	if err := run.Run("mkdir", "-p", bootstrapDir); err != nil {
		return err
	}
	if err := run.Run("wget", "-q", "-O", bootstrapTarball, bootstrapURL); err != nil {
		return fmt.Errorf("go: downloading bootstrap: %w", err)
	}
	if err := run.Run("tar", "xzf", bootstrapTarball, "-C", bootstrapDir, "--strip-components=1"); err != nil {
		return fmt.Errorf("go: extracting bootstrap: %w", err)
	}

	// Run make.bash to compile Go from source.
	// make.bash must be run from within $GOROOT/src (it infers GOROOT from its own location).
	srcGoSrc := fmt.Sprintf("%s/go/src", srcDir)
	if err := run.RunInDirWithEnv(
		srcGoSrc,
		map[string]string{
			"GOROOT_BOOTSTRAP": bootstrapDir,
			"GOROOT_FINAL":     "/usr/local/go",
			"GOTOOLCHAIN":      "local",
		},
		"bash", "./make.bash",
	); err != nil {
		return fmt.Errorf("go: make.bash: %w", err)
	}

	// Pack the compiled Go distribution.
	// srcDir/go/ contains bin/, src/, pkg/, etc. — pack the `go` directory itself
	// so the artifact has a top-level `go/` entry, matching the Ruby recipe layout.
	if err := run.Run("tar", "czf", artifactPath, "-C", srcDir, "go"); err != nil {
		return fmt.Errorf("go: packing artifact: %w", err)
	}

	// Strip the top-level `go/` directory from the artifact, matching what Ruby's
	// builder.rb build_go does via Archive.strip_top_level_directory_from_tar.
	// This produces "./" prefixed paths (./bin/go, ./src/..., etc.) in the tarball.
	if err := archive.StripTopLevelDir(artifactPath); err != nil {
		return fmt.Errorf("go: stripping top-level dir: %w", err)
	}

	return nil
}
