package recipe

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// BowerRecipe downloads an npm tarball directly — simplest possible recipe.
type BowerRecipe struct {
	Fetcher fetch.Fetcher
}

func (b *BowerRecipe) Name() string { return "bower" }
func (b *BowerRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}
func (b *BowerRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName: "bower",
		Meta:    ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher: b.Fetcher,
	}).Build(ctx, s, src, r, out)
}

// YarnRecipe downloads yarn, strips 'v' prefix from version, strips top-level dir.
type YarnRecipe struct {
	Fetcher fetch.Fetcher
}

func (y *YarnRecipe) Name() string { return "yarn" }
func (y *YarnRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}
func (y *YarnRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName:            "yarn",
		Meta:               ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher:            y.Fetcher,
		StripTopLevelDir:   true,
		StripVersionPrefix: "v",
	}).Build(ctx, s, src, r, out)
}

// SetuptoolsRecipe downloads setuptools, strips top-level dir (handles both tar.gz and zip).
type SetuptoolsRecipe struct {
	Fetcher fetch.Fetcher
}

func (s *SetuptoolsRecipe) Name() string { return "setuptools" }
func (s *SetuptoolsRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}
func (s *SetuptoolsRecipe) Build(ctx context.Context, stk *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName:          "setuptools",
		Meta:             ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher:          s.Fetcher,
		StripTopLevelDir: true,
		// Setuptools infers the destination filename from the URL's last path segment.
		DestFilename: func(_, url string) string {
			parts := strings.Split(url, "/")
			return parts[len(parts)-1]
		},
	}).Build(ctx, stk, src, r, out)
}

// RubygemsRecipe downloads rubygems and strips top-level dir.
type RubygemsRecipe struct {
	Fetcher fetch.Fetcher
}

func (rg *RubygemsRecipe) Name() string { return "rubygems" }
func (rg *RubygemsRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}
func (rg *RubygemsRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName:          "rubygems",
		Meta:             ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher:          rg.Fetcher,
		StripTopLevelDir: true,
	}).Build(ctx, s, src, r, out)
}

// MinicondaRecipe is a URL passthrough — no file produced, just sets outData.
type MinicondaRecipe struct {
	Fetcher fetch.Fetcher
}

func (m *MinicondaRecipe) Name() string { return "miniconda3-py39" }
func (m *MinicondaRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"}
}

func (m *MinicondaRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, r runner.Runner, outData *output.OutData) error {
	// Miniconda is special: no file produced. We just verify the URL body
	// and set outData.URL + outData.SHA256 directly.
	body, err := m.Fetcher.ReadBody(ctx, src.URL)
	if err != nil {
		return fmt.Errorf("reading miniconda URL: %w", err)
	}

	// Compute SHA256 of the body.
	sha256 := computeSHA256(body)

	outData.URL = src.URL
	outData.SHA256 = sha256

	return nil
}
