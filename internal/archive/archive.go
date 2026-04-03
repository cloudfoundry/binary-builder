// Package archive provides helpers for creating and manipulating tar/zip archives.
package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// Pack creates a gzipped tarball at outputPath from the given directory.
// pathName is the top-level directory name inside the tarball ("" for flat pack of dir contents).
func Pack(r runner.Runner, outputPath, sourceDir, pathName string) error {
	args := []string{"czf", outputPath}
	if pathName != "" {
		args = append(args, "-C", sourceDir, pathName)
	} else {
		args = append(args, "-C", sourceDir, ".")
	}

	return r.Run("tar", args...)
}

// PackWithDereference creates a gzipped tarball with --hard-dereference.
// Used by the Python recipe to resolve symlinks.
func PackWithDereference(r runner.Runner, outputPath, sourceDir string) error {
	return r.RunInDir(sourceDir, "tar", "zcvf", outputPath, "--hard-dereference", ".")
}

// PackXZ creates an xz-compressed tarball. Used by dotnet recipes.
func PackXZ(r runner.Runner, outputPath, sourceDir string) error {
	return r.RunInDir(sourceDir, "tar", "-Jcf", outputPath, ".")
}

// PackZip creates a zip archive from the given directory.
func PackZip(r runner.Runner, outputPath, sourceDir string) error {
	return r.RunInDir(sourceDir, "zip", outputPath, "-r", ".")
}

// StripTopLevelDir re-archives a tarball without its top-level directory.
// e.g. "node-v20.11.0/bin/node" → "./bin/node"
//
// The output uses "./" prefixed paths and includes directory entries,
// matching the output of `tar -czf out.tgz -C dir .` which is what
// the Ruby builder produces.
//
// Strategy: extract to a temp dir, then re-archive with `tar -C dir .`
// so that directory entries are synthesised for every subdirectory.
func StripTopLevelDir(path string) error {
	// Step 1: extract the tarball into a temp directory, stripping the top-level dir.
	tmpDir, err := os.MkdirTemp("", "strip-top-level-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("opening gzip %s: %w", path, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry from %s: %w", path, err)
		}

		// Strip the first real path component.
		// Entries may start with "./" (e.g. produced by `tar -C dir .`), so
		// normalise by removing any leading "./" before splitting.
		name := strings.TrimPrefix(hdr.Name, "./")
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			// Top-level directory entry itself — skip.
			continue
		}
		stripped := parts[1]

		target := filepath.Join(tmpDir, filepath.FromSlash(stripped))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0700); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent of %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0600)
			if err != nil {
				return fmt.Errorf("creating %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("writing %s: %w", target, err)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent of symlink %s: %w", target, err)
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil && !os.IsExist(err) {
				return fmt.Errorf("symlink %s → %s: %w", target, hdr.Linkname, err)
			}
		case tar.TypeLink:
			// Strip the top-level directory from the hard link target too.
			linkName := strings.TrimPrefix(hdr.Linkname, "./")
			linkParts := strings.SplitN(linkName, "/", 2)
			var strippedLink string
			if len(linkParts) >= 2 {
				strippedLink = linkParts[1]
			} else {
				strippedLink = linkName
			}
			linkTarget := filepath.Join(tmpDir, filepath.FromSlash(strippedLink))
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent of hardlink %s: %w", target, err)
			}
			if err := os.Link(linkTarget, target); err != nil && !os.IsExist(err) {
				return fmt.Errorf("hardlink %s → %s: %w", target, linkTarget, err)
			}
		}
	}

	// Step 2: re-archive with `tar -czf <output> -C <tmpDir> .`
	// This produces "./" prefixed paths and emits directory entries for every
	// subdirectory — identical to what Ruby's Archive.strip_top_level_directory_from_tar does.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	cmd := exec.Command("tar", "-czf", absPath, "-C", tmpDir, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("re-archiving %s: %w", path, err)
	}

	return nil
}

// StripTopLevelDirFromZip re-archives a zip without its top-level directory.
// Used by setuptools which may ship as .zip.
func StripTopLevelDirFromZip(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("opening zip %s: %w", path, err)
	}
	defer r.Close()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}

		newHeader := f.FileHeader
		newHeader.Name = parts[1]

		writer, err := w.CreateHeader(&newHeader)
		if err != nil {
			return fmt.Errorf("creating zip entry %s: %w", newHeader.Name, err)
		}

		if !f.FileInfo().IsDir() {
			reader, err := f.Open()
			if err != nil {
				return fmt.Errorf("opening zip entry %s: %w", f.Name, err)
			}
			if _, err := io.Copy(writer, reader); err != nil {
				reader.Close()
				return fmt.Errorf("copying zip entry %s: %w", f.Name, err)
			}
			reader.Close()
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("closing zip writer: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// StripFiles removes files matching a glob pattern from inside a tarball.
func StripFiles(path string, pattern string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("opening gzip %s: %w", path, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		matched, _ := filepath.Match(pattern, filepath.Base(hdr.Name))
		if matched {
			// Skip this entry — strip it from the archive.
			if hdr.Typeflag == tar.TypeReg {
				io.Copy(io.Discard, tr)
			}
			continue
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing header: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := io.Copy(tw, tr); err != nil {
				return fmt.Errorf("copying data: %w", err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// StripIncorrectWordsYAML removes incorrect_words.yaml from a tarball
// and from any nested .jar files within it.
// Used by ruby and jruby recipes.
func StripIncorrectWordsYAML(path string) error {
	return StripFiles(path, "incorrect_words.yaml")
}

// InjectFile adds a file with the given name and content into an existing
// gzipped tarball. The file is appended at the archive root (no directory
// prefix). Typically used to inject sources.yml into an artifact tarball.
func InjectFile(tarPath, filename string, content []byte) error {
	data, err := os.ReadFile(tarPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", tarPath, err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("opening gzip %s: %w", tarPath, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Copy all existing entries.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing header: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			if _, err := io.Copy(tw, tr); err != nil {
				return fmt.Errorf("copying data: %w", err)
			}
		}
	}

	// Append the new file at the archive root.
	hdr := &tar.Header{
		Name:     "./" + filename,
		Mode:     0644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing injected header: %w", err)
	}
	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("writing injected content: %w", err)
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	return os.WriteFile(tarPath, buf.Bytes(), 0644)
}
