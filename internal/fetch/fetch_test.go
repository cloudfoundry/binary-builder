package fetch_test

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadCorrectSHA256(t *testing.T) {
	body := []byte("hello world")
	sha := fmt.Sprintf("%x", sha256.Sum256(body))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{
		Algorithm: "sha256",
		Value:     sha,
	})
	require.NoError(t, err)

	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, body, content)
}

func TestDownloadWrongSHA256(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{
		Algorithm: "sha256",
		Value:     "0000000000000000000000000000000000000000000000000000000000000000",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sha256 digest does not match")

	// File should be cleaned up on checksum failure.
	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr))
}

func TestDownloadWrongSHA512(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{
		Algorithm: "sha512",
		Value:     "0000",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sha512 digest does not match")
}

func TestDownloadCorrectSHA512(t *testing.T) {
	body := []byte("hello world")
	h := sha512.Sum512(body)
	sha := fmt.Sprintf("%x", h)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{
		Algorithm: "sha512",
		Value:     sha,
	})
	require.NoError(t, err)
}

func TestDownloadNoChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{})
	require.NoError(t, err)
}

func TestDownload404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestReadBodySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response body"))
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	body, err := f.ReadBody(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, []byte("response body"), body)
}

func TestReadBody404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	_, err := f.ReadBody(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownloadFollowsRedirect(t *testing.T) {
	body := []byte("final content")
	sha := fmt.Sprintf("%x", sha256.Sum256(body))

	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL+"/redirect", dest, source.Checksum{
		Algorithm: "sha256",
		Value:     sha,
	})
	require.NoError(t, err)

	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, body, content)
}

func TestDownloadUnsupportedAlgorithm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	f := fetch.NewHTTPFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{
		Algorithm: "crc32",
		Value:     "abc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported checksum algorithm")
}
