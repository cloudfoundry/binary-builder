package recipe

import (
	"context"
	"fmt"
	"net/url"
	"path"
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

// PyPISourceRecipe downloads a PyPI source tarball and strips its top-level
// directory. It covers any dep published as a plain sdist on PyPI (e.g.
// setuptools, flit-core) where the artifact filename is the last path segment
// of the download URL and no compilation step is required.
type PyPISourceRecipe struct {
	DepName string
	Fetcher fetch.Fetcher
}

func (p *PyPISourceRecipe) Name() string { return p.DepName }
func (p *PyPISourceRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}
func (p *PyPISourceRecipe) Build(ctx context.Context, stk *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName:          p.DepName,
		Meta:             ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher:          p.Fetcher,
		StripTopLevelDir: true,
		// PyPI sdist URLs end with the canonical filename (e.g. setuptools-69.0.3.tar.gz).
		// Use url.Parse + path.Base to strip any query string or fragment before
		// using the last path segment as the local filename.
		DestFilename: func(_, rawURL string) string {
			if u, err := url.Parse(rawURL); err == nil {
				return path.Base(u.Path)
			}
			// Fallback: should not happen for well-formed URLs.
			parts := strings.Split(rawURL, "/")
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
