package output_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutData(t *testing.T) {
	src := &source.Input{
		Name:    "ruby",
		Version: "3.3.6",
		URL:     "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz",
		SHA256:  "abc123",
		SHA512:  "def456",
		MD5:     "md5val",
	}

	data := output.NewOutData(src)

	assert.Equal(t, "3.3.6", data.Version)
	assert.Equal(t, "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz", data.Source.URL)
	assert.Equal(t, "abc123", data.Source.SHA256)
	assert.Equal(t, "def456", data.Source.SHA512)
	require.NotNil(t, data.Source.MD5)
	assert.Equal(t, "md5val", *data.Source.MD5)
}

func TestNewOutDataEmptyChecksums(t *testing.T) {
	src := &source.Input{
		Version: "1.0.0",
		URL:     "https://example.com/dep.tgz",
	}

	data := output.NewOutData(src)

	assert.Equal(t, "1.0.0", data.Version)
	assert.Empty(t, data.Source.SHA256)
	assert.Empty(t, data.Source.SHA512)
	assert.Nil(t, data.Source.MD5)
	assert.Nil(t, data.Source.SHA1)
}

func TestOutDataJSON(t *testing.T) {
	data := &output.OutData{
		Version: "3.3.6",
		Source: output.OutDataSource{
			URL:    "https://example.com/ruby.tgz",
			SHA256: "abc123",
		},
		URL:    "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_linux_x64_cflinuxfs4_abc12345.tgz",
		SHA256: "fullsha256",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &parsed))

	assert.Equal(t, "3.3.6", parsed["version"])
	assert.Equal(t, "fullsha256", parsed["sha256"])

	src := parsed["source"].(map[string]interface{})
	assert.Equal(t, "abc123", src["sha256"])
}

func TestOutDataWithSubDependencies(t *testing.T) {
	data := &output.OutData{
		Version: "8.3.0",
		Source: output.OutDataSource{
			URL: "https://example.com/php.tgz",
		},
		SubDependencies: map[string]output.SubDependency{
			"redis":   {Version: "6.0.2"},
			"imagick": {Version: "3.7.0"},
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &parsed))

	subDeps := parsed["sub_dependencies"].(map[string]interface{})
	redis := subDeps["redis"].(map[string]interface{})
	assert.Equal(t, "6.0.2", redis["version"])
}

func TestOutDataWithSubDependenciesAndSource(t *testing.T) {
	data := &output.OutData{
		Version:      "4.3.2",
		GitCommitSHA: "deadbeef",
		SubDependencies: map[string]output.SubDependency{
			"forecast": {
				Source:  &output.SubDepSource{URL: "https://cran.r-project.org/forecast.tar.gz", SHA256: "abc"},
				Version: "8.21.1",
			},
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &parsed))

	assert.Equal(t, "deadbeef", parsed["git_commit_sha"])
	subDeps := parsed["sub_dependencies"].(map[string]interface{})
	forecast := subDeps["forecast"].(map[string]interface{})
	assert.Equal(t, "8.21.1", forecast["version"])
	forecastSrc := forecast["source"].(map[string]interface{})
	assert.Equal(t, "abc", forecastSrc["sha256"])
}

func TestBuildOutputAddOutput(t *testing.T) {
	tmpDir := t.TempDir()
	f := runner.NewFakeRunner()

	bo, err := output.NewBuildOutput("ruby", f, tmpDir)
	require.NoError(t, err)

	data := &output.OutData{
		Version: "3.3.6",
		Source: output.OutDataSource{
			URL:    "https://example.com/ruby.tgz",
			SHA256: "abc123",
		},
		URL:    "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6.tgz",
		SHA256: "fullsha",
	}

	err = bo.AddOutput("3.3.6-cflinuxfs4.json", data)
	require.NoError(t, err)

	// Verify file was written.
	path := filepath.Join(tmpDir, "binary-builds-new", "ruby", "3.3.6-cflinuxfs4.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed output.OutData
	require.NoError(t, json.Unmarshal(content, &parsed))
	assert.Equal(t, "3.3.6", parsed.Version)
	assert.Equal(t, "fullsha", parsed.SHA256)

	// AddOutput only writes the file; no git commands should be run.
	assert.Empty(t, f.Calls, "AddOutput should not invoke any git commands (git add is done in Commit)")
}

func TestDepMetadataOutputWriteMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	dmo := output.NewDepMetadataOutput(tmpDir)

	data := &output.OutData{
		Version: "3.3.6",
		URL:     "https://buildpacks.cloudfoundry.org/dependencies/ruby/ruby_3.3.6_linux_x64_cflinuxfs4_abc12345.tgz",
		SHA256:  "fullsha",
	}

	err := dmo.WriteMetadata("ruby_3.3.6_linux_x64_cflinuxfs4_abc12345.tgz", data)
	require.NoError(t, err)

	// Verify file was written with correct name.
	path := filepath.Join(tmpDir, "ruby_3.3.6_linux_x64_cflinuxfs4_abc12345.tgz_metadata.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed output.OutData
	require.NoError(t, json.Unmarshal(content, &parsed))
	assert.Equal(t, "3.3.6", parsed.Version)
	assert.Equal(t, "fullsha", parsed.SHA256)
}

func TestNewBuildOutputCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	f := runner.NewFakeRunner()

	bo, err := output.NewBuildOutput("python", f, tmpDir)
	require.NoError(t, err)

	expectedDir := filepath.Join(tmpDir, "binary-builds-new", "python")
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, expectedDir, bo.BaseDir)
}

func TestOutDataOmitsEmptyFields(t *testing.T) {
	data := &output.OutData{
		Version: "1.0.0",
		Source: output.OutDataSource{
			URL: "https://example.com/dep.tgz",
		},
	}

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	// Empty fields should be omitted.
	assert.NotContains(t, string(jsonData), "git_commit_sha")
	assert.NotContains(t, string(jsonData), "sub_dependencies")
	assert.NotContains(t, string(jsonData), `"sha256":""`)
}
