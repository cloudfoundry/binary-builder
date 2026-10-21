// Package fetch provides HTTP download with checksum verification.
package fetch

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/cloudfoundry/binary-builder/internal/source"
)

const (
	// MaxRetries is the number of additional attempts after the first failure.
	// Exported so tests can reference it without duplicating the value.
	MaxRetries = 5
	// BaseRetryWait is the base wait time before the first retry.
	// Exported so tests can override or reference it.
	BaseRetryWait = 5 * time.Second
)

// Fetcher is the interface for downloading files and reading URL bodies.
type Fetcher interface {
	// Download fetches a URL to a local file, verifying the checksum.
	// If checksum.Value is empty, no verification is performed.
	Download(ctx context.Context, url, dest string, checksum source.Checksum) error

	// ReadBody fetches a URL and returns the response body as bytes.
	ReadBody(ctx context.Context, url string) ([]byte, error)
}

// HTTPFetcher implements Fetcher using net/http.
type HTTPFetcher struct {
	Client    *http.Client
	RetryWait time.Duration // base wait between retries; 0 means use BaseRetryWait
}

// NewHTTPFetcher creates a Fetcher with a 10-minute HTTP client timeout.
// A timeout prevents builds hanging indefinitely on stalled downloads.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{Client: &http.Client{Timeout: 10 * time.Minute}}
}

// retryWait returns the configured base retry wait, defaulting to BaseRetryWait.
func (f *HTTPFetcher) retryWait() time.Duration {
	if f.RetryWait > 0 {
		return f.RetryWait
	}
	return BaseRetryWait
}

// doWithRetry executes an HTTP GET request with retry on transient errors.
// It retries up to MaxRetries times on server errors (5xx) and network
// failures. Client errors (4xx) fail immediately. Returns a successful
// response (HTTP 200) with an open body, or an error.
func (f *HTTPFetcher) doWithRetry(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			wait := f.retryWait() * time.Duration(math.Pow(2, float64(attempt-1)))
			fmt.Fprintf(os.Stderr, "fetch: retrying %s in %v (attempt %d/%d)\n", url, wait, attempt+1, MaxRetries+1)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, fmt.Errorf("fetching %s: %w", url, ctx.Err())
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request for %s: %w", url, err)
		}

		resp, err := f.Client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetching %s: %w", url, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()

		// Only retry on server errors (5xx); client errors (4xx) are permanent.
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
			continue
		}

		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	return nil, lastErr
}

// Download fetches a URL to dest, following redirects, and verifies the checksum.
// Retries up to MaxRetries times on transient HTTP errors (5xx, timeouts).
func (f *HTTPFetcher) Download(ctx context.Context, url, dest string, checksum source.Checksum) error {
	resp, err := f.doWithRetry(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer out.Close()

	var h hash.Hash
	if checksum.Value != "" {
		h, err = newHash(checksum.Algorithm)
		if err != nil {
			return err
		}
	}

	var w io.Writer = out
	if h != nil {
		w = io.MultiWriter(out, h)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("writing %s: %w", dest, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", dest, err)
	}

	if h != nil {
		actual := fmt.Sprintf("%x", h.Sum(nil))
		if actual != checksum.Value {
			os.Remove(dest)
			return fmt.Errorf("%s digest does not match: expected %s, got %s", checksum.Algorithm, checksum.Value, actual)
		}
	}

	return nil
}

// ReadBody fetches a URL and returns the response body.
// Retries up to MaxRetries times on transient HTTP errors (5xx, timeouts).
func (f *HTTPFetcher) ReadBody(ctx context.Context, url string) ([]byte, error) {
	resp, err := f.doWithRetry(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body from %s: %w", url, err)
	}

	return body, nil
}

func newHash(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}
}
