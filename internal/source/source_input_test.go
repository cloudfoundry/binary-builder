package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseModernFormat verifies parsing of the real depwatcher output format:
//
//	{
//	  "source": {"name": "...", "type": "...", "repo": "..."},
//	  "version": {"url": "...", "ref": "...", "sha256": "...", "sha512": "..."}
//	}
func TestParseModernFormat(t *testing.T) {
	data := []byte(`{
		"source": {
			"name": "ruby",
			"type": "github-releases",
			"repo": "ruby/ruby"
		},
		"version": {
			"url": "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz",
			"ref": "3.3.6",
			"sha256": "abc123",
			"sha512": "def456",
			"git_commit_sha": "deadbeef"
		}
	}`)

	input, err := source.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "ruby", input.Name)
	assert.Equal(t, "3.3.6", input.Version)
	assert.Equal(t, "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz", input.URL)
	assert.Equal(t, "abc123", input.SHA256)
	assert.Equal(t, "def456", input.SHA512)
	assert.Equal(t, "deadbeef", input.GitCommitSHA)
	assert.Equal(t, "github-releases", input.Type)
	assert.Equal(t, "ruby/ruby", input.Repo)
}

func TestParseModernFormatWithMD5(t *testing.T) {
	data := []byte(`{
		"source": {"name": "node", "type": "node_lts"},
		"version": {
			"url": "https://nodejs.org/dist/v20.11.0/node-v20.11.0.tar.gz",
			"ref": "20.11.0",
			"md5_digest": "md5hashvalue"
		}
	}`)

	input, err := source.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "node", input.Name)
	assert.Equal(t, "20.11.0", input.Version)
	assert.Equal(t, "md5hashvalue", input.MD5)
	assert.Empty(t, input.SHA256)
}

func TestParseLegacyFormat(t *testing.T) {
	data := []byte(`{
		"name": "python",
		"version": "3.12.0",
		"source_uri": "https://www.python.org/ftp/python/3.12.0/Python-3.12.0.tgz",
		"source_sha": "abc123md5"
	}`)

	input, err := source.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "python", input.Name)
	assert.Equal(t, "3.12.0", input.Version)
	assert.Equal(t, "https://www.python.org/ftp/python/3.12.0/Python-3.12.0.tgz", input.URL)
	assert.Equal(t, "abc123md5", input.MD5)
	assert.Empty(t, input.SHA256)
}

func TestPrimaryChecksumPrefersSHA512(t *testing.T) {
	input := &source.Input{
		SHA512: "sha512val",
		SHA256: "sha256val",
		MD5:    "md5val",
	}

	cs := input.PrimaryChecksum()
	assert.Equal(t, "sha512", cs.Algorithm)
	assert.Equal(t, "sha512val", cs.Value)
}

func TestPrimaryChecksumFallsBackToSHA256(t *testing.T) {
	input := &source.Input{
		SHA256: "sha256val",
		MD5:    "md5val",
	}

	cs := input.PrimaryChecksum()
	assert.Equal(t, "sha256", cs.Algorithm)
	assert.Equal(t, "sha256val", cs.Value)
}

func TestPrimaryChecksumFallsBackToMD5(t *testing.T) {
	input := &source.Input{
		MD5: "md5val",
	}

	cs := input.PrimaryChecksum()
	assert.Equal(t, "md5", cs.Algorithm)
	assert.Equal(t, "md5val", cs.Value)
}

func TestPrimaryChecksumFallsBackToSHA1(t *testing.T) {
	input := &source.Input{
		SHA1: "sha1val",
	}

	cs := input.PrimaryChecksum()
	assert.Equal(t, "sha1", cs.Algorithm)
	assert.Equal(t, "sha1val", cs.Value)
}

func TestPrimaryChecksumEmpty(t *testing.T) {
	input := &source.Input{}

	cs := input.PrimaryChecksum()
	assert.Empty(t, cs.Algorithm)
	assert.Empty(t, cs.Value)
}

func TestFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.json")
	data := []byte(`{
		"source": {"name": "node", "type": "node_lts"},
		"version": {
			"url": "https://nodejs.org/dist/v20.11.0/node-v20.11.0.tar.gz",
			"ref": "20.11.0",
			"sha256": "abc123"
		}
	}`)
	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)

	input, err := source.FromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "node", input.Name)
	assert.Equal(t, "20.11.0", input.Version)
	assert.Equal(t, "abc123", input.SHA256)
}

func TestFromFileMissing(t *testing.T) {
	_, err := source.FromFile("/nonexistent/data.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading source file")
}

func TestParseMalformedJSON(t *testing.T) {
	_, err := source.Parse([]byte("{invalid json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing source JSON")
}

func TestParseModernWithRepo(t *testing.T) {
	data := []byte(`{
		"source": {
			"name": "libgdiplus",
			"type": "github_releases",
			"repo": "mono/libgdiplus"
		},
		"version": {
			"url": "https://github.com/mono/libgdiplus/archive/6.1.tar.gz",
			"ref": "6.1",
			"sha256": "abc123"
		}
	}`)

	input, err := source.Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "mono/libgdiplus", input.Repo)
	assert.Equal(t, "6.1", input.Version)
}
