// Package source parses the Concourse source/data.json resource,
// handling both legacy and modern JSON formats.
package source

import (
	"encoding/json"
	"fmt"
	"os"
)

// Checksum represents a hash algorithm and its expected value.
type Checksum struct {
	Algorithm string // "sha256", "sha512", "md5", "sha1"
	Value     string
}

// Input represents the parsed source/data.json from a Concourse resource.
type Input struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Version      string `json:"version"`
	MD5          string `json:"md5"`
	SHA256       string `json:"sha256"`
	SHA512       string `json:"sha512"`
	SHA1         string `json:"sha1"`
	GitCommitSHA string `json:"git_commit_sha"`
	Repo         string `json:"repo"`
	Type         string `json:"type"`
}

// legacyInput handles the older JSON format with different field names.
type legacyInput struct {
	Name      string `json:"name"`
	SourceURI string `json:"source_uri"`
	Version   string `json:"version"`
	SourceSHA string `json:"source_sha"`
	Repo      string `json:"repo"`
	Type      string `json:"type"`
}

// modernInput handles the depwatcher JSON format with nested source + version objects:
//
//	{
//	  "source": {"name": "composer", "type": "github_releases", "repo": "composer/composer"},
//	  "version": {"url": "https://...", "ref": "2.7.1", "sha256": "...", "sha512": "..."}
//	}
type modernInput struct {
	Source struct {
		Name string `json:"name"`
		Repo string `json:"repo"`
		Type string `json:"type"`
	} `json:"source"`
	Version struct {
		URL          string `json:"url"`
		Ref          string `json:"ref"`
		SHA256       string `json:"sha256"`
		SHA512       string `json:"sha512"`
		MD5          string `json:"md5_digest"`
		SHA1         string `json:"sha1"`
		GitCommitSHA string `json:"git_commit_sha"`
	} `json:"version"`
}

// FromFile reads and parses a source data.json file.
// It auto-detects the format (legacy vs modern) based on the presence
// of a "source" key or "source_uri" key.
func FromFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading source file %q: %w", path, err)
	}

	return Parse(data)
}

// Parse parses source JSON data, auto-detecting the format.
func Parse(data []byte) (*Input, error) {
	// Try to detect format by checking for key fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing source JSON: %w", err)
	}

	// Modern format has a "source" object.
	if _, hasSource := raw["source"]; hasSource {
		return parseModern(data)
	}

	// Legacy format has "source_uri".
	if _, hasSourceURI := raw["source_uri"]; hasSourceURI {
		return parseLegacy(data)
	}

	// Fallback: try modern first, then legacy.
	if input, err := parseModern(data); err == nil && input.URL != "" {
		return input, nil
	}

	return parseLegacy(data)
}

func parseModern(data []byte) (*Input, error) {
	var m modernInput
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing modern source format: %w", err)
	}

	return &Input{
		Name:         m.Source.Name,
		URL:          m.Version.URL,
		Version:      m.Version.Ref,
		SHA256:       m.Version.SHA256,
		SHA512:       m.Version.SHA512,
		MD5:          m.Version.MD5,
		SHA1:         m.Version.SHA1,
		GitCommitSHA: m.Version.GitCommitSHA,
		Repo:         m.Source.Repo,
		Type:         m.Source.Type,
	}, nil
}

func parseLegacy(data []byte) (*Input, error) {
	var l legacyInput
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parsing legacy source format: %w", err)
	}

	return &Input{
		Name:    l.Name,
		URL:     l.SourceURI,
		Version: l.Version,
		MD5:     l.SourceSHA, // legacy format uses source_sha for MD5
		Repo:    l.Repo,
		Type:    l.Type,
	}, nil
}

// PrimaryChecksum returns the strongest available checksum.
// Preference order: SHA512 > SHA256 > MD5 > SHA1.
func (i *Input) PrimaryChecksum() Checksum {
	if i.SHA512 != "" {
		return Checksum{Algorithm: "sha512", Value: i.SHA512}
	}
	if i.SHA256 != "" {
		return Checksum{Algorithm: "sha256", Value: i.SHA256}
	}
	if i.MD5 != "" {
		return Checksum{Algorithm: "md5", Value: i.MD5}
	}
	if i.SHA1 != "" {
		return Checksum{Algorithm: "sha1", Value: i.SHA1}
	}
	return Checksum{}
}
