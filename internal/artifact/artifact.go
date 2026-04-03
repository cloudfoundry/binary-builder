// Package artifact handles artifact naming, SHA256 computation, and S3 URL construction.
package artifact

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

const s3BaseURL = "https://buildpacks.cloudfoundry.org/dependencies"

// Artifact represents a built dependency artifact with its naming components.
type Artifact struct {
	Name    string // dependency name, e.g. "ruby", "php"
	Version string // version string, e.g. "3.3.6", "11.0.22+7"
	OS      string // "linux" or "windows"
	Arch    string // "x64", "noarch", "x86-64"
	Stack   string // "cflinuxfs4", "cflinuxfs5", "any-stack"
}

// Filename returns the canonical artifact filename:
// "name_version_os_arch_stack_sha256prefix.ext"
//
// The sha256 parameter is the full hex-encoded SHA256 of the artifact file.
// Only the first 8 characters are used in the filename.
func (a Artifact) Filename(sha256hex string, ext string) string {
	prefix := a.FilenamePrefix()
	sha8 := sha256hex
	if len(sha8) > 8 {
		sha8 = sha8[:8]
	}
	return fmt.Sprintf("%s_%s.%s", prefix, sha8, ext)
}

// FilenamePrefix returns the artifact filename without the SHA prefix and extension:
// "name_version_os_arch_stack"
func (a Artifact) FilenamePrefix() string {
	return fmt.Sprintf("%s_%s_%s_%s_%s", a.Name, a.Version, a.OS, a.Arch, a.Stack)
}

// S3URL returns the canonical S3 URL for the artifact.
//
// The filename is URL-safe-encoded: '+' is replaced with '%2B' to prevent
// AWS S3 permission denied errors (S3 interprets unencoded '+' as space).
// See: https://github.com/cloudfoundry/buildpacks-ci/pull/553
func (a Artifact) S3URL(filename string) string {
	encoded := strings.ReplaceAll(filename, "+", "%2B")
	return fmt.Sprintf("%s/%s/%s", s3BaseURL, a.Name, encoded)
}

// SHA256File computes the SHA256 hex digest of a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s for SHA256: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("reading %s for SHA256: %w", path, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ExtFromPath extracts the file extension from a path, normalizing
// "tar.gz" to "tgz" to match the existing artifact naming convention.
func ExtFromPath(path string) string {
	base := strings.ToLower(path)

	extensions := []struct {
		suffix string
		ext    string
	}{
		{".tar.gz", "tgz"},
		{".tgz", "tgz"},
		{".tar.xz", "tar.xz"},
		{".tar.bz2", "tar.bz2"},
		{".zip", "zip"},
		{".phar", "phar"},
		{".sh", "sh"},
		{".txt", "txt"},
	}

	for _, e := range extensions {
		if strings.HasSuffix(base, e.suffix) {
			return e.ext
		}
	}

	return "tgz" // default
}
