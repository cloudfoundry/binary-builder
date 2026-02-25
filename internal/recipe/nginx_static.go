package recipe

import (
	"context"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// NginxStaticRecipe builds nginx with PIE flags and a minimal module set.
// It shares most logic with NginxRecipe via buildNginxVariant.
type NginxStaticRecipe struct {
	Fetcher fetch.Fetcher
}

func (n *NginxStaticRecipe) Name() string { return "nginx-static" }
func (n *NginxStaticRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (n *NginxStaticRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	return buildNginxVariant(ctx, src, run, n.Fetcher, true)
}
