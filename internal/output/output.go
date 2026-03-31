// Package output handles writing build output JSON files and dep-metadata JSON files.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
)

// SubDependency represents a sub-dependency with its source and version.
type SubDependency struct {
	Source  *SubDepSource `json:"source,omitempty"`
	Version string        `json:"version"`
}

// SubDepSource holds the source URL and checksum for a sub-dependency.
type SubDepSource struct {
	URL    string `json:"url,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// OutData is the canonical output data structure for a dependency build.
// It is written to both builds-artifacts JSON and dep-metadata JSON.
type OutData struct {
	Version         string                   `json:"version"`
	Source          OutDataSource            `json:"source"`
	URL             string                   `json:"url,omitempty"`
	SHA256          string                   `json:"sha256,omitempty"`
	GitCommitSHA    string                   `json:"git_commit_sha,omitempty"`
	SubDependencies map[string]SubDependency `json:"sub_dependencies,omitempty"`

	// ArtifactVersion is the version string used for the artifact filename and
	// intermediate-file lookup. It is NOT serialized to JSON. When set, it
	// overrides Version for artifact purposes only — allowing the dep-metadata
	// and builds JSON to carry the raw source version (e.g. "9.4.14.0") while
	// the artifact filename uses the full version (e.g. "9.4.14.0-ruby-3.1").
	// If empty, Version is used for both.
	ArtifactVersion string `json:"-"`

	// ArtifactFilename is the actual filename on disk produced by finalizeArtifact.
	// It is NOT serialized to JSON. It differs from filepath.Base(URL) for deps
	// whose versions contain '+' (e.g. openjdk 8.0.482+10): the URL encodes '+' as
	// '%2B' (required for S3), but the local filename uses the literal '+'.
	// build.sh uses artifact_path to locate the file — so this must be the real
	// disk name, not the URL-derived one.
	ArtifactFilename string `json:"-"`
}

// OutDataSource holds the source checksums.
// MD5 and SHA1 use pointer types so they serialize as JSON null when not set,
// matching the Ruby builder's output where unset checksum fields are null.
// SHA512 uses a plain string so an empty value is preserved as "" (Ruby outputs
// the raw depwatcher value which may be an empty string for non-applicable fields).
type OutDataSource struct {
	URL    string  `json:"url"`
	MD5    *string `json:"md5"`
	SHA256 string  `json:"sha256"`
	SHA512 string  `json:"sha512"`
	SHA1   *string `json:"sha1"`
}

// NewOutData creates an OutData from a source input.
func NewOutData(src *source.Input) *OutData {
	return &OutData{
		Version: src.Version,
		Source: OutDataSource{
			URL:    src.URL,
			MD5:    nullableString(src.MD5),
			SHA256: src.SHA256,
			SHA512: src.SHA512,
			SHA1:   nullableString(src.SHA1),
		},
	}
}

// nullableString converts an empty string to nil (JSON null) and a non-empty
// string to a pointer. This matches Ruby's behavior where unset values are nil
// and serialize as JSON null.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BuildOutput writes build JSON files and commits them to git.
type BuildOutput struct {
	BaseDir string
	Runner  runner.Runner
}

// NewBuildOutput creates a BuildOutput for the given dependency name.
// Creates the directory structure: {baseDir}/binary-builds-new/{name}/
func NewBuildOutput(name string, r runner.Runner, baseDir string) (*BuildOutput, error) {
	dir := filepath.Join(baseDir, "binary-builds-new", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating build output dir %s: %w", dir, err)
	}
	return &BuildOutput{BaseDir: dir, Runner: r}, nil
}

// AddOutput writes a JSON file with the given data and stages it for commit.
func (b *BuildOutput) AddOutput(filename string, data *OutData) error {
	path := filepath.Join(b.BaseDir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling build output: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("writing build output %s: %w", path, err)
	}

	return nil
}

// Commit stages and commits the output file with the given message.
// It skips the commit if there are no staged changes (safe-commit behaviour).
func (b *BuildOutput) Commit(msg string) error {
	if err := b.Runner.RunInDir(b.BaseDir, "git", "add", "."); err != nil {
		return fmt.Errorf("running git add: %w", err)
	}

	if err := b.Runner.RunInDir(b.BaseDir, "git", "config", "user.email", "cf-buildpacks-eng@pivotal.io"); err != nil {
		return fmt.Errorf("setting git email: %w", err)
	}
	if err := b.Runner.RunInDir(b.BaseDir, "git", "config", "user.name", "CF Buildpacks Team CI Server"); err != nil {
		return fmt.Errorf("setting git name: %w", err)
	}

	// safe_commit: only commit when there are staged changes.
	// git diff --cached --quiet exits 0 when nothing is staged, 1 when changes exist.
	// RunInDir returns an error on non-zero exit, so a nil error means nothing to commit.
	if err := b.Runner.RunInDir(b.BaseDir, "git", "diff", "--cached", "--quiet"); err == nil {
		// Exit 0: no staged changes — nothing to commit.
		return nil
	}

	return b.Runner.RunInDir(b.BaseDir, "git", "commit", "-m", msg)
}

// DepMetadataOutput writes dep-metadata JSON files.
type DepMetadataOutput struct {
	BaseDir string
}

// NewDepMetadataOutput creates a DepMetadataOutput.
func NewDepMetadataOutput(baseDir string) *DepMetadataOutput {
	return &DepMetadataOutput{BaseDir: baseDir}
}

// WriteMetadata writes the metadata JSON file for a dependency artifact.
// The filename is "{artifactFilename}_metadata.json".
func (d *DepMetadataOutput) WriteMetadata(artifactFilename string, data *OutData) error {
	basename := filepath.Base(artifactFilename)
	metadataFilename := basename + "_metadata.json"
	path := filepath.Join(d.BaseDir, metadataFilename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling dep metadata: %w", err)
	}

	return os.WriteFile(path, jsonData, 0644)
}
