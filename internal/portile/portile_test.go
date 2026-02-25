package portile_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/portile"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFetcher records download calls without performing them.
type fakeFetcher struct {
	Downloads []downloadCall
}

type downloadCall struct {
	URL      string
	Dest     string
	Checksum source.Checksum
}

func (f *fakeFetcher) Download(_ context.Context, url, dest string, checksum source.Checksum) error {
	f.Downloads = append(f.Downloads, downloadCall{URL: url, Dest: dest, Checksum: checksum})
	return nil
}

func (f *fakeFetcher) ReadBody(_ context.Context, url string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestTmpPath(t *testing.T) {
	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
	}

	path := p.TmpPath()
	assert.Contains(t, path, "/tmp/")
	assert.Contains(t, path, "ports/ruby/3.3.6")
}

func TestPortPath(t *testing.T) {
	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
	}

	path := p.PortPath()
	assert.Contains(t, path, "ports/ruby/3.3.6/port")
}

func TestCookSequence(t *testing.T) {
	f := runner.NewFakeRunner()
	ff := &fakeFetcher{}

	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
		URL:     "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz",
		Checksum: source.Checksum{
			Algorithm: "sha256",
			Value:     "abc123",
		},
		Prefix:  "/usr/local",
		Options: []string{"--enable-shared", "--disable-install-doc"},
		Runner:  f,
		Fetcher: ff,
	}

	err := p.Cook(context.Background())
	require.NoError(t, err)

	// Verify download was called.
	require.Len(t, ff.Downloads, 1)
	assert.Equal(t, "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz", ff.Downloads[0].URL)
	assert.Equal(t, "sha256", ff.Downloads[0].Checksum.Algorithm)

	// Verify the sequence of runner calls:
	// 1. mkdir -p (create tmp dir)
	// 2. tar xf (extract)
	// 3. mv (rename to port)
	// 4. ./configure (in port dir)
	// 5. make -j4 (in port dir)
	// 6. make install (in port dir)
	require.Len(t, f.Calls, 6)

	assert.Equal(t, "mkdir", f.Calls[0].Name)
	assert.Equal(t, "-p", f.Calls[0].Args[0])

	assert.Equal(t, "tar", f.Calls[1].Name)
	assert.Equal(t, "xf", f.Calls[1].Args[0])

	assert.Equal(t, "mv", f.Calls[2].Name)

	assert.Equal(t, "./configure", f.Calls[3].Name)
	assert.Contains(t, f.Calls[3].Args, "--prefix=/usr/local")
	assert.Contains(t, f.Calls[3].Args, "--enable-shared")
	assert.Contains(t, f.Calls[3].Args, "--disable-install-doc")
	assert.NotEmpty(t, f.Calls[3].Dir, "configure should run in port dir")

	assert.Equal(t, "make", f.Calls[4].Name)
	assert.Equal(t, "-j4", f.Calls[4].Args[0])
	assert.NotEmpty(t, f.Calls[4].Dir)

	assert.Equal(t, "make", f.Calls[5].Name)
	assert.Equal(t, "install", f.Calls[5].Args[0])
	assert.NotEmpty(t, f.Calls[5].Dir)
}

func TestCookWithExtraOptions(t *testing.T) {
	f := runner.NewFakeRunner()
	ff := &fakeFetcher{}

	p := &portile.Portile{
		Name:    "nginx",
		Version: "1.25.3",
		URL:     "https://nginx.org/download/nginx-1.25.3.tar.gz",
		Options: []string{"--with-http_ssl_module", "--with-http_v2_module"},
		Runner:  f,
		Fetcher: ff,
	}

	err := p.Cook(context.Background())
	require.NoError(t, err)

	// Find the configure call.
	var configureCall runner.Call
	for _, c := range f.Calls {
		if c.Name == "./configure" {
			configureCall = c
			break
		}
	}

	assert.Contains(t, configureCall.Args, "--with-http_ssl_module")
	assert.Contains(t, configureCall.Args, "--with-http_v2_module")
}

func TestCookFailureOnMake(t *testing.T) {
	f := runner.NewFakeRunner()
	ff := &fakeFetcher{}

	// Make the "make" command fail.
	f.ErrorMap["make -j4"] = fmt.Errorf("compilation failed")

	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
		URL:     "https://example.com/ruby-3.3.6.tar.gz",
		Runner:  f,
		Fetcher: ff,
	}

	err := p.Cook(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile")

	// Verify install was NOT called after make failed.
	for _, c := range f.Calls {
		if c.Name == "make" && len(c.Args) > 0 && c.Args[0] == "install" {
			t.Fatal("make install should not be called after make fails")
		}
	}
}

func TestCookDefaultPrefix(t *testing.T) {
	f := runner.NewFakeRunner()
	ff := &fakeFetcher{}

	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
		URL:     "https://example.com/ruby-3.3.6.tar.gz",
		Runner:  f,
		Fetcher: ff,
	}

	err := p.Cook(context.Background())
	require.NoError(t, err)

	// When Prefix is empty, it defaults to PortPath().
	var configureCall runner.Call
	for _, c := range f.Calls {
		if c.Name == "./configure" {
			configureCall = c
			break
		}
	}

	assert.Contains(t, configureCall.Args[0], "--prefix=")
	assert.Contains(t, configureCall.Args[0], "ports/ruby/3.3.6/port")
}

func TestCookCustomJobs(t *testing.T) {
	f := runner.NewFakeRunner()
	ff := &fakeFetcher{}

	p := &portile.Portile{
		Name:    "ruby",
		Version: "3.3.6",
		URL:     "https://example.com/ruby-3.3.6.tar.gz",
		Jobs:    2,
		Runner:  f,
		Fetcher: ff,
	}

	err := p.Cook(context.Background())
	require.NoError(t, err)

	var makeCall runner.Call
	for _, c := range f.Calls {
		if c.Name == "make" && len(c.Args) > 0 && c.Args[0] != "install" {
			makeCall = c
			break
		}
	}

	assert.Equal(t, "-j2", makeCall.Args[0])
}
