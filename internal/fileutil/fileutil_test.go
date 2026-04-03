package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveFileSameDevice(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	require.NoError(t, fileutil.MoveFile(src, dst))

	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err), "source file should have been removed")
}

func TestMoveFileDestinationContainsContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")

	payload := []byte("binary content \x00\x01\x02")
	require.NoError(t, os.WriteFile(src, payload, 0644))

	require.NoError(t, fileutil.MoveFile(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}
