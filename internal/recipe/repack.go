package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// RepackRecipe downloads an upstream archive and optionally transforms it.
// It covers recipes that follow the pattern:
//
//  1. Infer or compute a local destination filename
//  2. Fetcher.Download(ctx, src.URL, dest, checksum)
//  3. [optional] archive.StripTopLevelDir(dest)  — for .tar.gz / .tgz
//     or archive.StripTopLevelDirFromZip(dest)   — for .zip
//  4. [optional] update outData.Version with the stripped version
type RepackRecipe struct {
	DepName string
	Meta    ArtifactMeta
	Fetcher fetch.Fetcher
	// StripTopLevelDir strips the top-level directory from the archive after download.
	// For .zip files the zip-specific stripper is used automatically.
	StripTopLevelDir bool
	// StripVersionPrefix strips this prefix from src.Version before building the
	// destination filename and writing outData.Version (e.g. "v" for yarn).
	StripVersionPrefix string
	// DestFilename derives the local destination filename from version and URL.
	// If nil, the default is "<depname>-<version>.<ext inferred from URL>".
	// PyPI sdist recipes use this to infer the filename from the URL's last path segment.
	DestFilename func(version, url string) string
}

func (r *RepackRecipe) Name() string           { return r.DepName }
func (r *RepackRecipe) Artifact() ArtifactMeta { return r.Meta }

func (r *RepackRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, _ runner.Runner, outData *output.OutData) error {
	version := strings.TrimPrefix(src.Version, r.StripVersionPrefix)
	if r.StripVersionPrefix != "" {
		outData.Version = version
	}

	var dest string
	if r.DestFilename != nil {
		dest = filepath.Join(os.TempDir(), r.DestFilename(version, src.URL))
	} else {
		ext := inferExt(src.URL)
		dest = filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s%s", r.DepName, version, ext))
	}

	if err := r.Fetcher.Download(ctx, src.URL, dest, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("downloading %s: %w", r.DepName, err)
	}

	if !r.StripTopLevelDir {
		return nil
	}

	// Use dest (already fragment-free) rather than src.URL to detect zip archives.
	// PyPI download URLs may contain a #sha256=… fragment that would fool a
	// suffix check on the raw URL.
	if strings.HasSuffix(dest, ".zip") {
		return archive.StripTopLevelDirFromZip(dest)
	}
	return archive.StripTopLevelDir(dest)
}

// inferExt returns the file extension for a download URL, recognising .tar.gz
// as a two-part extension and falling back to the last path segment's suffix.
// Any URL fragment (e.g. #sha256=…) is stripped before inspection.
func inferExt(rawURL string) string {
	// Strip fragment (e.g. "#sha256=abc123") before inspecting the path.
	u := rawURL
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	if strings.HasSuffix(u, ".tar.gz") {
		return ".tar.gz"
	}
	parts := strings.Split(u, "/")
	last := parts[len(parts)-1]
	idx := strings.LastIndex(last, ".")
	if idx < 0 {
		return ""
	}
	return last[idx:]
}
