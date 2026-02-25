package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/portile"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// RubyRecipe builds Ruby via portile (configure/make/install) and strips
// incorrect_words.yaml from the resulting tarball.
type RubyRecipe struct {
	Fetcher fetch.Fetcher
}

func (r *RubyRecipe) Name() string { return "ruby" }
func (r *RubyRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (r *RubyRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	a := apt.New(run)

	// Install ruby build dependencies from stack config.
	if err := a.Install(ctx, s.AptPackages["ruby_build"]...); err != nil {
		return fmt.Errorf("ruby: apt install ruby_build: %w", err)
	}

	builtPath := fmt.Sprintf("/app/vendor/ruby-%s", src.Version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("ruby-%s-linux-x64.tgz", src.Version))

	p := &portile.Portile{
		Name:     "ruby",
		Version:  src.Version,
		URL:      src.URL,
		Checksum: src.PrimaryChecksum(),
		Prefix:   builtPath,
		Options: []string{
			"--enable-load-relative",
			"--disable-install-doc",
			"--without-gmp",
		},
		Runner:  run,
		Fetcher: r.Fetcher,
	}

	if err := p.Cook(ctx); err != nil {
		return fmt.Errorf("ruby: portile cook: %w", err)
	}

	// Pack the installed tree flat (no top-level dir), matching Ruby builder's
	// ArchiveRecipe#compress! which copies archive_files into tmpdir/ directly
	// and tars from there, so archive root contains bin/, lib/, etc.
	if err := run.Run("tar", "czf", artifactPath, "-C", builtPath, "."); err != nil {
		return fmt.Errorf("ruby: packing artifact: %w", err)
	}

	// Inject sources.yml into the artifact tarball at the archive root, matching
	// Ruby's ArchiveRecipe#compress! which writes YAMLPresenter output into the
	// tmpdir before running tar (alongside the archive_files).
	// src.SHA256 is the sha256 of the downloaded source tarball, matching what
	// YAMLPresenter computes via Digest::SHA256.file(local_path).hexdigest.
	sourcesContent := buildSourcesYAML([]SourceEntry{{URL: src.URL, SHA256: src.SHA256}})
	if err := archive.InjectFile(artifactPath, "sources.yml", sourcesContent); err != nil {
		return fmt.Errorf("ruby: injecting sources.yml: %w", err)
	}

	// Remove incorrect_words.yaml from the tarball and any nested jars.
	if err := archive.StripIncorrectWordsYAML(artifactPath); err != nil {
		return fmt.Errorf("ruby: stripping incorrect_words.yaml: %w", err)
	}

	return nil
}
