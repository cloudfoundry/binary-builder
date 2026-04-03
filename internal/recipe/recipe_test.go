package recipe_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/recipe"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── FakeFetcher ──────────────────────────────────────────────────────────────

// FakeFetcher satisfies fetch.Fetcher without making any network calls.
type FakeFetcher struct {
	// DownloadedURLs records every (url, dest) pair passed to Download.
	DownloadedURLs []fetchCall
	// BodyMap maps URL → body bytes for ReadBody.
	BodyMap map[string][]byte
	// ErrMap maps URL → error for Download or ReadBody.
	ErrMap map[string]error
}

type fetchCall struct {
	URL  string
	Dest string
}

func newFakeFetcher() *FakeFetcher {
	return &FakeFetcher{
		BodyMap: make(map[string][]byte),
		ErrMap:  make(map[string]error),
	}
}

func (f *FakeFetcher) Download(_ context.Context, url, dest string, _ source.Checksum) error {
	f.DownloadedURLs = append(f.DownloadedURLs, fetchCall{URL: url, Dest: dest})
	if err, ok := f.ErrMap[url]; ok {
		return err
	}
	// Create the destination directory.
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	// For .tar.gz / .tgz destinations write a minimal valid gzip tarball so
	// callers that decompress the file (e.g. archive.StripTopLevelDir) don't
	// fail with "invalid gzip header".
	if strings.HasSuffix(dest, ".tar.gz") || strings.HasSuffix(dest, ".tgz") {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		// Write one top-level directory entry so StripTopLevelDir has something to strip.
		_ = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: "fake-top/", Mode: 0755})
		tw.Close() //nolint:errcheck
		gw.Close() //nolint:errcheck
		return os.WriteFile(dest, buf.Bytes(), 0644)
	}
	return os.WriteFile(dest, []byte("fake-content"), 0644)
}

func (f *FakeFetcher) ReadBody(_ context.Context, url string) ([]byte, error) {
	if err, ok := f.ErrMap[url]; ok {
		return nil, err
	}
	if body, ok := f.BodyMap[url]; ok {
		return body, nil
	}
	return []byte("fake-body"), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newStack(t *testing.T) *stack.Stack {
	t.Helper()
	return &stack.Stack{
		Name: "cflinuxfs4",
		AptPackages: map[string][]string{
			"hwc_build":         {"mingw-w64"},
			"pip_build":         {"python3", "python3-pip"},
			"python_deb_extras": {"libxss1"},
			"python_build":      {"libdb-dev", "libgdbm-dev", "tk8.6-dev"},
			"node_build":        {},
		},
		Python: stack.PythonConfig{TCLVersion: "8.6"},
	}
}

func newInput(name, version, url string) *source.Input {
	return &source.Input{
		Name:    name,
		Version: version,
		URL:     url,
		SHA256:  "abc123",
	}
}

// ── Registry ─────────────────────────────────────────────────────────────────

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := recipe.NewRegistry()
	f := newFakeFetcher()
	r := &recipe.PassthroughRecipe{
		DepName:            "tomcat",
		SourceFilenameFunc: func(v string) string { return fmt.Sprintf("apache-tomcat-%s.tar.gz", v) },
		Meta:               recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
		Fetcher:            f,
	}
	reg.Register(r)

	got, err := reg.Get("tomcat")
	require.NoError(t, err)
	assert.Equal(t, "tomcat", got.Name())
}

func TestRegistryGetUnknown(t *testing.T) {
	reg := recipe.NewRegistry()
	_, err := reg.Get("does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestRegistryNames(t *testing.T) {
	reg := recipe.NewRegistry()
	f := newFakeFetcher()
	for _, name := range []string{"tomcat", "composer", "yarn"} {
		reg.Register(&recipe.PassthroughRecipe{
			DepName:            name,
			SourceFilenameFunc: func(v string) string { return name + ".tgz" },
			Meta:               recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher:            f,
		})
	}

	names := reg.Names()
	assert.Len(t, names, 3)
	assert.ElementsMatch(t, []string{"tomcat", "composer", "yarn"}, names)
}

// ── PassthroughRecipe ────────────────────────────────────────────────────────

func TestPassthroughRecipeDownloadsWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// Ensure the "source" dir exists inside our temp dir so the recipe can write to it.
	sourceDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	// Change to the temp dir so relative paths in the recipe resolve correctly.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	f := newFakeFetcher()
	r := &recipe.PassthroughRecipe{
		DepName:            "tomcat",
		SourceFilenameFunc: func(v string) string { return fmt.Sprintf("apache-tomcat-%s.tar.gz", v) },
		Meta:               recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
		Fetcher:            f,
	}

	src := newInput("tomcat", "9.0.85", "https://example.com/tomcat.tar.gz")
	err = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.NoError(t, err)

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, "https://example.com/tomcat.tar.gz", f.DownloadedURLs[0].URL)
	assert.Equal(t, filepath.Join("source", "apache-tomcat-9.0.85.tar.gz"), f.DownloadedURLs[0].Dest)
}

func TestPassthroughRecipeSkipsDownloadIfFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	// Pre-create the file so it already exists.
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "apache-tomcat-9.0.85.tar.gz"), []byte("data"), 0644))

	f := newFakeFetcher()
	r := &recipe.PassthroughRecipe{
		DepName:            "tomcat",
		SourceFilenameFunc: func(v string) string { return fmt.Sprintf("apache-tomcat-%s.tar.gz", v) },
		Meta:               recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
		Fetcher:            f,
	}

	src := newInput("tomcat", "9.0.85", "https://example.com/tomcat.tar.gz")
	err = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.NoError(t, err)

	assert.Empty(t, f.DownloadedURLs, "should not re-download existing file")
}

func TestPassthroughRecipeFetchError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "source"), 0755))
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	f := newFakeFetcher()
	f.ErrMap["https://example.com/tomcat.tar.gz"] = errors.New("network failure")

	r := &recipe.PassthroughRecipe{
		DepName:            "tomcat",
		SourceFilenameFunc: func(v string) string { return fmt.Sprintf("apache-tomcat-%s.tar.gz", v) },
		Meta:               recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
		Fetcher:            f,
	}

	src := newInput("tomcat", "9.0.85", "https://example.com/tomcat.tar.gz")
	err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network failure")
}

// Test the source filename functions for every passthrough recipe.
func TestPassthroughSourceFilenames(t *testing.T) {
	f := newFakeFetcher()
	recipes := recipe.NewPassthroughRecipes(f)

	cases := []struct {
		name     string
		version  string
		wantFile string
	}{
		{"tomcat", "9.0.85", "apache-tomcat-9.0.85.tar.gz"},
		{"composer", "2.7.1", "composer.phar"},
		{"appdynamics", "23.11.0.35198", "appdynamics-php-agent-linux_x64-23.11.0.35198.tar.bz2"},
		{"appdynamics-java", "23.11.0.35198", "appdynamics-java-agent-23.11.0.35198.zip"},
		{"skywalking-agent", "9.2.0", "apache-skywalking-java-agent-9.2.0.tgz"},
		{"openjdk", "21.0.2+13", "bellsoft-jre21.0.2+13-linux-amd64.tar.gz"},
		{"zulu", "21.0.2", "zulu21.0.2-jre-linux_x64.tar.gz"},
		{"sapmachine", "21.0.2", "sapmachine-jre-21.0.2_linux-x64_bin.tar.gz"},
		{"jprofiler-profiler", "13.0.14", "jprofiler_linux_13_0_14.tar.gz"},
		{"your-kit-profiler", "2023.11.462", "YourKit-JavaProfiler-2023.11.462.zip"},
	}

	recipeMap := make(map[string]recipe.Recipe)
	for _, rec := range recipes {
		recipeMap[rec.Name()] = rec
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := recipeMap[tc.name]
			require.True(t, ok, "recipe %q not found", tc.name)

			// We need to check the source filename function indirectly by calling Build
			// and inspecting what was passed to the fetcher. Use a fresh fetcher.
			ff := newFakeFetcher()
			pr := rec.(*recipe.PassthroughRecipe)
			pr.Fetcher = ff

			tmpDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "source"), 0755))
			origDir, _ := os.Getwd()
			require.NoError(t, os.Chdir(tmpDir))
			defer os.Chdir(origDir)

			src := newInput(tc.name, tc.version, "https://example.com/file")
			err := pr.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
			require.NoError(t, err)

			require.Len(t, ff.DownloadedURLs, 1)
			assert.Equal(t, filepath.Join("source", tc.wantFile), ff.DownloadedURLs[0].Dest)
		})
	}
}

func TestPassthroughArtifactMeta(t *testing.T) {
	f := newFakeFetcher()
	recipes := recipe.NewPassthroughRecipes(f)
	recipeMap := make(map[string]recipe.Recipe)
	for _, rec := range recipes {
		recipeMap[rec.Name()] = rec
	}

	anyStack := []string{"tomcat", "composer", "appdynamics", "appdynamics-java", "skywalking-agent"}
	for _, name := range anyStack {
		t.Run(name+"_any-stack", func(t *testing.T) {
			rec := recipeMap[name]
			assert.Equal(t, "any-stack", rec.Artifact().Stack)
			assert.Equal(t, "noarch", rec.Artifact().Arch)
		})
	}

	stackSpecific := []string{"openjdk", "zulu", "sapmachine", "jprofiler-profiler", "your-kit-profiler"}
	for _, name := range stackSpecific {
		t.Run(name+"_x64", func(t *testing.T) {
			rec := recipeMap[name]
			assert.Equal(t, "x64", rec.Artifact().Arch)
		})
	}
}

// ── NewPassthroughRecipes ─────────────────────────────────────────────────────

// TestNewPassthroughRecipesContents verifies that every expected dep name is
// present. Prefer extending this list over bumping a raw count.
func TestNewPassthroughRecipesContents(t *testing.T) {
	f := newFakeFetcher()
	recipes := recipe.NewPassthroughRecipes(f)
	names := make([]string, len(recipes))
	for i, r := range recipes {
		names[i] = r.Name()
	}
	assert.Subset(t, names, []string{
		"tomcat", "composer", "appdynamics", "appdynamics-java",
		"skywalking-agent", "openjdk", "zulu", "sapmachine",
		"jprofiler-profiler", "your-kit-profiler",
		"setuptools", "flit-core",
	})
}

// ── BowerRecipe ───────────────────────────────────────────────────────────────

func TestBowerRecipeDownloads(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.BowerRecipe{Fetcher: f}

	src := newInput("bower", "1.8.14", "https://example.com/bower.tgz")
	err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.NoError(t, err)

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, "https://example.com/bower.tgz", f.DownloadedURLs[0].URL)
	assert.Equal(t, filepath.Join(os.TempDir(), "bower-1.8.14.tgz"), f.DownloadedURLs[0].Dest)
}

func TestBowerRecipeArtifact(t *testing.T) {
	r := &recipe.BowerRecipe{}
	assert.Equal(t, "bower", r.Name())
	assert.Equal(t, "noarch", r.Artifact().Arch)
	assert.Equal(t, "linux", r.Artifact().OS)
}

// ── YarnRecipe ────────────────────────────────────────────────────────────────

func TestYarnRecipeStripsVPrefix(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	src := newInput("yarn", "v1.22.22", "https://example.com/yarn.tgz")
	r := &recipe.YarnRecipe{Fetcher: f}
	outData := &output.OutData{}
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, outData)

	require.Len(t, f.DownloadedURLs, 1)
	// File on disk uses the stripped version.
	assert.Equal(t, filepath.Join(os.TempDir(), "yarn-1.22.22.tgz"), f.DownloadedURLs[0].Dest)
	// outData.Version must be the stripped version so findIntermediateArtifact matches.
	assert.Equal(t, "1.22.22", outData.Version)
	// src.Version must NOT be mutated — callers after Build rely on the original value.
	assert.Equal(t, "v1.22.22", src.Version)
}

func TestYarnRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.YarnRecipe{}
	assert.Equal(t, "yarn", r.Name())
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

// ── PyPISourceRecipe ──────────────────────────────────────────────────────────

func TestPyPISourceRecipeFilenameFromURL(t *testing.T) {
	cases := []struct {
		depName string
		version string
		url     string
		wantDst string
	}{
		{
			depName: "setuptools",
			version: "69.0.3",
			url:     "https://example.com/setuptools-69.0.3.tar.gz",
			wantDst: filepath.Join(os.TempDir(), "setuptools-69.0.3.tar.gz"),
		},
		{
			depName: "flit-core",
			version: "3.9.0",
			url:     "https://example.com/flit_core-3.9.0.tar.gz",
			wantDst: filepath.Join(os.TempDir(), "flit_core-3.9.0.tar.gz"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.depName, func(t *testing.T) {
			f := newFakeFetcher()
			r := &recipe.PyPISourceRecipe{DepName: tc.depName, Fetcher: f}
			src := newInput(tc.depName, tc.version, tc.url)
			_ = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})

			require.Len(t, f.DownloadedURLs, 1)
			assert.Equal(t, tc.wantDst, f.DownloadedURLs[0].Dest)
		})
	}
}

func TestPyPISourceRecipeZipURL(t *testing.T) {
	f := newFakeFetcher()

	src := newInput("setuptools", "69.0.3", "https://example.com/setuptools-69.0.3.zip")
	r := &recipe.PyPISourceRecipe{DepName: "setuptools", Fetcher: f}
	_ = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, filepath.Join(os.TempDir(), "setuptools-69.0.3.zip"), f.DownloadedURLs[0].Dest)
}

func TestPyPISourceRecipeNameAndArtifact(t *testing.T) {
	cases := []struct{ depName string }{
		{"setuptools"},
		{"flit-core"},
	}
	for _, tc := range cases {
		t.Run(tc.depName, func(t *testing.T) {
			r := &recipe.PyPISourceRecipe{DepName: tc.depName}
			assert.Equal(t, tc.depName, r.Name())
			assert.Equal(t, "noarch", r.Artifact().Arch)
			assert.Equal(t, "linux", r.Artifact().OS)
		})
	}
}

func TestPyPISourceRecipeStripsURLFragment(t *testing.T) {
	// PyPI JSON API URLs sometimes include a #sha256=… fragment; the local
	// filename must not contain the fragment.
	f := newFakeFetcher()
	r := &recipe.PyPISourceRecipe{DepName: "flit-core", Fetcher: f}
	src := newInput("flit-core", "3.9.0", "https://files.pythonhosted.org/packages/flit_core-3.9.0.tar.gz#sha256=abc123")
	_ = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, filepath.Join(os.TempDir(), "flit_core-3.9.0.tar.gz"), f.DownloadedURLs[0].Dest,
		"fragment must be stripped from destination filename")
}

// ── RubygemsRecipe ────────────────────────────────────────────────────────────

func TestRubygemsRecipeDownloads(t *testing.T) {
	f := newFakeFetcher()

	src := newInput("rubygems", "3.5.6", "https://example.com/rubygems.tgz")
	r := &recipe.RubygemsRecipe{Fetcher: f}
	_ = r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, "https://example.com/rubygems.tgz", f.DownloadedURLs[0].URL)
	assert.Equal(t, filepath.Join(os.TempDir(), "rubygems-3.5.6.tgz"), f.DownloadedURLs[0].Dest)
}

func TestRubygemsRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.RubygemsRecipe{}
	assert.Equal(t, "rubygems", r.Name())
	assert.Equal(t, "noarch", r.Artifact().Arch)
	assert.Equal(t, "any-stack", r.Artifact().Stack)
}

// ── MinicondaRecipe ───────────────────────────────────────────────────────────

func TestMinicondaRecipeSetsOutData(t *testing.T) {
	body := []byte("#!/bin/bash\necho miniconda installer")
	expectedSHA := fmt.Sprintf("%x", sha256.Sum256(body))

	f := newFakeFetcher()
	f.BodyMap["https://repo.anaconda.com/miniconda/Miniconda3-py39_4.12.0-Linux-x86_64.sh"] = body

	r := &recipe.MinicondaRecipe{Fetcher: f}
	src := newInput("miniconda3-py39", "py39_4.12.0", "https://repo.anaconda.com/miniconda/Miniconda3-py39_4.12.0-Linux-x86_64.sh")
	outData := &output.OutData{}

	err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), outData)
	require.NoError(t, err)

	assert.Equal(t, src.URL, outData.URL)
	assert.Equal(t, expectedSHA, outData.SHA256)
}

func TestMinicondaRecipeNoFileDownloaded(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.MinicondaRecipe{Fetcher: f}

	src := newInput("miniconda3-py39", "py39_4.12.0", "https://repo.anaconda.com/miniconda/installer.sh")
	err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.NoError(t, err)

	// No Download calls — miniconda uses ReadBody, not Download.
	assert.Empty(t, f.DownloadedURLs)
}

func TestMinicondaRecipeFetchError(t *testing.T) {
	f := newFakeFetcher()
	f.ErrMap["https://repo.anaconda.com/miniconda/installer.sh"] = errors.New("timeout")

	r := &recipe.MinicondaRecipe{Fetcher: f}
	src := newInput("miniconda3-py39", "py39_4.12.0", "https://repo.anaconda.com/miniconda/installer.sh")
	err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestMinicondaRecipeArtifact(t *testing.T) {
	r := &recipe.MinicondaRecipe{}
	assert.Equal(t, "miniconda3-py39", r.Name())
	assert.Equal(t, "any-stack", r.Artifact().Stack)
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

// ── PipRecipe ─────────────────────────────────────────────────────────────────

func TestPipRecipeCallSequence(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipRecipe{Fetcher: f}
	src := newInput("pip", "24.0", "https://example.com/pip.tgz")
	err := r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})
	require.NoError(t, err)

	// Verify apt-get update and install were called.
	names := callNames(fakeRunner.Calls)
	assert.Contains(t, names, "apt-get")

	// Verify pip3 was invoked.
	assert.True(t, anyCallContains(fakeRunner.Calls, "pip3"), "pip3 should be called")
	// Verify tar was called to bundle.
	assert.True(t, anyCallContains(fakeRunner.Calls, "tar"), "tar should be called")
}

func TestPipRecipeCVEWheelPin(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipRecipe{Fetcher: f}
	src := newInput("pip", "24.0", "https://example.com/pip.tgz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// Verify wheel>=0.46.2 pin is present somewhere in the call args.
	assert.True(t, anyArgsContain(fakeRunner.Calls, "wheel>=0.46.2"),
		"CVE-2026-24049 wheel pin must be present")
}

func TestPipRecipeOutputPath(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipRecipe{Fetcher: f}
	src := newInput("pip", "24.0", "https://example.com/pip.tgz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// The fetcher downloads pip source into the build tmpDir, not the final output path.
	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, "/tmp/pip-build-24.0/pip-24.0.tar.gz", f.DownloadedURLs[0].Dest)
}

func TestPipRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.PipRecipe{}
	assert.Equal(t, "pip", r.Name())
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

// ── PipenvRecipe ──────────────────────────────────────────────────────────────

func TestPipenvRecipeCallSequence(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipenvRecipe{Fetcher: f}
	src := newInput("pipenv", "2023.12.1", "https://example.com/pipenv.tgz")
	err := r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, anyCallContains(fakeRunner.Calls, "pip3"), "pip3 should be called")
	assert.True(t, anyCallContains(fakeRunner.Calls, "tar"), "tar should be called")
}

func TestPipenvRecipeOutputPathHasVPrefix(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipenvRecipe{Fetcher: f}
	src := newInput("pipenv", "2023.12.1", "https://example.com/pipenv.tgz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// Output tarball must have 'v' prefix: /tmp/pipenv-v{version}.tgz
	assert.True(t, anyArgsContain(fakeRunner.Calls, "/tmp/pipenv-v2023.12.1.tgz"),
		"pipenv output path must have v prefix")
}

func TestPipenvRecipeBundledDeps(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipenvRecipe{Fetcher: f}
	src := newInput("pipenv", "2023.12.1", "https://example.com/pipenv.tgz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// All 7 bundled packages must be downloaded.
	expectedDeps := []string{
		"pytest-runner", "setuptools_scm", "parver", "wheel>=0.46.2",
		"invoke", "flit_core", "hatch-vcs",
	}
	for _, dep := range expectedDeps {
		assert.True(t, anyArgsContain(fakeRunner.Calls, dep),
			"expected bundled dep %q to appear in runner calls", dep)
	}
}

func TestPipenvRecipeCVEWheelPin(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.PipenvRecipe{Fetcher: f}
	src := newInput("pipenv", "2023.12.1", "https://example.com/pipenv.tgz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	assert.True(t, anyArgsContain(fakeRunner.Calls, "wheel>=0.46.2"),
		"CVE-2026-24049 wheel pin must be present in pipenv build")
}

func TestPipenvRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.PipenvRecipe{}
	assert.Equal(t, "pipenv", r.Name())
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

// ── HWCRecipe ─────────────────────────────────────────────────────────────────

func TestHWCRecipeInstallsFromStackConfig(t *testing.T) {
	useTempWorkDir(t)
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"hwc_build": {"mingw-w64"}},
	}
	src := newInput("hwc", "2.9.0", "https://example.com/hwc.tgz")
	r := &recipe.HWCRecipe{Fetcher: f}
	_ = r.Build(context.Background(), s, src, fakeRunner, &output.OutData{})

	// mingw-w64 must be installed via apt-get from the stack config, not hardcoded.
	assert.True(t, hasCallMatching(fakeRunner.Calls, "apt-get", "mingw-w64"),
		"hwc_build apt package 'mingw-w64' must be installed from stack config")
}

func TestHWCRecipeUsesStackAptPackages(t *testing.T) {
	// Verify that a custom hwc_build list is honoured — the recipe must not
	// hardcode the package name.
	useTempWorkDir(t)
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "future-stack",
		AptPackages: map[string][]string{"hwc_build": {"mingw-w64-custom"}},
	}
	src := newInput("hwc", "2.9.0", "https://example.com/hwc.tgz")
	r := &recipe.HWCRecipe{Fetcher: f}
	_ = r.Build(context.Background(), s, src, fakeRunner, &output.OutData{})

	assert.True(t, hasCallMatching(fakeRunner.Calls, "apt-get", "mingw-w64-custom"),
		"recipe must use hwc_build packages from stack config, not a hardcoded value")
	assert.False(t, hasCallMatching(fakeRunner.Calls, "apt-get", "mingw-w64\x00"),
		"hardcoded 'mingw-w64' (without suffix) must not appear when stack config overrides it")
}

// ── PipRecipe / pip_build config ──────────────────────────────────────────────

func TestPipRecipeInstallsFromStackConfig(t *testing.T) {
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"pip_build": {"python3", "python3-pip"}},
	}
	src := newInput("pip", "24.0", "https://example.com/pip.tgz")
	r := &recipe.PipRecipe{Fetcher: f}
	_ = r.Build(context.Background(), s, src, fakeRunner, &output.OutData{})

	// python3 and python3-pip must come from stack config.
	assert.True(t, hasCallMatching(fakeRunner.Calls, "apt-get", "python3"),
		"pip_build package 'python3' must be installed from stack config")
	assert.True(t, hasCallMatching(fakeRunner.Calls, "apt-get", "python3-pip"),
		"pip_build package 'python3-pip' must be installed from stack config")
}

func TestPipRecipeUsesStackPipBuildPackages(t *testing.T) {
	// Verify a custom pip_build list is honoured.
	f := newFakeFetcher()
	fakeRunner := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "future-stack",
		AptPackages: map[string][]string{"pip_build": {"python3.12", "python3.12-pip"}},
	}
	src := newInput("pip", "24.0", "https://example.com/pip.tgz")
	r := &recipe.PipRecipe{Fetcher: f}
	_ = r.Build(context.Background(), s, src, fakeRunner, &output.OutData{})

	assert.True(t, hasCallMatching(fakeRunner.Calls, "apt-get", "python3.12"),
		"recipe must honour custom pip_build packages from stack config")
}

// ── DotnetSDKRecipe ───────────────────────────────────────────────────────────

// writeFakeDotnetSource creates a source/ directory with a minimal .tar.gz file
// so that filepath.Glob("source/*.tar.gz") in pruneDotnetFiles resolves correctly.
// Returns the resolved path (e.g. "source/dotnet-sdk-8.0.101.tar.gz").
func writeFakeDotnetSource(t *testing.T, filename string) string {
	t.Helper()
	if err := os.MkdirAll("source", 0755); err != nil {
		t.Fatalf("writeFakeDotnetSource: mkdir source: %v", err)
	}
	srcPath := filepath.Join("source", filename)
	if err := os.WriteFile(srcPath, []byte("fake-dotnet-tarball"), 0644); err != nil {
		t.Fatalf("writeFakeDotnetSource: write %s: %v", srcPath, err)
	}
	return srcPath
}

func TestDotnetSDKRecipeCallSequence(t *testing.T) {
	useTempWorkDir(t)
	srcPath := writeFakeDotnetSource(t, "dotnet-sdk-8.0.101.tar.gz")

	fakeRunner := runner.NewFakeRunner()
	// Provide output for the tar tf command using the resolved path.
	fakeRunner.OutputMap["tar tf "+srcPath+" ./shared/Microsoft.NETCore.App/"] =
		"./shared/Microsoft.NETCore.App/\n./shared/Microsoft.NETCore.App/8.0.1/\n"

	r := &recipe.DotnetSDKRecipe{}
	src := newInput("dotnet-sdk", "8.0.101", "https://example.com/dotnet-sdk.tar.gz")
	err := r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})
	require.NoError(t, err)

	// mkdir, tar extract, tar tf (runtime version), tar compress.
	// RuntimeVersion.txt is written via os.WriteFile (no "sh" call).
	names := callNames(fakeRunner.Calls)
	assert.Contains(t, names, "mkdir")
	assert.Contains(t, names, "tar")
	assert.NotContains(t, names, "sh", "RuntimeVersion.txt must not use sh; written via os.WriteFile")
}

func TestDotnetSDKRecipeExcludesSharedDir(t *testing.T) {
	useTempWorkDir(t)
	srcPath := writeFakeDotnetSource(t, "dotnet-sdk-8.0.101.tar.gz")

	fakeRunner := runner.NewFakeRunner()
	fakeRunner.OutputMap["tar tf "+srcPath+" ./shared/Microsoft.NETCore.App/"] = ""

	r := &recipe.DotnetSDKRecipe{}
	src := newInput("dotnet-sdk", "8.0.101", "https://example.com/dotnet-sdk.tar.gz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// Verify --exclude=./shared/* appears in the tar extract call.
	assert.True(t, anyArgsContain(fakeRunner.Calls, "--exclude=./shared/*"),
		"dotnet-sdk must exclude ./shared/*")
}

func TestDotnetSDKRecipeUsesXZCompression(t *testing.T) {
	useTempWorkDir(t)
	srcPath := writeFakeDotnetSource(t, "dotnet-sdk-8.0.101.tar.gz")

	fakeRunner := runner.NewFakeRunner()
	fakeRunner.OutputMap["tar tf "+srcPath+" ./shared/Microsoft.NETCore.App/"] =
		"./shared/Microsoft.NETCore.App/\n./shared/Microsoft.NETCore.App/8.0.1/\n"

	r := &recipe.DotnetSDKRecipe{}
	src := newInput("dotnet-sdk", "8.0.101", "https://example.com/dotnet-sdk.tar.gz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// Re-compression must use -Jcf (xz), not -czf (gzip).
	assert.True(t, anyArgsContain(fakeRunner.Calls, "-Jcf"),
		"dotnet-sdk must use xz compression (-Jcf)")
}

func TestDotnetSDKRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.DotnetSDKRecipe{}
	assert.Equal(t, "dotnet-sdk", r.Name())
	assert.Equal(t, "x64", r.Artifact().Arch)
}

// ── DotnetRuntimeRecipe ───────────────────────────────────────────────────────

func TestDotnetRuntimeRecipeExcludesDotnet(t *testing.T) {
	useTempWorkDir(t)
	writeFakeDotnetSource(t, "dotnet-runtime-8.0.1.tar.gz")
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.DotnetRuntimeRecipe{}
	src := newInput("dotnet-runtime", "8.0.1", "https://example.com/dotnet-runtime.tar.gz")
	err := r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, anyArgsContain(fakeRunner.Calls, "--exclude=./dotnet"),
		"dotnet-runtime must exclude ./dotnet")
}

func TestDotnetRuntimeRecipeNoRuntimeVersionTxt(t *testing.T) {
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.DotnetRuntimeRecipe{}
	src := newInput("dotnet-runtime", "8.0.1", "https://example.com/dotnet-runtime.tar.gz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	// No "sh" call: RuntimeVersion.txt must NOT be written for dotnet-runtime.
	assert.False(t, anyCallContains(fakeRunner.Calls, "sh"),
		"dotnet-runtime must NOT write RuntimeVersion.txt")
}

func TestDotnetRuntimeRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.DotnetRuntimeRecipe{}
	assert.Equal(t, "dotnet-runtime", r.Name())
	assert.Equal(t, "x64", r.Artifact().Arch)
}

// ── DotnetAspnetcoreRecipe ────────────────────────────────────────────────────

func TestDotnetAspnetcoreRecipeExcludesBoth(t *testing.T) {
	useTempWorkDir(t)
	writeFakeDotnetSource(t, "dotnet-aspnetcore-8.0.1.tar.gz")
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.DotnetAspnetcoreRecipe{}
	src := newInput("dotnet-aspnetcore", "8.0.1", "https://example.com/dotnet-aspnetcore.tar.gz")
	err := r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, anyArgsContain(fakeRunner.Calls, "--exclude=./dotnet"),
		"dotnet-aspnetcore must exclude ./dotnet")
	assert.True(t, anyArgsContain(fakeRunner.Calls, "--exclude=./shared/Microsoft.NETCore.App"),
		"dotnet-aspnetcore must exclude ./shared/Microsoft.NETCore.App")
}

func TestDotnetAspnetcoreRecipeNoRuntimeVersionTxt(t *testing.T) {
	fakeRunner := runner.NewFakeRunner()

	r := &recipe.DotnetAspnetcoreRecipe{}
	src := newInput("dotnet-aspnetcore", "8.0.1", "https://example.com/dotnet-aspnetcore.tar.gz")
	_ = r.Build(context.Background(), newStack(t), src, fakeRunner, &output.OutData{})

	assert.False(t, anyCallContains(fakeRunner.Calls, "sh"),
		"dotnet-aspnetcore must NOT write RuntimeVersion.txt")
}

func TestDotnetAspnetcoreRecipeNameAndArtifact(t *testing.T) {
	r := &recipe.DotnetAspnetcoreRecipe{}
	assert.Equal(t, "dotnet-aspnetcore", r.Name())
	assert.Equal(t, "x64", r.Artifact().Arch)
}

// ── computeSHA256 helper (via MinicondaRecipe) ────────────────────────────────

func TestComputeSHA256Determinism(t *testing.T) {
	// We test computeSHA256 indirectly via MinicondaRecipe which is the only
	// consumer. Same body → same SHA256 on every call.
	body := []byte("deterministic content")
	expected := fmt.Sprintf("%x", sha256.Sum256(body))

	f := newFakeFetcher()
	f.BodyMap["https://example.com/installer.sh"] = body

	r := &recipe.MinicondaRecipe{Fetcher: f}
	src := newInput("miniconda3-py39", "py39_4.12.0", "https://example.com/installer.sh")

	for i := 0; i < 3; i++ {
		outData := &output.OutData{}
		err := r.Build(context.Background(), newStack(t), src, runner.NewFakeRunner(), outData)
		require.NoError(t, err)
		assert.Equal(t, expected, outData.SHA256, "SHA256 must be deterministic (run %d)", i+1)
	}
}

// ── test helpers are in recipe_helpers_test.go ────────────────────────────────
