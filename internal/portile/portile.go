// Package portile provides a Go equivalent of mini_portile2.
// It manages the download → extract → configure → compile → install
// lifecycle for autoconf-based software (./configure && make && make install).
package portile

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
)

// Portile manages the build lifecycle for an autoconf-based dependency.
type Portile struct {
	Name    string
	Version string
	URL     string

	// Checksum for verifying the downloaded source.
	Checksum source.Checksum

	// Prefix is the --prefix= value for ./configure.
	// Defaults to TmpPath()/port if empty.
	Prefix string

	// Options are extra flags passed to ./configure after --prefix.
	Options []string

	// Jobs is the -j flag for make. Defaults to 4.
	Jobs int

	// ExtractedDirName overrides the assumed "{name}-{version}" directory name
	// that the tarball extracts to. Use this when the tarball extracts to a
	// differently-named directory (e.g. node tarballs extract to "node-v{version}"
	// rather than "node-{version}").
	// When empty, defaults to "{name}-{version}".
	ExtractedDirName string

	// InstallArgs are extra arguments appended to "make install".
	// Used for recipes that need DESTDIR=... or PORTABLE=1 etc.
	InstallArgs []string

	Runner  runner.Runner
	Fetcher fetch.Fetcher
}

// TmpPath returns the temporary build directory:
// /tmp/{arch}/ports/{name}/{version}
func (p *Portile) TmpPath() string {
	return filepath.Join("/tmp", runtime.GOARCH, "ports", p.Name, p.Version)
}

// PortPath returns the extracted source directory inside TmpPath.
func (p *Portile) PortPath() string {
	return filepath.Join(p.TmpPath(), "port")
}

func (p *Portile) prefix() string {
	if p.Prefix != "" {
		return p.Prefix
	}
	return p.PortPath()
}

func (p *Portile) jobs() int {
	if p.Jobs > 0 {
		return p.Jobs
	}
	return 4
}

// Cook performs the full build lifecycle: download, extract, configure, compile, install.
func (p *Portile) Cook(ctx context.Context) error {
	if err := p.download(ctx); err != nil {
		return fmt.Errorf("portile %s %s download: %w", p.Name, p.Version, err)
	}

	if err := p.extract(ctx); err != nil {
		return fmt.Errorf("portile %s %s extract: %w", p.Name, p.Version, err)
	}

	if err := p.configure(ctx); err != nil {
		return fmt.Errorf("portile %s %s configure: %w", p.Name, p.Version, err)
	}

	if err := p.compile(ctx); err != nil {
		return fmt.Errorf("portile %s %s compile: %w", p.Name, p.Version, err)
	}

	if err := p.install(ctx); err != nil {
		return fmt.Errorf("portile %s %s install: %w", p.Name, p.Version, err)
	}

	return nil
}

func (p *Portile) tarballPath() string {
	// Extract filename from URL.
	parts := strings.Split(p.URL, "/")
	filename := parts[len(parts)-1]
	// Strip query parameters.
	if idx := strings.Index(filename, "?"); idx >= 0 {
		filename = filename[:idx]
	}
	return filepath.Join(p.TmpPath(), filename)
}

func (p *Portile) download(ctx context.Context) error {
	// Create the tmp directory.
	if err := p.Runner.Run("mkdir", "-p", p.TmpPath()); err != nil {
		return err
	}

	return p.Fetcher.Download(ctx, p.URL, p.tarballPath(), p.Checksum)
}

func (p *Portile) extractedDirName() string {
	if p.ExtractedDirName != "" {
		return p.ExtractedDirName
	}
	return fmt.Sprintf("%s-%s", p.Name, p.Version)
}

// ExtractFlag returns an explicit tar compression flag derived from the
// filename extension. This avoids relying on tar's auto-detect heuristic,
// which misidentifies .tar.gz files as zstd-compressed in some environments
// (e.g. cflinuxfs4) where zstd is not installed.
func ExtractFlag(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "xzf"
	case strings.HasSuffix(lower, ".tar.bz2"):
		return "xjf"
	case strings.HasSuffix(lower, ".tar.xz"):
		return "xJf"
	default:
		return "xf"
	}
}

func (p *Portile) extract(_ context.Context) error {
	srcDir := filepath.Join(p.TmpPath(), p.extractedDirName())

	flag := ExtractFlag(p.tarballPath())
	if err := p.Runner.Run("tar", flag, p.tarballPath(), "-C", p.TmpPath()); err != nil {
		return err
	}

	// mini_portile2 renames the extracted directory to "port".
	return p.Runner.Run("mv", srcDir, p.PortPath())
}

func (p *Portile) configure(_ context.Context) error {
	args := []string{fmt.Sprintf("--prefix=%s", p.prefix())}
	args = append(args, p.Options...)

	return p.Runner.RunInDir(p.PortPath(), "./configure", args...)
}

func (p *Portile) compile(_ context.Context) error {
	return p.Runner.RunInDir(p.PortPath(), "make", fmt.Sprintf("-j%d", p.jobs()))
}

func (p *Portile) install(_ context.Context) error {
	args := []string{"install"}
	args = append(args, p.InstallArgs...)
	return p.Runner.RunInDir(p.PortPath(), "make", args...)
}
