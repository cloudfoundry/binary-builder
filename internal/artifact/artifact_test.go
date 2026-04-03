package artifact_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/artifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilenameLinuxDep(t *testing.T) {
	a := artifact.Artifact{
		Name:    "ruby",
		Version: "3.3.6",
		OS:      "linux",
		Arch:    "x64",
		Stack:   "cflinuxfs4",
	}

	filename := a.Filename("e4311262abcdef01", "tgz")
	assert.Equal(t, "ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz", filename)
}

func TestFilenameWindowsDep(t *testing.T) {
	a := artifact.Artifact{
		Name:    "hwc",
		Version: "2.0.0",
		OS:      "windows",
		Arch:    "x86-64",
		Stack:   "any-stack",
	}

	filename := a.Filename("abcd1234deadbeef", "zip")
	assert.Equal(t, "hwc_2.0.0_windows_x86-64_any-stack_abcd1234.zip", filename)
}

func TestFilenameNoarchDep(t *testing.T) {
	a := artifact.Artifact{
		Name:    "bundler",
		Version: "2.5.0",
		OS:      "linux",
		Arch:    "noarch",
		Stack:   "cflinuxfs4",
	}

	filename := a.Filename("abcd1234deadbeef", "tgz")
	assert.Equal(t, "bundler_2.5.0_linux_noarch_cflinuxfs4_abcd1234.tgz", filename)
}

func TestFilenamePrefixOnly8CharsSHA(t *testing.T) {
	a := artifact.Artifact{
		Name:    "ruby",
		Version: "3.3.6",
		OS:      "linux",
		Arch:    "x64",
		Stack:   "cflinuxfs4",
	}

	fullSHA := "e4311262abcdef0123456789abcdef0123456789abcdef0123456789abcdef01"
	filename := a.Filename(fullSHA, "tgz")
	assert.Contains(t, filename, "_e4311262.")
	assert.NotContains(t, filename, "abcdef01")
}

func TestS3URL(t *testing.T) {
	a := artifact.Artifact{
		Name: "ruby",
	}

	url := a.S3URL("ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz")
	assert.Equal(t, "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz", url)
}

func TestS3URLEncodesPlus(t *testing.T) {
	// PR #553: '+' in version strings (semver v2) must be encoded as %2B
	// to prevent AWS S3 permission denied errors.
	a := artifact.Artifact{
		Name: "openjdk",
	}

	url := a.S3URL("openjdk_11.0.22+7_linux_x64_cflinuxfs4_abcd1234.tgz")
	assert.Equal(t, "https://buildpacks.cloudfoundry.org/dependencies/openjdk/openjdk_11.0.22%2B7_linux_x64_cflinuxfs4_abcd1234.tgz", url)
	assert.NotContains(t, url, "+")
}

func TestS3URLNoEncodingNeeded(t *testing.T) {
	a := artifact.Artifact{
		Name: "ruby",
	}

	url := a.S3URL("ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz")
	// No '+' in filename, URL should be unchanged.
	assert.Equal(t, "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz", url)
}

func TestSHA256File(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	expected := fmt.Sprintf("%x", sha256.Sum256(content))
	actual, err := artifact.SHA256File(path)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSHA256FileMissing(t *testing.T) {
	_, err := artifact.SHA256File("/nonexistent/file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opening")
}

func TestExtFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"ruby-3.3.6.tar.gz", "tgz"},
		{"ruby-3.3.6.tgz", "tgz"},
		{"dotnet-sdk.tar.xz", "tar.xz"},
		{"appdynamics.tar.bz2", "tar.bz2"},
		{"hwc.zip", "zip"},
		{"composer.phar", "phar"},
		{"install.sh", "sh"},
		{"version.txt", "txt"},
		{"unknown.bin", "tgz"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, artifact.ExtFromPath(tt.path))
		})
	}
}

func TestFilenamePrefix(t *testing.T) {
	a := artifact.Artifact{
		Name:    "python",
		Version: "3.12.0",
		OS:      "linux",
		Arch:    "x64",
		Stack:   "cflinuxfs5",
	}

	assert.Equal(t, "python_3.12.0_linux_x64_cflinuxfs5", a.FilenamePrefix())
}
