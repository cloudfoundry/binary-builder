package recipe_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/recipe"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newCompiledStack(t *testing.T) *stack.Stack {
	t.Helper()
	return &stack.Stack{
		Name: "cflinuxfs4",
		RubyBootstrap: stack.RubyBootstrap{
			URL:        "https://example.com/ruby-bootstrap.tgz",
			SHA256:     "deadbeef",
			InstallDir: "/opt/ruby",
		},
		Compilers: stack.CompilerConfig{
			GCC: stack.GCCConfig{
				Version:  12,
				Packages: []string{"gcc-12", "g++-12"},
				PPA:      "ppa:ubuntu-toolchain-r/test",
			},
			Gfortran: stack.GfortranConfig{
				Version:  11,
				Bin:      "/usr/bin/gfortran-11",
				LibPath:  "/usr/lib/gcc/x86_64-linux-gnu/11",
				Packages: []string{"gfortran"},
			},
		},
		AptPackages: map[string][]string{
			"ruby_build":       {"libffi-dev"},
			"python_build":     {"libdb-dev", "libgdbm-dev", "tk8.6-dev"},
			"node_build":       {},
			"libgdiplus_build": {"automake", "libtool", "libglib2.0-dev", "libcairo2-dev"},
		},
		Python: stack.PythonConfig{
			TCLVersion:  "8.6",
			UseForceYes: true,
		},
		JRuby: stack.JRubyConfig{
			JDKURL:        "https://example.com/openjdk.tar.gz",
			JDKSHA256:     "cafebabe",
			JDKInstallDir: t.TempDir() + "/java",
		},
		Go: stack.GoConfig{
			BootstrapURL:    "https://example.com/go-bootstrap.tar.gz",
			BootstrapSHA256: "deadbeef",
		},
		HTTPDSubDeps: stack.HTTPDSubDepsConfig{
			APR: stack.HTTPDSubDep{
				Version: "1.7.4",
				URL:     "https://example.com/apr-1.7.4.tar.gz",
				SHA256:  "aprsha256",
			},
			APRIconv: stack.HTTPDSubDep{
				Version: "1.2.2",
				URL:     "https://example.com/apr-iconv-1.2.2.tar.gz",
				SHA256:  "apriconvsha256",
			},
			APRUtil: stack.HTTPDSubDep{
				Version: "1.6.3",
				URL:     "https://example.com/apr-util-1.6.3.tar.gz",
				SHA256:  "aprutilsha256",
			},
			ModAuthOpenidc: stack.HTTPDSubDep{
				Version: "2.3.8",
				URL:     "https://example.com/mod_auth_openidc-2.3.8.tar.gz",
				SHA256:  "modauthsha256",
			},
		},
	}
}

// helpers are in recipe_helpers_test.go

// ── RubyRecipe ───────────────────────────────────────────────────────────────

func TestRubyRecipeName(t *testing.T) {
	r := &recipe.RubyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "ruby", r.Name())
}

func TestRubyRecipeArtifact(t *testing.T) {
	r := &recipe.RubyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "linux", r.Artifact().OS)
	assert.Equal(t, "x64", r.Artifact().Arch)
	assert.Equal(t, "", r.Artifact().Stack) // stack-specific → set at build time
}

func TestRubyRecipeBuild(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "ruby-3.3.1-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.RubyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("ruby", "3.3.1", "https://cache.ruby-lang.org/pub/ruby/ruby-3.3.1.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should install apt packages.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "libffi-dev"), "should apt-install ruby_build packages")

	// Should have downloaded via portile fetcher.
	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, src.URL, f.DownloadedURLs[0].URL)

	// Should invoke tar + configure chain (mkdir, tar xf, mv, ./configure, make, make install, tar czf).
	names := callNames(run.Calls)
	assert.Contains(t, names, "mkdir")
	assert.Contains(t, names, "tar")

	// Should configure with the correct portile flags.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--enable-load-relative"), "missing --enable-load-relative")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--disable-install-doc"), "missing --disable-install-doc")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--without-gmp"), "missing --without-gmp")

	// Prefix should include the version.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "ruby-3.3.1"), "prefix should reference version")
}

func TestRubyRecipeFetchError(t *testing.T) {
	f := newFakeFetcher()
	f.ErrMap[newInput("ruby", "3.3.1", "https://example.com/ruby.tgz").URL] = assert.AnError
	r := &recipe.RubyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()

	src := newInput("ruby", "3.3.1", "https://example.com/ruby.tgz")
	err := r.Build(context.Background(), newCompiledStack(t), src, run, &output.OutData{})
	require.Error(t, err)
}

// ── BundlerRecipe ─────────────────────────────────────────────────────────────

func TestBundlerRecipeName(t *testing.T) {
	r := &recipe.BundlerRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "bundler", r.Name())
}

func TestBundlerRecipeArtifact(t *testing.T) {
	r := &recipe.BundlerRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

func TestBundlerRecipeBuild(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.BundlerRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("bundler", "2.5.6", "https://rubygems.org/gems/bundler-2.5.6.gem")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should have downloaded the Ruby bootstrap binary.
	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, s.RubyBootstrap.URL, f.DownloadedURLs[0].URL)

	// Should create the install dir and extract the bootstrap.
	assert.True(t, hasCallMatching(run.Calls, "mkdir", "/opt/ruby"), "should mkdir install dir")
	assert.True(t, hasCallMatching(run.Calls, "tar", "/opt/ruby"), "should extract bootstrap to install dir")

	// Should call gem install with version.
	// The gem binary is invoked by full path (e.g. /opt/ruby/bin/gem) to avoid
	// PATH resolution issues, so we match on the bootstrap install dir.
	gemBin := filepath.Join(s.RubyBootstrap.InstallDir, "bin", "gem")
	assert.True(t, hasCallMatching(run.Calls, gemBin, "bundler"), "should call gem install bundler")
	assert.True(t, hasCallMatching(run.Calls, gemBin, "2.5.6"), "should install specific version")
	assert.True(t, hasCallMatching(run.Calls, gemBin, "--no-document"), "should skip documentation")

	// gem install should run with GEM_HOME set so gems land in an isolated tmpdir.
	assert.True(t, hasCallWithEnv(run.Calls, gemBin, "GEM_HOME"), "gem install should have GEM_HOME env set")
}

// ── PythonRecipe ──────────────────────────────────────────────────────────────

func TestPythonRecipeName(t *testing.T) {
	r := &recipe.PythonRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "python", r.Name())
}

func TestPythonRecipeArtifact(t *testing.T) {
	r := &recipe.PythonRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestPythonRecipeBuildForceYes(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.PythonRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.Python.UseForceYes = true
	src := newInput("python", "3.12.0", "https://python.org/Python-3.12.0.tgz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// cflinuxfs4: should use --force-yes when downloading deb packages.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "--force-yes"), "cflinuxfs4 should use --force-yes")

	// Should invoke portile configure with tcl/tk flags.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--enable-shared"), "missing --enable-shared")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "tcl8.6"), "configure should reference tcl version")

	// Should extract each deb via dpkg -x.
	assert.True(t, hasCallMatching(run.Calls, "sh", "dpkg -x"), "should run dpkg -x for each deb")

	// Should create bin/python symlink.
	assert.True(t, hasCallMatching(run.Calls, "ln", "python"), "should create bin/python symlink")
}

func TestPythonRecipeBuildNoForceYes(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.PythonRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.Python.UseForceYes = false // cflinuxfs5
	src := newInput("python", "3.12.0", "https://python.org/Python-3.12.0.tgz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// cflinuxfs5: should NOT use --force-yes.
	assert.False(t, hasCallMatching(run.Calls, "apt-get", "--force-yes"), "cflinuxfs5 must not use --force-yes")
}

// ── NodeRecipe ────────────────────────────────────────────────────────────────

func TestNodeRecipeName(t *testing.T) {
	r := &recipe.NodeRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "node", r.Name())
}

func TestNodeRecipeArtifact(t *testing.T) {
	r := &recipe.NodeRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestNodeRecipeStripsVPrefix(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "node-22.14.0-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NodeRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("node", "v22.14.0", "https://nodejs.org/v22.14.0.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// configure uses --prefix=/ (DESTDIR-based install) + --openssl-use-def-ca-store.
	// The version (without `v`) appears in the DESTDIR passed to make install, not in configure.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--prefix=/"), "configure must use --prefix=/")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--openssl-use-def-ca-store"), "configure must pass --openssl-use-def-ca-store")
	assert.False(t, hasCallMatching(run.Calls, "./configure", "v22.14.0"), "configure must not reference v-prefixed version")
	// DESTDIR uses stripped version; "node-v22.14.0" is acceptable since DESTDIR path includes the `v`.
	assert.True(t, hasCallMatching(run.Calls, "make", "DESTDIR="), "make install must pass DESTDIR")
	assert.True(t, hasCallMatching(run.Calls, "make", "22.14.0"), "make install DESTDIR must contain the version")
}

func TestNodeRecipeSetsUpGCC(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "node-22.14.0-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NodeRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("node", "v22.14.0", "https://nodejs.org/v22.14.0.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should install software-properties-common.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "software-properties-common"), "should install software-properties-common")

	// cflinuxfs4 has a PPA — should add it.
	assert.True(t, hasCallMatching(run.Calls, "add-apt-repository", "ppa:ubuntu-toolchain-r/test"), "should add GCC PPA on cflinuxfs4")

	// Should set up update-alternatives.
	assert.True(t, hasCallMatching(run.Calls, "update-alternatives", "gcc"), "should set up update-alternatives for gcc")
}

func TestNodeRecipeSkipsPPAWhenEmpty(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "node-22.14.0-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NodeRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.Compilers.GCC.PPA = "" // cflinuxfs5 — no PPA
	src := newInput("node", "v22.14.0", "https://nodejs.org/v22.14.0.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should NOT call add-apt-repository when PPA is empty.
	assert.False(t, hasCallMatching(run.Calls, "add-apt-repository", ""), "must not add PPA when PPA is empty")
}

// ── GoRecipe ──────────────────────────────────────────────────────────────────

func TestGoRecipeName(t *testing.T) {
	r := &recipe.GoRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "go", r.Name())
}

func TestGoRecipeArtifact(t *testing.T) {
	r := &recipe.GoRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestGoRecipeStripsGoPrefix(t *testing.T) {
	useTempWorkDir(t)
	// Artifact filename uses a dash between name and version so findIntermediateArtifact
	// can locate it via the "go-1.24.2*.tar.gz" glob pattern.
	writeFakeArtifact(t, "go-1.24.2.linux-amd64.tar.gz")

	f := newFakeFetcher()
	r := &recipe.GoRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("go", "go1.24.2", "https://go.dev/dl/go1.24.2.src.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should download bootstrap AND source — 2 downloads total.
	require.Len(t, f.DownloadedURLs, 2)
	assert.True(t, hasDownload(f, s.Go.BootstrapURL), "should download bootstrap go binary")
	assert.True(t, hasDownload(f, src.URL), "should download go source")

	// Artifact path should use stripped version.
	assert.True(t, hasCallMatching(run.Calls, "tar", "1.24.2"), "artifact should use version without go prefix")

	// Should run make.bash.
	assert.True(t, hasCallMatching(run.Calls, "bash", "make.bash"), "should run make.bash")

	// make.bash should have GOROOT_BOOTSTRAP env.
	assert.True(t, hasCallWithEnv(run.Calls, "bash", "GOROOT_BOOTSTRAP"), "make.bash should set GOROOT_BOOTSTRAP")
}

// ── NginxRecipe ───────────────────────────────────────────────────────────────

func TestNginxRecipeName(t *testing.T) {
	r := &recipe.NginxRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "nginx", r.Name())
}

func TestNginxRecipeArtifact(t *testing.T) {
	r := &recipe.NginxRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestNginxRecipeBuildRunsGPGVerify(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "nginx-1.25.3-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NginxRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("nginx", "1.25.3", "https://nginx.org/download/nginx-1.25.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should download GPG keys and verify signature.
	assert.True(t, hasCallMatching(run.Calls, "wget", "nginx.org/keys"), "should download nginx GPG keys")
	assert.True(t, hasCallMatching(run.Calls, "gpg", "--import"), "should import GPG keys")
	assert.True(t, hasCallMatching(run.Calls, "gpg", "--verify"), "should verify GPG signature")
}

func TestNginxRecipeUsesPICFlags(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "nginx-1.25.3-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NginxRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("nginx", "1.25.3", "https://nginx.org/download/nginx-1.25.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// nginx uses PIC flags.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "-fPIC"), "nginx configure should use -fPIC")

	// nginx includes dynamic modules.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--with-compat"), "nginx should have --with-compat")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--with-mail=dynamic"), "nginx should have --with-mail=dynamic")
}

// ── NginxStaticRecipe ─────────────────────────────────────────────────────────

func TestNginxStaticRecipeName(t *testing.T) {
	r := &recipe.NginxStaticRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "nginx-static", r.Name())
}

func TestNginxStaticRecipeUsesPIEFlags(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "nginx-static-1.25.3-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NginxStaticRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("nginx-static", "1.25.3", "https://nginx.org/download/nginx-1.25.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// nginx-static uses PIE flags.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "-fPIE"), "nginx-static configure should use -fPIE")

	// nginx-static does NOT include the extra dynamic modules.
	assert.False(t, hasCallMatching(run.Calls, "./configure", "--with-compat"), "nginx-static must not have --with-compat")
	assert.False(t, hasCallMatching(run.Calls, "./configure", "--with-mail=dynamic"), "nginx-static must not have --with-mail=dynamic")
}

func TestNginxStaticRecipeAlsoRunsGPGVerify(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "nginx-static-1.25.3-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.NginxStaticRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("nginx-static", "1.25.3", "https://nginx.org/download/nginx-1.25.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "gpg", "--verify"), "nginx-static should also verify GPG signature")
}

// ── OpenrestyRecipe ───────────────────────────────────────────────────────────

func TestOpenrestyRecipeName(t *testing.T) {
	r := &recipe.OpenrestyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "openresty", r.Name())
}

func TestOpenrestyRecipeArtifact(t *testing.T) {
	r := &recipe.OpenrestyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestOpenrestyRecipeBuild(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.OpenrestyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("openresty", "1.21.4.3", "https://openresty.org/download/openresty-1.21.4.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should download source via Fetcher (not wget).
	assert.True(t, hasDownload(f, src.URL), "should download source via Fetcher")
	assert.False(t, hasCallMatching(run.Calls, "wget", src.URL), "must not use wget")
	assert.False(t, hasCallMatching(run.Calls, "gpg", ""), "openresty must not run gpg verify")

	// Should configure with PIC flags.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "-fPIC"), "openresty configure should use -fPIC")

	// Should use -j2 for make.
	assert.True(t, hasCallMatching(run.Calls, "make", "-j2"), "openresty should make -j2")
}

// ── LibunwindRecipe ───────────────────────────────────────────────────────────

func TestLibunwindRecipeName(t *testing.T) {
	r := &recipe.LibunwindRecipe{}
	assert.Equal(t, "libunwind", r.Name())
}

func TestLibunwindRecipeArtifact(t *testing.T) {
	r := &recipe.LibunwindRecipe{}
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

func TestLibunwindRecipeBuild(t *testing.T) {
	r := &recipe.LibunwindRecipe{}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := &source.Input{
		Name:    "libunwind",
		Version: "1.6.2",
		URL:     "https://github.com/libunwind/libunwind/releases/download/v1.6.2/libunwind-1.6.2.tar.gz",
		SHA256:  "abc",
	}

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should extract from source/ (pre-downloaded by Concourse).
	assert.True(t, hasCallMatching(run.Calls, "tar", "source/libunwind-1.6.2.tar.gz"), "should extract pre-downloaded source tarball")

	// Should run configure, make, make install.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--prefix="), "should configure with prefix")
	assert.True(t, hasCallMatching(run.Calls, "make", ""), "should run make")

	// Should pack only include/ and lib/.
	assert.True(t, hasCallMatching(run.Calls, "tar", "include"), "artifact should contain include/")
	assert.True(t, hasCallMatching(run.Calls, "tar", "lib"), "artifact should contain lib/")
}

// ── LibgdiplusRecipe ──────────────────────────────────────────────────────────

func TestLibgdiplusRecipeName(t *testing.T) {
	r := &recipe.LibgdiplusRecipe{}
	assert.Equal(t, "libgdiplus", r.Name())
}

func TestLibgdiplusRecipeArtifact(t *testing.T) {
	r := &recipe.LibgdiplusRecipe{}
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

func TestLibgdiplusRecipeBuild(t *testing.T) {
	r := &recipe.LibgdiplusRecipe{}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := &source.Input{
		Name:    "libgdiplus",
		Version: "6.1",
		URL:     "https://github.com/mono/libgdiplus/releases/tag/6.1",
		SHA256:  "abc",
		Repo:    "mono/libgdiplus",
	}

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should apt install libgdiplus_build packages.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "automake"), "should install libgdiplus_build packages")

	// Should git clone the repo at the version tag.
	assert.True(t, hasCallMatching(run.Calls, "git", "mono/libgdiplus"), "should clone mono/libgdiplus")
	assert.True(t, hasCallMatching(run.Calls, "git", "6.1"), "should clone at version tag")

	// Should run autogen with warning suppression flags.
	assert.True(t, hasCallWithEnv(run.Calls, "sh", "CFLAGS"), "autogen.sh should have CFLAGS env")

	// Should pack only lib/.
	assert.True(t, hasCallMatching(run.Calls, "tar", "lib"), "artifact should contain only lib/")
}

// ── DepRecipe ─────────────────────────────────────────────────────────────────

func TestDepRecipeName(t *testing.T) {
	r := &recipe.DepRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "dep", r.Name())
}

func TestDepRecipeArtifact(t *testing.T) {
	r := &recipe.DepRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestDepRecipeBuild(t *testing.T) {
	useTempWorkDir(t)

	f := newFakeFetcher()
	r := &recipe.DepRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("dep", "0.5.4", "https://github.com/golang/dep/archive/v0.5.4.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	require.Len(t, f.DownloadedURLs, 1)
	assert.Equal(t, src.URL, f.DownloadedURLs[0].URL)

	// Ruby runs `go get -asmflags -trimpath ./...` via `sh -c "cd {srcDir} && GOPATH=... go get ..."`.
	assert.True(t, hasCallMatching(run.Calls, "sh", "go get"), "should run go get via sh -c")
	assert.True(t, hasCallMatching(run.Calls, "sh", "-asmflags"), "go get should use -asmflags flag")
	assert.True(t, hasCallMatching(run.Calls, "sh", "GOPATH="), "should set GOPATH")

	// Should pack bin/dep + bin/LICENSE.
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/dep"), "artifact should contain bin/dep")
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/LICENSE"), "artifact should contain bin/LICENSE")
}

// ── GlideRecipe ───────────────────────────────────────────────────────────────

func TestGlideRecipeName(t *testing.T) {
	r := &recipe.GlideRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "glide", r.Name())
}

func TestGlideRecipeBuild(t *testing.T) {
	useTempWorkDir(t)

	f := newFakeFetcher()
	r := &recipe.GlideRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("glide", "0.13.3", "https://github.com/Masterminds/glide/archive/v0.13.3.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Ruby runs `go build` via `sh -c "cd {srcDir} && GOPATH=/tmp go build"`.
	assert.True(t, hasCallMatching(run.Calls, "sh", "go build"), "should run go build via sh -c")
	assert.True(t, hasCallMatching(run.Calls, "sh", "GOPATH="), "should set GOPATH")

	// Should pack bin/glide + bin/LICENSE.
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/glide"), "artifact should contain bin/glide")
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/LICENSE"), "artifact should contain bin/LICENSE")
}

// ── GodepRecipe ───────────────────────────────────────────────────────────────

func TestGodepRecipeName(t *testing.T) {
	r := &recipe.GodepRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "godep", r.Name())
}

func TestGodepRecipeBuild(t *testing.T) {
	useTempWorkDir(t)

	f := newFakeFetcher()
	r := &recipe.GodepRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("godep", "80", "https://github.com/tools/godep/archive/v80.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Ruby runs `go get ./...` via `sh -c "cd {srcDir} && GOPATH=... go get ./..."`.
	assert.True(t, hasCallMatching(run.Calls, "sh", "go get"), "should run go get via sh -c")
	assert.True(t, hasCallMatching(run.Calls, "sh", "GOPATH="), "should set GOPATH")

	// Should pack bin/godep + bin/License (capital L, no E — matches Ruby).
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/godep"), "artifact should contain bin/godep")
	assert.True(t, hasCallMatching(run.Calls, "tar", "bin/License"), "artifact should contain bin/License (capital L)")
}

// ── HWCRecipe ─────────────────────────────────────────────────────────────────

func TestHWCRecipeName(t *testing.T) {
	r := &recipe.HWCRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "hwc", r.Name())
}

func TestHWCRecipeArtifact(t *testing.T) {
	r := &recipe.HWCRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "windows", r.Artifact().OS)
	assert.Equal(t, "x86-64", r.Artifact().Arch)
	assert.Equal(t, "any-stack", r.Artifact().Stack)
}

func TestHWCRecipeBuild(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.HWCRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("hwc", "2.0.10", "https://github.com/cloudfoundry/hwc/archive/v2.0.10.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should install mingw-w64.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "mingw-w64"), "should install mingw-w64")

	// Should download source.
	require.Len(t, f.DownloadedURLs, 1)

	// Should cross-compile with GOOS=windows.
	assert.True(t, hasCallWithEnv(run.Calls, "go", "GOOS"), "go build should have GOOS env set")
	assert.True(t, hasCallWithEnv(run.Calls, "go", "GOARCH"), "go build should have GOARCH env set")

	// Verify GOOS=windows value.
	for _, c := range run.Calls {
		if c.Name == "go" && c.Env != nil {
			if goos, ok := c.Env["GOOS"]; ok {
				assert.Equal(t, "windows", goos)
			}
		}
	}

	// Should produce a .zip (not .tgz) containing BOTH hwc.exe and hwc_x86.exe.
	assert.True(t, hasCallMatching(run.Calls, "zip", "hwc.exe"), "hwc artifact should be a zip containing hwc.exe")
	assert.True(t, hasCallMatching(run.Calls, "zip", "hwc_x86.exe"), "hwc artifact should be a zip containing hwc_x86.exe (386)")

	// Should build BOTH amd64 and 386.
	amd64Found := false
	x86Found := false
	for _, c := range run.Calls {
		if c.Name == "go" && c.Env != nil {
			if arch, ok := c.Env["GOARCH"]; ok {
				if arch == "amd64" {
					amd64Found = true
				}
				if arch == "386" {
					x86Found = true
				}
			}
		}
	}
	assert.True(t, amd64Found, "should build amd64 binary")
	assert.True(t, x86Found, "should build 386 binary")
}

// ── RRecipe ───────────────────────────────────────────────────────────────────

// writeRSubDepFiles creates the stub Concourse source data.json files that
// r.go reads via source.FromFile for each sub-dependency.
// Must be called after useTempWorkDir(t) so the files land in the temp dir.
func writeRSubDepFiles(t *testing.T) {
	t.Helper()
	subDeps := []struct {
		dir     string
		version string
	}{
		{"source-forecast-latest", "8.21.0"},
		{"source-plumber-latest", "1.2.1"},
		{"source-rserve-latest", "1.8.14"}, // will be formatted as 1.8-14
		{"source-shiny-latest", "1.8.0"},
	}
	for _, sd := range subDeps {
		if err := os.MkdirAll(sd.dir, 0755); err != nil {
			t.Fatalf("writeRSubDepFiles: mkdir %s: %v", sd.dir, err)
		}
		// Use the real depwatcher modern format:
		// { "source": {"name": "...", "type": "..."}, "version": {"url": "...", "ref": "...", "sha256": "..."} }
		jsonContent := fmt.Sprintf(
			`{"source":{"name":%q,"type":"cran"},"version":{"url":"https://cran.r-project.org/fake/%s.tar.gz","ref":%q,"sha256":"abc"}}`,
			sd.dir, sd.dir, sd.version,
		)
		if err := os.WriteFile(sd.dir+"/data.json", []byte(jsonContent), 0644); err != nil {
			t.Fatalf("writeRSubDepFiles: write %s/data.json: %v", sd.dir, err)
		}
	}
}

func TestRRecipeName(t *testing.T) {
	r := &recipe.RRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "r", r.Name())
}

func TestRRecipeArtifact(t *testing.T) {
	r := &recipe.RRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "linux", r.Artifact().OS)
	assert.Equal(t, "noarch", r.Artifact().Arch)
}

func TestRRecipeInstallsDevtoolsBeforePackages(t *testing.T) {
	useTempWorkDir(t)
	writeRSubDepFiles(t)

	f := newFakeFetcher()
	r := &recipe.RRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["r_build"] = []string{"r-base"}
	src := newInput("r", "4.3.1", "https://cran.r-project.org/src/base/R-4.3.1.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Find indices of devtools install and first install_version call.
	devtoolsIdx := -1
	installVersionIdx := -1
	for i, c := range run.Calls {
		if c.Name == "sh" {
			joined := strings.Join(c.Args, " ")
			if strings.Contains(joined, `install.packages("devtools"`) && devtoolsIdx < 0 {
				devtoolsIdx = i
			}
			if strings.Contains(joined, "install_version") && installVersionIdx < 0 {
				installVersionIdx = i
			}
		}
	}

	require.True(t, devtoolsIdx >= 0, "devtools install call not found")
	require.True(t, installVersionIdx >= 0, "install_version call not found")
	assert.Less(t, devtoolsIdx, installVersionIdx,
		"devtools must be installed BEFORE any install_version call (devtools at %d, install_version at %d)",
		devtoolsIdx, installVersionIdx)
}

func TestRRecipeRemovesDevtools(t *testing.T) {
	useTempWorkDir(t)
	writeRSubDepFiles(t)

	f := newFakeFetcher()
	r := &recipe.RRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["r_build"] = []string{"r-base"}
	src := newInput("r", "4.3.1", "https://cran.r-project.org/src/base/R-4.3.1.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "sh", `remove.packages("devtools")`),
		"should call remove.packages(\"devtools\") after package installs")
}

func TestRserveVersionFormatting(t *testing.T) {
	useTempWorkDir(t)
	writeRSubDepFiles(t)

	f := newFakeFetcher()
	r := &recipe.RRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["r_build"] = []string{"r-base"}
	src := newInput("r", "4.3.1", "https://cran.r-project.org/src/base/R-4.3.1.tar.gz")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Rserve version "1.8.14" must be formatted as "1.8-14" in the install_version call.
	// The R command uses single quotes (matching Ruby builder style).
	assert.True(t, hasCallMatching(run.Calls, "sh", "install_version('Rserve'"),
		"should call install_version for Rserve")
	assert.True(t, hasCallMatching(run.Calls, "sh", "1.8-14"),
		"Rserve version should be formatted as '1.8-14'")
	assert.False(t, hasCallMatching(run.Calls, "sh", "1.8.14"),
		"Rserve version must not be passed as '1.8.14' (unformatted)")
}

// ── JRubyRecipe ───────────────────────────────────────────────────────────────

func TestJRubyRecipeName(t *testing.T) {
	r := &recipe.JRubyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "jruby", r.Name())
}

func TestJRubyRecipeArtifact(t *testing.T) {
	r := &recipe.JRubyRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "linux", r.Artifact().OS)
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestJRubyRecipeBuild(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "jruby-9.4.5.0-ruby-3.1-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.JRubyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)

	// The recipe globs for jdk*/ inside JDKInstallDir after extracting the JDK
	// tarball. FakeRunner doesn't execute commands, so create the subdir manually.
	require.NoError(t, os.MkdirAll(filepath.Join(s.JRuby.JDKInstallDir, "jdk8u452"), 0755))

	src := newInput("jruby", "9.4.5.0", "https://repo1.maven.org/maven2/org/jruby/jruby-dist/9.4.5.0/jruby-dist-9.4.5.0-src.zip")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// JDK must be downloaded from the stack-configured URL.
	assert.True(t, hasDownload(f, s.JRuby.JDKURL), "should download JDK from stack JRuby.JDKURL")

	// Maven must be downloaded.
	assert.True(t, hasDownloadContaining(f, "apache-maven"), "should download Maven")

	// mvn must be invoked inside the correct source directory.
	assert.True(t, hasCallMatching(run.Calls, "sh", "cd /tmp/jruby-9.4.5.0"),
		"mvn must run inside srcDir")
	assert.True(t, hasCallMatching(run.Calls, "sh", "mvn"),
		"should invoke mvn")

	// Artifact must encode the full version including Ruby compatibility suffix.
	assert.True(t, hasCallMatching(run.Calls, "tar", "9.4.5.0-ruby-3.1"),
		"artifact should use full version 9.4.5.0-ruby-3.1")
}

func TestJRubyRecipeVersion93(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "jruby-9.3.14.0-ruby-2.6-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.JRubyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)

	require.NoError(t, os.MkdirAll(filepath.Join(s.JRuby.JDKInstallDir, "jdk8u452"), 0755))

	src := newInput("jruby", "9.3.14.0", "https://repo1.maven.org/maven2/org/jruby/jruby-dist/9.3.14.0/jruby-dist-9.3.14.0-src.zip")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// 9.3.x maps to Ruby 2.6.
	assert.True(t, hasCallMatching(run.Calls, "tar", "9.3.14.0-ruby-2.6"),
		"JRuby 9.3.x should produce artifact with ruby-2.6")
}

func TestJRubyRecipeUnknownVersion(t *testing.T) {
	f := newFakeFetcher()
	r := &recipe.JRubyRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	src := newInput("jruby", "9.9.0.0", "https://repo1.maven.org/maven2/org/jruby/jruby-dist/9.9.0.0/jruby-dist-9.9.0.0-src.zip")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9.9")
}

// ── HTTPDRecipe ───────────────────────────────────────────────────────────────

func TestHTTPDRecipeName(t *testing.T) {
	r := &recipe.HTTPDRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "httpd", r.Name())
}

func TestHTTPDRecipeArtifact(t *testing.T) {
	r := &recipe.HTTPDRecipe{Fetcher: newFakeFetcher()}
	assert.Equal(t, "linux", r.Artifact().OS)
	assert.Equal(t, "x64", r.Artifact().Arch)
}

func TestHTTPDRecipeBuild(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "httpd-2.4.58-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.HTTPDRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["httpd_build"] = []string{"libssl-dev", "libpcre3-dev", "libcjose-dev"}
	src := newInput("httpd", "2.4.58", "https://archive.apache.org/dist/httpd/httpd-2.4.58.tar.bz2")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// Should apt install httpd_build packages.
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "libssl-dev"),
		"should apt install httpd_build packages")

	// Should read GitHub release API 3 times (APR, APR-Iconv, APR-Util).
	// We verify via BodyMap keys — if the fetcher called ReadBody for those URLs
	// the recipe would have gotten valid JSON and proceeded without error.
	// Indirectly verified by no error and correct portile configure flags below.

	// Should configure HTTPD with --enable-mods-shared=reallyall.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--enable-mods-shared=reallyall"),
		"httpd configure should include --enable-mods-shared=reallyall")

	// Should configure HTTPD with --with-apr= pointing to the APR prefix.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--with-apr="),
		"httpd configure should include --with-apr=")

	// mod_auth_openidc configure should set APR_LIBS/APR_CFLAGS via env.
	assert.True(t, hasCallWithEnv(run.Calls, "sh", "APR_LIBS"),
		"mod_auth_openidc configure should have APR_LIBS env")
	assert.True(t, hasCallWithEnv(run.Calls, "sh", "APR_CFLAGS"),
		"mod_auth_openidc configure should have APR_CFLAGS env")

	// Artifact should be packed.
	assert.True(t, hasCallMatching(run.Calls, "tar", "httpd"),
		"should pack httpd artifact")
}

func TestHTTPDRecipeSetupTar(t *testing.T) {
	useTempWorkDir(t)
	writeFakeArtifact(t, "httpd-2.4.58-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.HTTPDRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["httpd_build"] = []string{"libssl-dev"}
	src := newInput("httpd", "2.4.58", "https://archive.apache.org/dist/httpd/httpd-2.4.58.tar.bz2")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// setup_tar should copy APR library.
	assert.True(t, hasCallMatching(run.Calls, "cp", "libapr-1.so.0"),
		"setup_tar should copy libapr-1.so.0")

	// setup_tar should copy APR-Util library.
	assert.True(t, hasCallMatching(run.Calls, "cp", "libaprutil-1.so.0"),
		"setup_tar should copy libaprutil-1.so.0")

	// setup_tar should copy APR-Iconv library.
	assert.True(t, hasCallMatching(run.Calls, "cp", "libapriconv-1.so.0"),
		"setup_tar should copy libapriconv-1.so.0")

	// setup_tar should copy system libs (cjose, hiredis, jansson).
	assert.True(t, hasCallMatching(run.Calls, "sh", "libcjose.so"),
		"setup_tar should copy libcjose.so*")
	assert.True(t, hasCallMatching(run.Calls, "sh", "libhiredis.so"),
		"setup_tar should copy libhiredis.so*")
	assert.True(t, hasCallMatching(run.Calls, "sh", "libjansson.so"),
		"setup_tar should copy libjansson.so*")

	// setup_tar should remove unneeded directories.
	assert.True(t, hasCallMatching(run.Calls, "rm", "cgi-bin"),
		"setup_tar should remove cgi-bin")
	assert.True(t, hasCallMatching(run.Calls, "rm", "manual"),
		"setup_tar should remove manual")
}

func TestHTTPDRecipeVersionsFromStackConfig(t *testing.T) {
	// Verify that APR sub-dep versions are read from the stack YAML config and
	// that configure calls use the plain version string (no leading 'v' prefix).
	useTempWorkDir(t)
	writeFakeArtifact(t, "httpd-2.4.58-linux-x64.tgz")

	f := newFakeFetcher()
	r := &recipe.HTTPDRecipe{Fetcher: f}
	run := runner.NewFakeRunner()
	s := newCompiledStack(t)
	s.AptPackages["httpd_build"] = []string{"libssl-dev"}
	src := newInput("httpd", "2.4.58", "https://archive.apache.org/dist/httpd/httpd-2.4.58.tar.bz2")

	err := r.Build(context.Background(), s, src, run, &output.OutData{})
	require.NoError(t, err)

	// The APR version from stack config must not include a leading 'v' prefix.
	assert.False(t, hasCallMatching(run.Calls, "./configure", "apr-v"),
		"configure prefix must not include a 'v' prefix")
	// A configure call for APR-Util or HTTPD must reference --with-apr.
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--with-apr"),
		"configure for APR-Util or HTTPD should reference --with-apr")
}

// ── Artifact naming sanity checks ─────────────────────────────────────────────

func TestCompiledRecipeArtifactMetaSanity(t *testing.T) {
	f := newFakeFetcher()
	cases := []struct {
		recipe   recipe.Recipe
		wantOS   string
		wantArch string
	}{
		{&recipe.RubyRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.BundlerRecipe{Fetcher: f}, "linux", "noarch"},
		{&recipe.PythonRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.NodeRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.GoRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.NginxRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.NginxStaticRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.OpenrestyRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.LibunwindRecipe{}, "linux", "noarch"},
		{&recipe.LibgdiplusRecipe{}, "linux", "noarch"},
		{&recipe.DepRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.GlideRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.GodepRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.HWCRecipe{Fetcher: f}, "windows", "x86-64"},
		{&recipe.RRecipe{Fetcher: f}, "linux", "noarch"},
		{&recipe.JRubyRecipe{Fetcher: f}, "linux", "x64"},
		{&recipe.HTTPDRecipe{Fetcher: f}, "linux", "x64"},
	}

	for _, tc := range cases {
		t.Run(tc.recipe.Name(), func(t *testing.T) {
			meta := tc.recipe.Artifact()
			assert.Equal(t, tc.wantOS, meta.OS, "wrong OS for %s", tc.recipe.Name())
			assert.Equal(t, tc.wantArch, meta.Arch, "wrong Arch for %s", tc.recipe.Name())
		})
	}
}
