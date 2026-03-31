// Command binary-builder builds a single dependency for a given CF stack.
//
// Two input modes are supported:
//
// Mode 1 — Direct flags (manual/local use):
//
//	binary-builder build \
//	  --stack cflinuxfs4 \
//	  --name ruby \
//	  --version 3.3.6 \
//	  --url https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz \
//	  --sha256 8dc48f...
//
// Mode 2 — Source file (CI use, depwatcher data.json):
//
//	binary-builder build \
//	  --stack cflinuxfs4 \
//	  --source-file source/data.json
//
// Selection logic:
//   - If --name is provided → build source.Input directly from flags
//     (--version is required; --url/--sha256/--sha512 are optional)
//   - Else if --source-file is explicitly given OR source/data.json exists
//     at the default path → read from file
//   - Else → error: provide either --name/--version or --source-file
//
// The tool compiles the dependency inside a temp directory and writes the
// final artifact to the current working directory using the canonical filename:
//
//	<name>_<version>_<os>_<arch>_<stack>_<sha8>.<ext>
//
// On success it writes a JSON summary to --output-file (default: summary.json):
//
//	{
//	  "artifact_path": "ruby_3.3.6_linux_x64_cflinuxfs4_abcdef01.tgz",
//	  "version":       "3.3.6",
//	  "sha256":        "abcdef01...",
//	  "url":           "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_...",
//	  "source":        {"url": "...", "sha256": "...", ...},
//	  "sub_dependencies": {...}
//	}
//
// All build subprocess output goes to stdout/stderr and is visible in logs.
// The JSON summary is always written to a file, never to stdout, so that
// build noise from compilers and make does not corrupt the structured output.
//
// All artifact renaming, dep-metadata writing, builds-artifacts JSON, and git
// commits are the responsibility of the CI task that wraps this tool.
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

const defaultOutputFile = "summary.json"

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

	// Required.
	stackName := fs.String("stack", "", "Stack name (e.g. cflinuxfs4, cflinuxfs5) [required]")

	// Mode 1 — direct flags (manual/local use).
	name := fs.String("name", "", "Dependency name (e.g. ruby); triggers direct-input mode")
	version := fs.String("version", "", "Version string (required when --name is set)")
	url := fs.String("url", "", "Source tarball URL (optional with --name)")
	sha256 := fs.String("sha256", "", "SHA256 of the source tarball (optional with --name)")
	sha512 := fs.String("sha512", "", "SHA512 of the source tarball (optional with --name)")

	// Mode 2 — source file (CI / depwatcher use).
	sourceFile := fs.String("source-file", "source/data.json", "Path to depwatcher data.json")

	stacksDir := fs.String("stacks-dir", "stacks", "Directory containing stack YAML files")

	// Output — JSON summary is always written to a file, never to stdout.
	// Build subprocess output (compilers, make, etc.) flows to stdout/stderr
	// so it is visible in logs without corrupting the structured JSON output.
	outputFile := fs.String("output-file", defaultOutputFile, "Path to write the JSON build summary")

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	if *stackName == "" {
		return fmt.Errorf("--stack is required")
	}

	// Determine whether --source-file was explicitly passed.
	sourceFileExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "source-file" {
			sourceFileExplicit = true
		}
	})

	// Resolve source input using the agreed mode-selection logic.
	var src *source.Input
	switch {
	case *name != "":
		// Mode 1: build source.Input directly from flags.
		if *version == "" {
			return fmt.Errorf("--version is required when --name is set")
		}
		src = &source.Input{
			Name:    *name,
			Version: *version,
			URL:     *url,
			SHA256:  *sha256,
			SHA512:  *sha512,
		}

	case sourceFileExplicit:
		// Mode 2a: --source-file was explicitly provided.
		var err error
		src, err = source.FromFile(*sourceFile)
		if err != nil {
			return fmt.Errorf("loading source file: %w", err)
		}

	default:
		// Mode 2b: check whether the default path exists on disk.
		if _, statErr := os.Stat(*sourceFile); statErr == nil {
			var err error
			src, err = source.FromFile(*sourceFile)
			if err != nil {
				return fmt.Errorf("loading source file: %w", err)
			}
		} else {
			return fmt.Errorf("provide either --name (with --version) or --source-file")
		}
	}

	// Load stack config.
	s, err := stack.Load(*stacksDir, *stackName)
	if err != nil {
		return fmt.Errorf("loading stack %q: %w", *stackName, err)
	}

	// Look up the recipe.
	reg := buildRegistry()
	rec, err := reg.Get(src.Name)
	if err != nil {
		return fmt.Errorf("no recipe for %q — registered: %v", src.Name, reg.Names())
	}

	// Prepare output data seeded from the source input.
	outData := output.NewOutData(src)

	// Run the build.
	ctx := context.Background()
	r := &runner.RealRunner{}

	fmt.Fprintf(os.Stderr, "[binary-builder] building %s %s for %s\n", src.Name, src.Version, *stackName)

	if err := rec.Build(ctx, s, src, r, outData); err != nil {
		return fmt.Errorf("building %s: %w", src.Name, err)
	}

	// Miniconda sets outData.URL directly (passthrough — no compiled artifact).
	// All other recipes produce an intermediate file in the working directory.
	if outData.URL == "" {
		if err := finalizeArtifact(src, rec, s, outData); err != nil {
			return err
		}
	}

	// Write the JSON summary to the output file.
	// Build subprocess output has already flowed to stdout/stderr (visible in
	// logs), so the output file contains only the clean structured JSON.
	f, err := os.Create(*outputFile)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", *outputFile, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buildSummary(outData)); err != nil {
		return fmt.Errorf("writing JSON summary: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[binary-builder] summary written to %s\n", *outputFile)
	return nil
}

// buildSummaryOutput is the JSON struct emitted to stdout.
type buildSummaryOutput struct {
	ArtifactPath    string                          `json:"artifact_path"`
	Version         string                          `json:"version"`
	SHA256          string                          `json:"sha256,omitempty"`
	URL             string                          `json:"url,omitempty"`
	Source          output.OutDataSource            `json:"source"`
	SubDependencies map[string]output.SubDependency `json:"sub_dependencies,omitempty"`
	GitCommitSHA    string                          `json:"git_commit_sha,omitempty"`
}

func buildSummary(outData *output.OutData) buildSummaryOutput {
	// ArtifactFilename is the actual disk filename (e.g. "openjdk_8.0.482+10_...tgz").
	// For URL-passthrough deps (e.g. miniconda) ArtifactFilename is empty, so we
	// fall back to filepath.Base(URL) — those deps produce no file and build.sh
	// skips the move anyway.
	artifactPath := outData.ArtifactFilename
	if artifactPath == "" {
		artifactPath = filepath.Base(outData.URL)
	}
	return buildSummaryOutput{
		ArtifactPath:    artifactPath,
		Version:         outData.Version,
		SHA256:          outData.SHA256,
		URL:             outData.URL,
		Source:          outData.Source,
		SubDependencies: outData.SubDependencies,
		GitCommitSHA:    outData.GitCommitSHA,
	}
}

// finalizeArtifact finds the intermediate artifact file written to CWD by the
// recipe, computes its SHA256, renames it to the canonical filename, and
// populates outData.SHA256 and outData.URL.
func finalizeArtifact(src *source.Input, rec recipe.Recipe, s *stack.Stack, outData *output.OutData) error {
	meta := rec.Artifact()

	// Resolve the effective stack label for the artifact filename.
	stackLabel := meta.Stack
	if stackLabel == "" {
		stackLabel = s.Name
	}

	// Use ArtifactVersion when set (e.g. jruby uses "9.4.14.0-ruby-3.1");
	// otherwise fall back to the raw source version.
	artifactVersion := outData.ArtifactVersion
	if artifactVersion == "" {
		artifactVersion = outData.Version
	}

	// Find the intermediate artifact produced by the recipe.
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

	// Build canonical filename and move artifact to CWD.
	a := artifact.Artifact{
		Name:    src.Name,
		Version: artifactVersion,
		OS:      meta.OS,
		Arch:    meta.Arch,
		Stack:   stackLabel,
	}
	finalFilename := a.Filename(sha256hex, ext)
	finalPath := filepath.Join(".", finalFilename)

	// Use cross-device-safe move in case the recipe wrote to os.TempDir.
	if err := fileutil.MoveFile(intermediatePath, finalPath); err != nil {
		return fmt.Errorf("moving artifact to %s: %w", finalPath, err)
	}

	outData.SHA256 = sha256hex
	outData.URL = a.S3URL(finalFilename)
	outData.ArtifactFilename = finalFilename

	fmt.Fprintf(os.Stderr, "[binary-builder] artifact: %s\n", finalFilename)
	return nil
}

// buildRegistry constructs and populates the full recipe registry.
func buildRegistry() *recipe.Registry {
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
	reg.Register(&recipe.OpenrestyRecipe{Fetcher: f})
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
	reg.Register(&recipe.PHPRecipe{Fetcher: f})

	// Simple / repack recipes.
	reg.Register(&recipe.PipRecipe{Fetcher: f})
	reg.Register(&recipe.PipenvRecipe{Fetcher: f})
	reg.Register(&recipe.BowerRecipe{Fetcher: f})
	reg.Register(&recipe.YarnRecipe{Fetcher: f})
	reg.Register(&recipe.RubygemsRecipe{Fetcher: f})
	reg.Register(&recipe.MinicondaRecipe{Fetcher: f})
	reg.Register(&recipe.DotnetSDKRecipe{Fetcher: f})
	reg.Register(&recipe.DotnetRuntimeRecipe{Fetcher: f})
	reg.Register(&recipe.DotnetAspnetcoreRecipe{Fetcher: f})

	// Passthrough recipes.
	for _, r := range recipe.NewPassthroughRecipes(f) {
		reg.Register(r)
	}

	return reg
}

// findIntermediateArtifact searches CWD (and os.TempDir as fallback) for the
// artifact file produced by a recipe. Extensions are tried in priority order.
func findIntermediateArtifact(name, version string) (string, error) {
	// Extensions in priority order (matches ArtifactOutput.ext in the old Ruby code).
	exts := []string{"tgz", "tar.gz", "zip", "tar.xz", "tar.bz2", "sh", "phar", "txt"}

	// Recipes that cannot write to CWD (e.g. pip, pipenv) write to os.TempDir.
	searchDirs := []string{".", os.TempDir()}

	for _, dir := range searchDirs {
		// name-version prefix first (most specific).
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
