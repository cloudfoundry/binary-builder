package archive_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func listTarEntries(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
	sort.Strings(names)
	return names
}

func TestStripTopLevelDir(t *testing.T) {
	files := map[string]string{
		"node-v20.11.0/bin/node":    "binary",
		"node-v20.11.0/lib/libv8.a": "library",
		"node-v20.11.0/README.md":   "readme",
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tgz")
	require.NoError(t, os.WriteFile(path, createTestTarball(t, files), 0644))

	err := archive.StripTopLevelDir(path)
	require.NoError(t, err)

	entries := listTarEntries(t, path)
	// Expect "./" root entry, explicit dir entries, then "./" prefixed files —
	// matching `tar -czf out.tgz -C dir .` output (what Ruby builder produces).
	assert.Equal(t, []string{"./", "./README.md", "./bin/", "./bin/node", "./lib/", "./lib/libv8.a"}, entries)
}

func TestStripTopLevelDirDotSlashPrefix(t *testing.T) {
	// Simulate `tar czf out.tgz -C destDir .` where destDir contains `nginx/`.
	// This produces entries like `./nginx/sbin/nginx` — the `./` is NOT a component
	// to strip; `nginx/` is the real top-level dir.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	tw.WriteHeader(&tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0755})
	tw.WriteHeader(&tar.Header{Name: "./nginx/", Typeflag: tar.TypeDir, Mode: 0755})
	tw.WriteHeader(&tar.Header{Name: "./nginx/sbin/", Typeflag: tar.TypeDir, Mode: 0755})
	tw.WriteHeader(&tar.Header{Name: "./nginx/sbin/nginx", Typeflag: tar.TypeReg, Mode: 0755, Size: 6})
	tw.Write([]byte("binary"))
	tw.WriteHeader(&tar.Header{Name: "./nginx/conf/", Typeflag: tar.TypeDir, Mode: 0755})
	tw.WriteHeader(&tar.Header{Name: "./nginx/modules/", Typeflag: tar.TypeDir, Mode: 0755})

	tw.Close()
	gw.Close()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tgz")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))

	err := archive.StripTopLevelDir(path)
	require.NoError(t, err)

	entries := listTarEntries(t, path)
	// After stripping `nginx/`, expect sbin/, conf/, modules/ at top-level.
	assert.Contains(t, entries, "./sbin/nginx")
	assert.NotContains(t, entries, "./nginx/")
	assert.NotContains(t, entries, "./nginx/sbin/nginx")
}

func TestStripTopLevelDirSkipsTopDir(t *testing.T) {
	// Include the directory entry itself.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Directory entry.
	tw.WriteHeader(&tar.Header{Name: "top/", Typeflag: tar.TypeDir, Mode: 0755})
	// File entry.
	tw.WriteHeader(&tar.Header{Name: "top/file.txt", Typeflag: tar.TypeReg, Mode: 0644, Size: 5})
	tw.Write([]byte("hello"))

	tw.Close()
	gw.Close()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tgz")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))

	err := archive.StripTopLevelDir(path)
	require.NoError(t, err)

	entries := listTarEntries(t, path)
	// Expect "./" root entry + "./"-prefixed file, matching Ruby tar output.
	assert.Equal(t, []string{"./", "./file.txt"}, entries)
}

func TestStripFiles(t *testing.T) {
	files := map[string]string{
		"bin/ruby":                 "binary",
		"lib/ruby/gems/foo.rb":     "gem",
		"incorrect_words.yaml":     "should be removed",
		"lib/incorrect_words.yaml": "also removed",
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tgz")
	require.NoError(t, os.WriteFile(path, createTestTarball(t, files), 0644))

	err := archive.StripFiles(path, "incorrect_words.yaml")
	require.NoError(t, err)

	entries := listTarEntries(t, path)
	assert.Contains(t, entries, "bin/ruby")
	assert.Contains(t, entries, "lib/ruby/gems/foo.rb")
	assert.NotContains(t, entries, "incorrect_words.yaml")
	assert.NotContains(t, entries, "lib/incorrect_words.yaml")
}

func TestStripIncorrectWordsYAML(t *testing.T) {
	files := map[string]string{
		"bin/ruby":             "binary",
		"incorrect_words.yaml": "should be removed",
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.tgz")
	require.NoError(t, os.WriteFile(path, createTestTarball(t, files), 0644))

	err := archive.StripIncorrectWordsYAML(path)
	require.NoError(t, err)

	entries := listTarEntries(t, path)
	assert.Contains(t, entries, "bin/ruby")
	assert.NotContains(t, entries, "incorrect_words.yaml")
}

func TestStripTopLevelDirFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.zip")

	// Create a zip with top-level directory.
	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	fw, err := w.Create("setuptools-69.0.3/setup.py")
	require.NoError(t, err)
	fw.Write([]byte("setup code"))

	fw2, err := w.Create("setuptools-69.0.3/README.md")
	require.NoError(t, err)
	fw2.Write([]byte("readme"))

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	err = archive.StripTopLevelDirFromZip(path)
	require.NoError(t, err)

	// Verify the zip contents.
	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer r.Close()

	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	assert.Equal(t, []string{"README.md", "setup.py"}, names)
}

func TestPackUsesRunner(t *testing.T) {
	f := runner.NewFakeRunner()

	err := archive.Pack(f, "/tmp/out.tgz", "/tmp/src", "mydir", nil)
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "tar", f.Calls[0].Name)
	assert.Contains(t, f.Calls[0].Args, "czf")
	assert.Contains(t, f.Calls[0].Args, "/tmp/out.tgz")
	assert.Contains(t, f.Calls[0].Args, "mydir")
}

func TestPackFlatUsesRunner(t *testing.T) {
	f := runner.NewFakeRunner()

	err := archive.Pack(f, "/tmp/out.tgz", "/tmp/src", "", nil)
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Contains(t, f.Calls[0].Args, ".")
}

func TestPackWithDereferenceUsesRunner(t *testing.T) {
	f := runner.NewFakeRunner()

	err := archive.PackWithDereference(f, "/tmp/out.tgz", "/tmp/src")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "tar", f.Calls[0].Name)
	assert.Contains(t, f.Calls[0].Args, "--hard-dereference")
	assert.Equal(t, "/tmp/src", f.Calls[0].Dir)
}

func TestPackXZUsesRunner(t *testing.T) {
	f := runner.NewFakeRunner()

	err := archive.PackXZ(f, "/tmp/out.tar.xz", "/tmp/src")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "tar", f.Calls[0].Name)
	assert.Contains(t, f.Calls[0].Args, "-Jcf")
	assert.Equal(t, "/tmp/src", f.Calls[0].Dir)
}

func TestPackZipUsesRunner(t *testing.T) {
	f := runner.NewFakeRunner()

	err := archive.PackZip(f, "/tmp/out.zip", "/tmp/src")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "zip", f.Calls[0].Name)
	assert.Equal(t, "/tmp/src", f.Calls[0].Dir)
}
