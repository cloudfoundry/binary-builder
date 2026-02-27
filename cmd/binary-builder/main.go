// Command binary-builder builds a single dependency for a given CF stack.
//
// Usage:
//
//	binary-builder build \
//	  --name ruby \
//	  --stack cflinuxfs5 \
//	  --source-file source/data.json \
//	  --stacks-dir binary-builder/stacks \
//	  --php-extensions-dir binary-builder/php_extensions \
//	  --artifacts-dir artifacts \
//	  --builds-dir builds-artifacts \
//	  --dep-metadata-dir dep-metadata \
//	  [--skip-commit]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/artifact"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/fileutil"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/recipe"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "binary-builder: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || os.Args[1] != "build" {
		return fmt.Errorf("usage: binary-builder build [flags]")
	}

	fs := flag.NewFlagSet("build", flag.ExitOnError)

	stackName := fs.String("stack", "", "Stack name (e.g. cflinuxfs4, cflinuxfs5) [required]")
	sourceFile := fs.String("source-file", "source/data.json", "Path to source/data.json")
	stacksDir := fs.String("stacks-dir", "binary-builder/stacks", "Directory containing stack YAML files")
	phpExtensionsDir := fs.String("php-extensions-dir", "binary-builder/php_extensions", "Directory containing PHP extension YAML files")
	artifactsDir := fs.String("artifacts-dir", "artifacts", "Output directory for built artifacts")
	buildsDir := fs.String("builds-dir", "builds-artifacts", "Output directory for builds-artifacts JSON")
	depMetadataDir := fs.String("dep-metadata-dir", "dep-metadata", "Output directory for dep-metadata JSON")
	skipCommit := fs.Bool("skip-commit", false, "Skip git commit of builds-artifacts JSON")

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	if *stackName == "" {
		return fmt.Errorf("--stack is required")
	}

	// Load source input.
	src, err := source.FromFile(*sourceFile)
	if err != nil {
		return fmt.Errorf("loading source file: %w", err)
	}

	// Load stack config.
	s, err := stack.Load(*stacksDir, *stackName)
	if err != nil {
		return fmt.Errorf("loading stack %q: %w", *stackName, err)
	}

	// Build recipe registry.
	reg := buildRegistry(*phpExtensionsDir)

	// Look up the recipe.
	rec, err := reg.Get(src.Name)
	if err != nil {
		return fmt.Errorf("no recipe for %q — registered: %v", src.Name, reg.Names())
	}

	// Ensure output directories exist.
	for _, dir := range []string{*artifactsDir, *depMetadataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Prepare output data.
	outData := output.NewOutData(src)

	// Run the build.
	ctx := context.Background()
	run := &runner.RealRunner{}

	fmt.Printf("[binary-builder] building %s %s for %s\n", src.Name, src.Version, *stackName)

	if err := rec.Build(ctx, s, src, run, outData); err != nil {
		return fmt.Errorf("building %s: %w", src.Name, err)
	}

	// Handle artifact output. Miniconda sets outData.URL directly (no file produced).
	// All other recipes produce a file in the working directory.
	if outData.URL == "" {
		if err := handleArtifact(src, rec, s, *artifactsDir, outData); err != nil {
			return err
		}
	}

	// Write builds-artifacts JSON and optionally commit.
	buildOut, err := output.NewBuildOutput(src.Name, run, *buildsDir)
	if err != nil {
		return fmt.Errorf("creating build output: %w", err)
	}

	buildsFilename := fmt.Sprintf("%s-%s-%s.json", src.Name, outData.Version, *stackName)
	if err := buildOut.AddOutput(buildsFilename, outData); err != nil {
		return fmt.Errorf("writing builds output: %w", err)
	}

	if !*skipCommit {
		commitMsg := fmt.Sprintf("Build %s %s [%s]", src.Name, outData.Version, *stackName)
		if err := buildOut.Commit(commitMsg); err != nil {
			return fmt.Errorf("committing builds output: %w", err)
		}
	}

	// Write dep-metadata JSON.
	depMeta := output.NewDepMetadataOutput(*depMetadataDir)
	if err := depMeta.WriteMetadata(outData.URL, outData); err != nil {
		return fmt.Errorf("writing dep-metadata: %w", err)
	}

	// Print final output data for debugging / Concourse put step consumption.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(outData)
}

// handleArtifact finds the built file in the working directory, computes its
// SHA256, moves it into artifactsDir with the canonical filename, and populates
// outData.URL and outData.SHA256.
func handleArtifact(src *source.Input, rec recipe.Recipe, s *stack.Stack, artifactsDir string, outData *output.OutData) error {
	meta := rec.Artifact()

	// Resolve the effective stack label for the artifact filename.
	stackLabel := meta.Stack
	if stackLabel == "" {
		stackLabel = s.Name
	}

	// Use ArtifactVersion for file lookup and naming if set; otherwise fall
	// back to Version. This allows recipes like jruby to keep the raw source
	// version in dep-metadata (Version) while using a richer artifact version
	// string (ArtifactVersion) for the filename, e.g. "9.4.14.0-ruby-3.1".
	artifactVersion := outData.ArtifactVersion
	if artifactVersion == "" {
		artifactVersion = outData.Version
	}

	// Find the intermediate artifact file. Recipes write to the working directory.
	// filepath.Glob does not support brace expansion, so we try each extension
	// separately. Preference order matches ArtifactOutput.ext in the Ruby code.
	intermediatePath, err := findIntermediateArtifact(src.Name, artifactVersion)
	if err != nil {
		return err
	}

	ext := artifact.ExtFromPath(intermediatePath)

	// Compute SHA256.
	sha256hex, err := artifact.SHA256File(intermediatePath)
	if err != nil {
		return fmt.Errorf("computing SHA256 of %s: %w", intermediatePath, err)
	}

	// Build canonical filename.
	a := artifact.Artifact{
		Name:    src.Name,
		Version: artifactVersion,
		OS:      meta.OS,
		Arch:    meta.Arch,
		Stack:   stackLabel,
	}
	finalFilename := a.Filename(sha256hex, ext)
	finalPath := filepath.Join(artifactsDir, finalFilename)

	// Move artifact into artifacts dir. Use cross-device-safe move (copy+delete)
	// because the workdir and artifactsDir may be on different filesystems.
	if err := fileutil.MoveFile(intermediatePath, finalPath); err != nil {
		return fmt.Errorf("moving artifact to %s: %w", finalPath, err)
	}

	outData.SHA256 = sha256hex
	outData.URL = a.S3URL(finalFilename)

	fmt.Printf("[binary-builder] artifact: %s\n", outData.URL)
	return nil
}

// buildRegistry constructs and populates the full recipe registry.
func buildRegistry(phpExtensionsDir string) *recipe.Registry {
	f := fetch.NewHTTPFetcher()
	reg := recipe.NewRegistry()

	// Compiled recipes.
	reg.Register(&recipe.RubyRecipe{Fetcher: f})
	reg.Register(&recipe.BundlerRecipe{Fetcher: f})
	reg.Register(&recipe.PythonRecipe{Fetcher: f})
	reg.Register(&recipe.NodeRecipe{Fetcher: f})
	reg.Register(&recipe.GoRecipe{Fetcher: f})
	reg.Register(&recipe.NginxRecipe{Fetcher: f})
	reg.Register(&recipe.NginxStaticRecipe{Fetcher: f})
	reg.Register(&recipe.OpenrestyRecipe{})
	reg.Register(&recipe.HTTPDRecipe{Fetcher: f})
	reg.Register(&recipe.JRubyRecipe{Fetcher: f})
	reg.Register(&recipe.RRecipe{Fetcher: f})
	reg.Register(&recipe.LibunwindRecipe{})
	reg.Register(&recipe.LibgdiplusRecipe{})
	reg.Register(&recipe.DepRecipe{Fetcher: f})
	reg.Register(&recipe.GlideRecipe{Fetcher: f})
	reg.Register(&recipe.GodepRecipe{Fetcher: f})
	reg.Register(&recipe.HWCRecipe{Fetcher: f})

	// PHP recipe.
	reg.Register(&recipe.PHPRecipe{Fetcher: f, ExtensionsDir: phpExtensionsDir})

	// Simple / repack recipes.
	reg.Register(&recipe.PipRecipe{Fetcher: f})
	reg.Register(&recipe.PipenvRecipe{Fetcher: f})
	reg.Register(&recipe.BowerRecipe{Fetcher: f})
	reg.Register(&recipe.YarnRecipe{Fetcher: f})
	reg.Register(&recipe.SetuptoolsRecipe{Fetcher: f})
	reg.Register(&recipe.RubygemsRecipe{Fetcher: f})
	reg.Register(&recipe.MinicondaRecipe{Fetcher: f})
	reg.Register(&recipe.DotnetSDKRecipe{})
	reg.Register(&recipe.DotnetRuntimeRecipe{})
	reg.Register(&recipe.DotnetAspnetcoreRecipe{})

	// Passthrough recipes.
	for _, r := range recipe.NewPassthroughRecipes(f) {
		reg.Register(r)
	}

	return reg
}

// findIntermediateArtifact searches the working directory (and os.TempDir as
// fallback) for the artifact file produced by a recipe. It tries common
// extensions in priority order. filepath.Glob does not support brace expansion,
// so we try each extension separately.
func findIntermediateArtifact(name, version string) (string, error) {
	// Extensions in priority order (matches ArtifactOutput.ext in Ruby).
	exts := []string{"tgz", "tar.gz", "zip", "tar.xz", "tar.bz2", "sh", "phar", "txt"}

	// Search dirs: CWD first, then os.TempDir. Recipes that cannot write to the
	// CWD (e.g. pip, pipenv, bower, yarn, rubygems, setuptools) write to TempDir.
	searchDirs := []string{".", os.TempDir()}

	for _, dir := range searchDirs {
		// Try name-version prefix first (most specific).
		for _, ext := range exts {
			pattern := filepath.Join(dir, fmt.Sprintf("%s-%s*.%s", name, version, ext))
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return "", fmt.Errorf("globbing %s: %w", pattern, err)
			}
			if len(matches) > 0 {
				return matches[0], nil
			}
		}

		// Fallback: just name prefix.
		for _, ext := range exts {
			pattern := filepath.Join(dir, fmt.Sprintf("%s-*.%s", name, ext))
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return "", fmt.Errorf("globbing %s: %w", pattern, err)
			}
			if len(matches) > 0 {
				return matches[0], nil
			}
		}
	}

	return "", fmt.Errorf("no intermediate artifact file found for %s %s", name, version)
}
