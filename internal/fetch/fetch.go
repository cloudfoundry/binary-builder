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
	"net/http"
	"os"

	"github.com/cloudfoundry/binary-builder/internal/source"
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
	Client *http.Client
}

// NewHTTPFetcher creates a Fetcher with a default HTTP client.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{Client: http.DefaultClient}
}

// Download fetches a URL to dest, following redirects, and verifies the checksum.
func (f *HTTPFetcher) Download(ctx context.Context, url, dest string, checksum source.Checksum) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}

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
func (f *HTTPFetcher) ReadBody(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

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
