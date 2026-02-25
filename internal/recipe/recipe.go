// Package recipe defines the Recipe interface and the global recipe registry.
// Each dependency type implements Recipe. The registry maps dep names to builders.
package recipe

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// Recipe is the interface every dependency builder must implement.
type Recipe interface {
	// Name returns the dependency name (e.g. "ruby", "php").
	Name() string

	// Build performs the full build: download, configure, compile, install, archive.
	// It populates outData with artifact URL, SHA256, and any sub-dependencies.
	Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, outData *output.OutData) error

	// Artifact returns the artifact metadata for this recipe.
	Artifact() ArtifactMeta
}

// ArtifactMeta describes the artifact naming for a recipe.
type ArtifactMeta struct {
	OS    string // "linux" or "windows"
	Arch  string // "x64", "noarch", "x86-64"
	Stack string // stack name, "any-stack", or "" (use build stack)
}

// Registry maps dependency names to recipe constructors.
type Registry struct {
	recipes map[string]Recipe
}

// NewRegistry creates an empty recipe registry.
func NewRegistry() *Registry {
	return &Registry{recipes: make(map[string]Recipe)}
}

// Register adds a recipe to the registry.
func (r *Registry) Register(recipe Recipe) {
	r.recipes[recipe.Name()] = recipe
}

// Get returns the recipe for the given dependency name.
func (r *Registry) Get(name string) (Recipe, error) {
	recipe, ok := r.recipes[name]
	if !ok {
		return nil, fmt.Errorf("no recipe registered for %q", name)
	}
	return recipe, nil
}

// Names returns all registered recipe names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.recipes))
	for name := range r.recipes {
		names = append(names, name)
	}
	return names
}
