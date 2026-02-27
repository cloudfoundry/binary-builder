package recipe_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// callNames returns the command name for every recorded call.
func callNames(calls []runner.Call) []string {
	names := make([]string, len(calls))
	for i, c := range calls {
		names[i] = c.Name
	}
	return names
}

// anyCallContains returns true if any call's Name equals name.
func anyCallContains(calls []runner.Call, name string) bool {
	for _, c := range calls {
		if c.Name == name {
			return true
		}
	}
	return false
}

// anyArgsContain returns true if any call has an argument that contains target.
func anyArgsContain(calls []runner.Call, target string) bool {
	for _, c := range calls {
		for _, arg := range c.Args {
			if strings.Contains(arg, target) {
				return true
			}
		}
	}
	return false
}

// hasCallMatching returns true if any call matches name and (optionally) has argSubstr in its joined args.
func hasCallMatching(calls []runner.Call, name string, argSubstr string) bool {
	for _, c := range calls {
		if c.Name == name {
			joined := strings.Join(c.Args, " ")
			if argSubstr == "" || strings.Contains(joined, argSubstr) {
				return true
			}
		}
	}
	return false
}

// hasDownload returns true if the fetcher recorded a download with the exact URL.
func hasDownload(f *FakeFetcher, url string) bool {
	for _, dl := range f.DownloadedURLs {
		if dl.URL == url {
			return true
		}
	}
	return false
}

// hasDownloadContaining returns true if any downloaded URL contains substr.
func hasDownloadContaining(f *FakeFetcher, substr string) bool {
	for _, dl := range f.DownloadedURLs {
		if strings.Contains(dl.URL, substr) {
			return true
		}
	}
	return false
}

// hasCallWithEnv returns true if any call matches name and has envKey in its Env map.
func hasCallWithEnv(calls []runner.Call, name string, envKey string) bool {
	for _, c := range calls {
		if c.Name == name && c.Env != nil {
			if _, ok := c.Env[envKey]; ok {
				return true
			}
		}
	}
	return false
}

// useTempWorkDir switches the process working directory to a fresh temp dir for
// the duration of the test and restores it afterwards.  It also creates the
// artifacts/ sub-directory so recipes that write relative artifact paths don't
// fail on directory-not-found before they even try to read the file.
//
// NOT safe to use in parallel sub-tests (os.Chdir is process-global).
func useTempWorkDir(t *testing.T) string {
	t.Helper()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("useTempWorkDir: getwd: %v", err)
	}

	tmp := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmp, "artifacts"), 0755); err != nil {
		t.Fatalf("useTempWorkDir: mkdir artifacts: %v", err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("useTempWorkDir: chdir to %s: %v", tmp, err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	return tmp
}

// writeFakeArtifact creates a minimal valid .tgz at <name> in the current
// working directory.  The tarball contains a single dummy file so that
// archive.StripTopLevelDir / StripIncorrectWordsYAML don't fail.
// Recipes write artifacts to mustCwd()/<name> (CWD root, not artifacts/).
func writeFakeArtifact(t *testing.T, name string) {
	t.Helper()

	path := name // write directly into CWD, matching mustCwd() usage in recipes
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("writeFakeArtifact: create %s: %v", path, err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write a single dummy entry so the tarball is valid.
	dummy := []byte("dummy")
	hdr := &tar.Header{
		Name:     "dummy-dir/dummy-file",
		Mode:     0644,
		Size:     int64(len(dummy)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("writeFakeArtifact: write header: %v", err)
	}
	if _, err := tw.Write(dummy); err != nil {
		t.Fatalf("writeFakeArtifact: write data: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("writeFakeArtifact: close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("writeFakeArtifact: close gzip: %v", err)
	}
}
