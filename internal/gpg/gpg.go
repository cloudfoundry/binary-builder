// Package gpg provides GPG signature verification for downloaded files.
// Used by nginx and nginx-static recipes.
package gpg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// VerifySignature downloads a file and its .asc signature, imports all
// public keys, and runs gpg --verify. Returns an error if verification fails.
//
// If gpg is not found in PATH, it is installed via apt-get (matching the
// behavior of the Ruby builder's GPGHelper.verify_gpg_signature).
func VerifySignature(ctx context.Context, fileURL, signatureURL string, publicKeyURLs []string, r runner.Runner) error {
	// Install gpg if not present — mirrors Ruby's GPGHelper behaviour.
	if _, err := exec.LookPath("gpg"); err != nil {
		if err := r.Run("apt-get", "update"); err != nil {
			return fmt.Errorf("apt-get update before gpg install: %w", err)
		}
		if err := r.Run("apt-get", "install", "-y", "gpg"); err != nil {
			return fmt.Errorf("installing gpg: %w", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "gpg-verify-*")
	if err != nil {
		return fmt.Errorf("creating temp dir for GPG verification: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download each public key and import it.
	for i, keyURL := range publicKeyURLs {
		keyPath := filepath.Join(tmpDir, fmt.Sprintf("key-%d.asc", i))
		if err := r.Run("wget", "-q", "-O", keyPath, keyURL); err != nil {
			return fmt.Errorf("downloading GPG key %s: %w", keyURL, err)
		}
		if err := r.Run("gpg", "--import", keyPath); err != nil {
			return fmt.Errorf("importing GPG key %s: %w", keyURL, err)
		}
	}

	// Download the file and its signature.
	filePath := filepath.Join(tmpDir, "file")
	sigPath := filepath.Join(tmpDir, "file.asc")

	if err := r.Run("wget", "-q", "-O", filePath, fileURL); err != nil {
		return fmt.Errorf("downloading file %s: %w", fileURL, err)
	}
	if err := r.Run("wget", "-q", "-O", sigPath, signatureURL); err != nil {
		return fmt.Errorf("downloading signature %s: %w", signatureURL, err)
	}

	// Verify the signature.
	if err := r.Run("gpg", "--verify", sigPath, filePath); err != nil {
		return fmt.Errorf("GPG verification failed for %s: %w", fileURL, err)
	}

	return nil
}
