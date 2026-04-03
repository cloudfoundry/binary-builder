package main

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/stretchr/testify/assert"
)

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
