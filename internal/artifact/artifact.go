// Package artifact handles artifact naming, SHA256 computation, and S3 URL construction.
package artifact

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// extensions is the single source of truth for the dependency file formats
// binary-builder understands. Each entry pairs the on-disk suffix of a file
// with the extension it is published under.
//
// Everything else derives from this table:
//
//   - SearchSuffixes → findIntermediateArtifact's glob patterns (which file a
//     recipe produced)
//   - SuffixFromPath → intermediate file naming (e.g. PassthroughRecipe)
//   - ExtFromPath    → the published artifact filename
//
// Adding a new format (say ".cjs") is therefore a one-line change here that
// takes effect in all three places at once. Before this table was shared, a
// format could be added to the intermediate-file search but forgotten when
// naming the published artifact, which is how yarn-berry's raw yarn.js was
// published as a ".tgz" that consumers could not gunzip.
//
// Order matters for findIntermediateArtifact, which globs in this order.
var extensions = []struct {
	suffix string // on-disk suffix, with dot, compound suffixes first-class
	ext    string // published extension, no dot
}{
	{".tgz", "tgz"},
	{".tar.gz", "tgz"},
	{".zip", "zip"},
	{".tar.xz", "tar.xz"},
	{".tar.bz2", "tar.bz2"},
	{".sh", "sh"},
	{".phar", "phar"},
	{".jar", "jar"},
	{".js", "js"},
	{".txt", "txt"},
}

// ExtFromPath extracts the published file extension from a path, normalizing
// "tar.gz" to "tgz" to match the existing artifact naming convention.
func ExtFromPath(path string) string {
	base := strings.ToLower(path)

	for _, e := range extensions {
		if strings.HasSuffix(base, e.suffix) {
			return e.ext
		}
	}

	return "tgz" // default
}

// SuffixFromPath returns the on-disk suffix of path, with the leading dot and
// compound suffixes preserved (".tar.gz", ".js"). Use this when naming
// intermediate files; unlike ExtFromPath it does not normalize to the
// published extension. Suffixes not in the table fall back to filepath.Ext.
//
// Matching is case-insensitive, so the returned suffix is always lowercase
// ("foo.TAR.GZ" → ".tar.gz").
func SuffixFromPath(path string) string {
	base := strings.ToLower(path)

	for _, e := range extensions {
		if strings.HasSuffix(base, e.suffix) {
			return e.suffix
		}
	}

	return filepath.Ext(base)
}

// SearchSuffixes returns the on-disk suffixes (without the leading dot) that
// findIntermediateArtifact globs for, in priority order.
func SearchSuffixes() []string {
	suffixes := make([]string, 0, len(extensions))
	for _, e := range extensions {
		suffixes = append(suffixes, strings.TrimPrefix(e.suffix, "."))
	}
	return suffixes
}
