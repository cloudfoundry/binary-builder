package gpg_test

import (
	"context"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/gpg"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	f := runner.NewFakeRunner()

	keyURLs := []string{
		"http://nginx.org/keys/maxim.key",
		"http://nginx.org/keys/arut.key",
	}

	err := gpg.VerifySignature(
		context.Background(),
		"http://nginx.org/download/nginx-1.25.3.tar.gz",
		"http://nginx.org/download/nginx-1.25.3.tar.gz.asc",
		keyURLs,
		f,
	)
	require.NoError(t, err)

	// Expect: wget key0, gpg import key0, wget key1, gpg import key1,
	//         wget file, wget sig, gpg verify
	require.Len(t, f.Calls, 7)

	// Key 0: download + import
	assert.Equal(t, "wget", f.Calls[0].Name)
	assert.Contains(t, f.Calls[0].Args[len(f.Calls[0].Args)-1], "maxim.key")
	assert.Equal(t, "gpg", f.Calls[1].Name)
	assert.Equal(t, "--import", f.Calls[1].Args[0])

	// Key 1: download + import
	assert.Equal(t, "wget", f.Calls[2].Name)
	assert.Contains(t, f.Calls[2].Args[len(f.Calls[2].Args)-1], "arut.key")
	assert.Equal(t, "gpg", f.Calls[3].Name)
	assert.Equal(t, "--import", f.Calls[3].Args[0])

	// File download
	assert.Equal(t, "wget", f.Calls[4].Name)
	assert.Contains(t, f.Calls[4].Args[len(f.Calls[4].Args)-1], "nginx-1.25.3.tar.gz")

	// Signature download
	assert.Equal(t, "wget", f.Calls[5].Name)
	assert.Contains(t, f.Calls[5].Args[len(f.Calls[5].Args)-1], "nginx-1.25.3.tar.gz.asc")

	// GPG verify
	assert.Equal(t, "gpg", f.Calls[6].Name)
	assert.Equal(t, "--verify", f.Calls[6].Args[0])
}

func TestVerifySignatureMultipleKeys(t *testing.T) {
	f := runner.NewFakeRunner()

	keyURLs := []string{
		"http://example.com/key1.asc",
		"http://example.com/key2.asc",
		"http://example.com/key3.asc",
	}

	err := gpg.VerifySignature(
		context.Background(),
		"http://example.com/file.tar.gz",
		"http://example.com/file.tar.gz.asc",
		keyURLs,
		f,
	)
	require.NoError(t, err)

	// 3 keys × 2 calls (wget + import) + 2 (file wget + sig wget) + 1 (verify) = 9
	assert.Len(t, f.Calls, 9)

	// All keys imported before verify
	importCount := 0
	for _, call := range f.Calls {
		if call.Name == "gpg" && len(call.Args) > 0 && call.Args[0] == "--import" {
			importCount++
		}
	}
	assert.Equal(t, 3, importCount)
}
