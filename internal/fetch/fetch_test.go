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
	"sync/atomic"
	"testing"
	"time"

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

// newFastRetryFetcher creates an HTTPFetcher with a minimal retry wait so
// tests that exercise the retry loop complete quickly.
func newFastRetryFetcher() *fetch.HTTPFetcher {
	return &fetch.HTTPFetcher{
		Client:    &http.Client{Timeout: 5 * time.Second},
		RetryWait: 1 * time.Millisecond,
	}
}

func TestDownloadRetryOn5xx(t *testing.T) {
	var calls atomic.Int32
	body := []byte("success after retry")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusBadGateway) // 502
			return
		}
		w.Write(body)
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{})
	require.NoError(t, err)

	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, body, content)
	assert.Equal(t, int32(3), calls.Load(), "expected 2 failures + 1 success = 3 total requests")
}

func TestReadBodyRetryOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError) // 500
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	body, err := f.ReadBody(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), body)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDownloadNoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden) // 403
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Equal(t, int32(1), calls.Load(), "4xx errors must not be retried")
}

func TestReadBodyNoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound) // 404
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	_, err := f.ReadBody(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Equal(t, int32(1), calls.Load(), "4xx errors must not be retried")
}

func TestDownloadRetryExhaustion(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable) // 503 every time
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	dest := filepath.Join(t.TempDir(), "file.tgz")

	err := f.Download(context.Background(), srv.URL, dest, source.Checksum{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
	assert.Equal(t, int32(fetch.MaxRetries+1), calls.Load(),
		"expected 1 initial + %d retries = %d total requests", fetch.MaxRetries, fetch.MaxRetries+1)
}

func TestReadBodyRetryExhaustion(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusGatewayTimeout) // 504 every time
	}))
	defer srv.Close()

	f := newFastRetryFetcher()
	_, err := f.ReadBody(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "504")
	assert.Equal(t, int32(fetch.MaxRetries+1), calls.Load())
}

func TestDownloadRespectsContextCancellation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	f := &fetch.HTTPFetcher{
		Client:    &http.Client{Timeout: 5 * time.Second},
		RetryWait: 500 * time.Millisecond, // long enough to cancel during wait
	}
	dest := filepath.Join(t.TempDir(), "file.tgz")

	// Cancel the context after the first failure triggers a retry wait.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := f.Download(ctx, srv.URL, dest, source.Checksum{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	// Should have made only 1 request before being cancelled during the retry wait.
	assert.Equal(t, int32(1), calls.Load(), "should stop retrying after context cancellation")
}

func TestReadBodyRespectsContextCancellation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	f := &fetch.HTTPFetcher{
		Client:    &http.Client{Timeout: 5 * time.Second},
		RetryWait: 500 * time.Millisecond,
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := f.ReadBody(ctx, srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Equal(t, int32(1), calls.Load())
}
