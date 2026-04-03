// Package fileutil provides file-system utilities shared across packages.
package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

// MoveFile moves src to dst. It tries os.Rename first; if that fails with a
// cross-device link error (EXDEV) it falls back to copy-then-delete so that
// moves across filesystem boundaries succeed.
func MoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	// Cross-device: copy then delete.
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing destination: %w", err)
	}
	return os.Remove(src)
}
