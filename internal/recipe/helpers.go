package recipe

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
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

// fileSHA256 returns the hex-encoded SHA256 digest of the file at path.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// mustCwd returns the current working directory, panicking on error.
func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("recipe: getting cwd: %v", err))
	}
	return cwd
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
