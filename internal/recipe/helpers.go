package recipe

import (
	"crypto/sha256"
	"fmt"
)

// computeSHA256 returns the hex-encoded SHA256 of the given data.
func computeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// SourceEntry is one entry in a sources.yml file.
type SourceEntry struct {
	URL    string
	SHA256 string // hex SHA256 of the source tarball
}

// buildSourcesYAML returns the content of a sources.yml file matching the
// format produced by Ruby's YAMLPresenter#to_yaml:
//
//	---
//	- url: https://...
//	  sha256: abc123...
//
// The returned bytes are intended to be injected into an artifact tarball
// via archive.InjectFile, mirroring what ArchiveRecipe#compress! does by
// writing sources.yml into the tmpdir alongside the archive_files before tar.
func buildSourcesYAML(entries []SourceEntry) []byte {
	content := "---\n"
	for _, e := range entries {
		content += fmt.Sprintf("- url: %s\n  sha256: %s\n", e.URL, e.SHA256)
	}
	return []byte(content)
}
