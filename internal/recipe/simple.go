package recipe

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/archive"
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

// pnpmWrapper is injected as bin/pnpm. The release archive lays the native
// binary out flat at the archive root, so this wrapper supplies the bin/<dep>
// layout consumers already expect from yarn.
//
// $0 is resolved with readlink -f because buildpacks symlink the artifact's
// bin/ entries into their own bin directory — without it $0's dirname points at
// the symlink's directory, where the binary does not exist. This mirrors what
// upstream yarn's own bin/yarn wrapper does. No interpreter is involved: the
// target is a native executable.
const pnpmWrapper = `#!/bin/sh
basedir=$(dirname "$(readlink -f "$0" 2>/dev/null || echo "$0")")
exec "$basedir/../pnpm" "$@"
`

// PnpmRecipe downloads pnpm's self-contained linux-x64 release archive and
// injects a bin/pnpm wrapper.
//
// The npm registry tarball is deliberately not used. As of pnpm 12 that package
// is a ~1 MB stub: its preinstall script downloads a platform-native binary, and
// the bin/pnpm.mjs Corepack shim fetches the same binary on first run. A
// buildpack cannot depend on either in an offline or air-gapped deployment. The
// release archive ships the native binary outright, so the artifact is
// self-contained and needs no network access at staging time.
//
// The archive is linux-x64 and glibc-linked, which covers every current
// cflinuxfs stack; musl and other architectures are published separately and are
// not built here.
type PnpmRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PnpmRecipe) Name() string { return "pnpm" }
func (p *PnpmRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}
func (p *PnpmRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&RepackRecipe{
		DepName: "pnpm",
		Meta:    ArtifactMeta{OS: "linux", Arch: "x64"},
		Fetcher: p.Fetcher,
		// Release tags carry a "v" prefix; the archive itself is already flat,
		// so there is no top-level directory to strip.
		StripVersionPrefix: "v",
		AfterRepack: func(dest string) error {
			if err := archive.InjectFileWithMode(dest, "bin/pnpm", []byte(pnpmWrapper), 0755); err != nil {
				return fmt.Errorf("pnpm: injecting bin/pnpm wrapper: %w", err)
			}
			return nil
		},
	}).Build(ctx, s, src, r, out)
}

// PyPISourceRecipe downloads a PyPI source tarball and strips its top-level
// directory. It covers any dep published as a plain sdist on PyPI (e.g.
// setuptools, flit-core) where no compilation step is required.
//
// We intentionally do NOT use the raw PyPI filename (e.g. flit_core-3.12.0.tar.gz)
// as the destination, because PyPI normalises package names with underscores while
// our dep names use hyphens (e.g. "flit-core"). Using the dep name directly ensures
// findIntermediateArtifact can locate the file by its dep-name prefix.
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
		// No DestFilename override: RepackRecipe's default produces
		// "<depname>-<version><ext>" (e.g. flit-core-3.12.0.tar.gz),
		// which findIntermediateArtifact can locate by dep-name prefix.
	}).Build(ctx, stk, src, r, out)
}

// RubygemsRecipe downloads rubygems and strips top-level dir.
type RubygemsRecipe struct {
	Fetcher fetch.Fetcher
}

func (rg *RubygemsRecipe) Name() string { return "rubygems" }
func (rg *RubygemsRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"}
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
