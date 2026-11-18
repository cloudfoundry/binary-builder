package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/artifact"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/recipe"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRecipe is a minimal recipe.Recipe used to drive finalizeArtifact without
// running a real build.
type stubRecipe struct {
	name string
	meta recipe.ArtifactMeta
}

func (r *stubRecipe) Name() string { return r.name }

func (r *stubRecipe) Build(context.Context, *stack.Stack, *source.Input, runner.Runner, *output.OutData) error {
	return nil
}

func (r *stubRecipe) Artifact() recipe.ArtifactMeta { return r.meta }

// TestBuildSummaryArtifactPath verifies that buildSummary uses ArtifactFilename
// (the real on-disk name) for artifact_path, not filepath.Base(URL).
// This is a regression test for the openjdk '+' encoding bug: S3URL encodes '+'
// as '%2B', so filepath.Base(URL) would produce the wrong local filename.
func TestBuildSummaryArtifactPath(t *testing.T) {
	t.Run("version with plus sign uses literal filename not URL-encoded path", func(t *testing.T) {
		// Simulates openjdk 8.0.482+10: the URL has %2B but the disk file has +.
		outData := &output.OutData{
			Version:          "8.0.482+10",
			SHA256:           "f05d0dba",
			URL:              "https://buildpacks.cloudfoundry.org/dependencies/openjdk/openjdk_8.0.482%2B10_linux_x64_cflinuxfs4_f05d0dba.tgz",
			ArtifactFilename: "openjdk_8.0.482+10_linux_x64_cflinuxfs4_f05d0dba.tgz",
		}

		summary := buildSummary(outData)

		// artifact_path must be the literal on-disk name (with '+'), not the
		// URL-encoded name (with '%2B') that build.sh would fail to find.
		assert.Equal(t, "openjdk_8.0.482+10_linux_x64_cflinuxfs4_f05d0dba.tgz", summary.ArtifactPath)
		assert.Equal(t, "https://buildpacks.cloudfoundry.org/dependencies/openjdk/openjdk_8.0.482%2B10_linux_x64_cflinuxfs4_f05d0dba.tgz", summary.URL)
	})

	t.Run("normal version without plus sign is unaffected", func(t *testing.T) {
		outData := &output.OutData{
			Version:          "3.3.6",
			SHA256:           "abcdef01",
			URL:              "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_linux_x64_cflinuxfs4_abcdef01.tgz",
			ArtifactFilename: "ruby_3.3.6_linux_x64_cflinuxfs4_abcdef01.tgz",
		}

		summary := buildSummary(outData)

		assert.Equal(t, "ruby_3.3.6_linux_x64_cflinuxfs4_abcdef01.tgz", summary.ArtifactPath)
	})

	t.Run("URL-passthrough dep (empty ArtifactFilename) falls back to filepath.Base(URL)", func(t *testing.T) {
		// Miniconda: no compiled artifact, outData.URL is set directly by the recipe.
		// ArtifactFilename is never set. build.sh skips the file move for these deps.
		outData := &output.OutData{
			Version: "py39_23.3.1-0",
			URL:     "https://repo.anaconda.com/miniconda/Miniconda3-py39_23.3.1-0-Linux-x86_64.sh",
			SHA256:  "deadbeef",
			// ArtifactFilename intentionally left empty (zero value).
		}

		summary := buildSummary(outData)

		assert.Equal(t, filepath.Base(outData.URL), summary.ArtifactPath)
	})
}

// TestFindIntermediateArtifactRecognizesAllPassthroughExtensions is a
// regression test for the yarn-berry rollout: PassthroughRecipe-based
// dependencies can produce a bare, non-archive file extension (e.g. ".js"
// for yarn-berry's single bundled CLI script), and findIntermediateArtifact
// must recognize every extension actually produced by a registered recipe,
// or the build fails with "no intermediate artifact file found" even though
// the recipe succeeded and the file is sitting right there in CWD.
func TestFindIntermediateArtifactRecognizesAllPassthroughExtensions(t *testing.T) {
	cases := []struct {
		name     string
		version  string
		filename string
	}{
		{"yarn-berry", "4.18.0", "yarn-berry-4.18.0.js"},
		{"composer", "2.7.1", "composer-2.7.1.phar"},
		{"tomcat", "9.0.85", "tomcat-9.0.85.tar.gz"},
		{"java-cfenv", "3.5.0", "java-cfenv-3.5.0.jar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origDir, err := os.Getwd()
			require.NoError(t, err)
			tmpDir := t.TempDir()
			require.NoError(t, os.Chdir(tmpDir))
			defer os.Chdir(origDir)

			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, tc.filename), []byte("fake artifact content"), 0644))

			got, err := findIntermediateArtifact(tc.name, tc.version)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(".", tc.filename), got)
		})
	}
}

// TestFinalizeArtifactPublishedExtension is a regression test for the
// yarn-berry artifact extension bug: the intermediate file is a raw Yarn CLI
// script named "yarn-berry-<version>.js", but it was published as
// "...<sha8>.tgz", which consumers then failed to gunzip.
//
// The gap this closes: TestFindIntermediateArtifactRecognizesAllPassthroughExtensions
// only asserts the intermediate name, so PR #120 passed while the published
// extension was still wrong.
func TestFinalizeArtifactPublishedExtension(t *testing.T) {
	cases := []struct {
		name         string
		version      string
		intermediate string
		wantExt      string
	}{
		{"yarn-berry", "4.18.1", "yarn-berry-4.18.1.js", "js"},
		{"composer", "2.7.1", "composer-2.7.1.phar", "phar"},
		{"tomcat", "9.0.85", "tomcat-9.0.85.tar.gz", "tgz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origDir, err := os.Getwd()
			require.NoError(t, err)
			tmpDir := t.TempDir()
			require.NoError(t, os.Chdir(tmpDir))
			defer os.Chdir(origDir)

			content := []byte("fake artifact content")
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, tc.intermediate), content, 0644))

			src := &source.Input{Name: tc.name, Version: tc.version}
			rec := &stubRecipe{
				name: tc.name,
				meta: recipe.ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			}
			s := &stack.Stack{Name: "cflinuxfs4"}
			outData := &output.OutData{Version: tc.version}

			require.NoError(t, finalizeArtifact(src, rec, s, outData))

			sum := sha256.Sum256(content)
			sha8 := fmt.Sprintf("%x", sum)[:8]
			a := artifact.Artifact{
				Name:    tc.name,
				Version: tc.version,
				OS:      "linux",
				Arch:    "noarch",
				Stack:   "any-stack",
			}
			want := a.Filename(sha8, tc.wantExt)

			assert.Equal(t, want, outData.ArtifactFilename)
			assert.Equal(t, "."+tc.wantExt, filepath.Ext(outData.ArtifactFilename))
			assert.Equal(t, a.S3URL(want), outData.URL)
		})
	}
}

// TestFindIntermediateArtifactRecognizesEveryKnownSuffix verifies that the
// intermediate-file search covers every format in artifact's extension table.
// Both now derive from the same table, so a new format (e.g. ".cjs") needs one
// edit and immediately works for both finding and publishing the artifact.
func TestFindIntermediateArtifactRecognizesEveryKnownSuffix(t *testing.T) {
	for _, suffix := range artifact.SearchSuffixes() {
		t.Run(suffix, func(t *testing.T) {
			origDir, err := os.Getwd()
			require.NoError(t, err)
			tmpDir := t.TempDir()
			require.NoError(t, os.Chdir(tmpDir))
			defer os.Chdir(origDir)

			filename := "dep-1.2.3." + suffix
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, filename), []byte("fake artifact content"), 0644))

			got, err := findIntermediateArtifact("dep", "1.2.3")
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(".", filename), got)

			// The published name must keep the file's identity: no suffix in
			// the table may silently fall back to ".tgz". ".tar.gz" is the one
			// deliberate normalization (published as ".tgz").
			if suffix != "tgz" && suffix != "tar.gz" {
				assert.Equal(t, suffix, artifact.ExtFromPath(filename))
			}
		})
	}
}
