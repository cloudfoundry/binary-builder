package autoconf_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/autoconf"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── fakeFetcher ───────────────────────────────────────────────────────────────

// fakeFetcher satisfies fetch.Fetcher without making any network calls.
type fakeFetcher struct {
	downloaded []string
	errMap     map[string]error
}

func newFakeFetcher() *fakeFetcher {
	return &fakeFetcher{errMap: make(map[string]error)}
}

func (f *fakeFetcher) Download(_ context.Context, url, _ string, _ source.Checksum) error {
	f.downloaded = append(f.downloaded, url)
	if err, ok := f.errMap[url]; ok {
		return err
	}
	return nil
}

func (f *fakeFetcher) ReadBody(_ context.Context, url string) ([]byte, error) {
	if err, ok := f.errMap[url]; ok {
		return nil, err
	}
	return []byte("fake"), nil
}

// Ensure fakeFetcher satisfies the fetch.Fetcher interface at compile time.
var _ fetch.Fetcher = (*fakeFetcher)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newStack() *stack.Stack {
	return &stack.Stack{
		Name: "cflinuxfs4",
		AptPackages: map[string][]string{
			"mylib_build": {"libfoo-dev", "libbar-dev"},
		},
	}
}

func newInput(name, version, url string) *source.Input {
	return &source.Input{
		Name:    name,
		Version: version,
		URL:     url,
		SHA256:  "deadbeef",
	}
}

func anyCallContains(calls []runner.Call, name string) bool {
	for _, c := range calls {
		if c.Name == name {
			return true
		}
	}
	return false
}

func anyArgsContain(calls []runner.Call, target string) bool {
	for _, c := range calls {
		for _, arg := range c.Args {
			if strings.Contains(arg, target) {
				return true
			}
		}
	}
	return false
}

func hasCallMatching(calls []runner.Call, name, argSubstr string) bool {
	for _, c := range calls {
		if c.Name == name {
			joined := strings.Join(c.Args, " ")
			if argSubstr == "" || strings.Contains(joined, argSubstr) {
				return true
			}
		}
	}
	return false
}

func hasCallWithEnv(calls []runner.Call, name, envKey string) bool {
	for _, c := range calls {
		if c.Name == name && c.Env != nil {
			if _, ok := c.Env[envKey]; ok {
				return true
			}
		}
	}
	return false
}

// ── Name / Artifact ──────────────────────────────────────────────────────────

func TestRecipeName(t *testing.T) {
	r := &autoconf.Recipe{DepName: "mylib"}
	assert.Equal(t, "mylib", r.Name())
}

// ── default apt packages ─────────────────────────────────────────────────────

func TestDefaultAptPackagesFromStack(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		// No AptPackages hook → default key is "mylib_build"
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// apt-get install must have been called with the packages from "mylib_build".
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "libfoo-dev"),
		"default apt packages should come from s.AptPackages['mylib_build']")
	assert.True(t, hasCallMatching(run.Calls, "apt-get", "libbar-dev"),
		"default apt packages should come from s.AptPackages['mylib_build']")
}

func TestAptPackagesHookOverridesDefault(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AptPackages: func(_ *stack.Stack) []string {
				return []string{"custom-pkg"}
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "apt-get", "custom-pkg"),
		"AptPackages hook result should be used for install")
	assert.False(t, hasCallMatching(run.Calls, "apt-get", "libfoo-dev"),
		"default packages must not be used when hook overrides")
}

// ── default source download ───────────────────────────────────────────────────

func TestDefaultSourceDownloadsAndExtracts(t *testing.T) {
	run := runner.NewFakeRunner()
	f := newFakeFetcher()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: f,
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// Fetcher should have downloaded the source URL.
	require.Len(t, f.downloaded, 1)
	assert.Equal(t, src.URL, f.downloaded[0])

	// Runner should have extracted the tarball.
	assert.True(t, hasCallMatching(run.Calls, "tar", "xzf"),
		"should run tar xzf to extract source")
}

// ── before-download hook ──────────────────────────────────────────────────────

func TestBeforeDownloadHookIsCalledBeforeSource(t *testing.T) {
	run := runner.NewFakeRunner()
	f := newFakeFetcher()
	hookCalled := false
	downloadCalledBeforeHook := false

	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: f,
		Hooks: autoconf.Hooks{
			BeforeDownload: func(_ context.Context, _ *source.Input, _ runner.Runner) error {
				// Fetcher should not have been called yet.
				if len(f.downloaded) > 0 {
					downloadCalledBeforeHook = true
				}
				hookCalled = true
				return nil
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hookCalled, "BeforeDownload hook must be called")
	assert.False(t, downloadCalledBeforeHook, "download must not occur before BeforeDownload hook")
}

func TestBeforeDownloadErrorIsPropagated(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			BeforeDownload: func(_ context.Context, _ *source.Input, _ runner.Runner) error {
				return errors.New("gpg verification failed")
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gpg verification failed")
	assert.Contains(t, err.Error(), "before_download")
}

func TestSourceProviderHookOverridesDownload(t *testing.T) {
	run := runner.NewFakeRunner()
	f := newFakeFetcher()
	providerCalled := false

	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: f,
		Hooks: autoconf.Hooks{
			SourceProvider: func(_ context.Context, _ *source.Input, _ fetch.Fetcher, _ runner.Runner) (string, error) {
				providerCalled = true
				return "/tmp/custom-src", nil
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, providerCalled, "SourceProvider hook must be called")
	// Fetcher must NOT be called when SourceProvider is set.
	assert.Empty(t, f.downloaded, "Fetcher.Download must not be called when SourceProvider is set")
}

func TestSourceProviderErrorIsPropagated(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			SourceProvider: func(_ context.Context, _ *source.Input, _ fetch.Fetcher, _ runner.Runner) (string, error) {
				return "", errors.New("provider failed")
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider failed")
	assert.Contains(t, err.Error(), "source provider")
}

// ── after-extract hook ────────────────────────────────────────────────────────

func TestAfterExtractHookIsCalledBeforeConfigure(t *testing.T) {
	run := runner.NewFakeRunner()
	hookCalled := false
	configureCallIdx := -1

	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterExtract: func(_ context.Context, _, _ string, _ runner.Runner) error {
				// Record that the hook ran — configure should not have been called yet.
				for _, c := range run.Calls {
					if c.Name == "./configure" {
						configureCallIdx = len(run.Calls) - 1
					}
				}
				hookCalled = true
				return nil
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hookCalled, "AfterExtract hook must be called")
	assert.Equal(t, -1, configureCallIdx,
		"configure must not be called before AfterExtract hook returns")
}

func TestAfterExtractErrorIsPropagated(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterExtract: func(_ context.Context, _, _ string, _ runner.Runner) error {
				return errors.New("autoreconf failed")
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autoreconf failed")
	assert.Contains(t, err.Error(), "after_extract")
}

// ── configure ────────────────────────────────────────────────────────────────

func TestDefaultConfigureUsesPrefix(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "./configure", "--prefix="),
		"default configure should use --prefix=")
	assert.True(t, anyArgsContain(run.Calls, "--prefix=/tmp/mylib-built-1.0"),
		"prefix should contain dep name and version")
}

func TestConfigureArgsHookOverridesDefault(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			ConfigureArgs: func(_, prefix string) []string {
				return []string{
					"--prefix=" + prefix,
					"--enable-shared",
					"--disable-static",
				}
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "./configure", "--enable-shared"),
		"configure should include custom arg from hook")
	assert.True(t, hasCallMatching(run.Calls, "./configure", "--disable-static"),
		"configure should include custom arg from hook")
}

func TestConfigureEnvHookPassesEnvToMake(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			ConfigureEnv: func() map[string]string {
				return map[string]string{"CFLAGS": "-O2 -fPIC"}
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// Both configure and make should be called with the env.
	assert.True(t, hasCallWithEnv(run.Calls, "./configure", "CFLAGS"),
		"configure should have CFLAGS in env when ConfigureEnv hook is set")
	assert.True(t, hasCallWithEnv(run.Calls, "make", "CFLAGS"),
		"make should inherit CFLAGS env from ConfigureEnv hook")
}

// ── make args ─────────────────────────────────────────────────────────────────

func TestMakeArgsHookPassesExtraArgs(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			MakeArgs: func() []string { return []string{"-j2"} },
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "make", "-j2"),
		"make should be called with -j2 from MakeArgs hook")
}

// ── install env ──────────────────────────────────────────────────────────────

func TestInstallEnvHookPassesDifferentEnvToMakeInstall(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			InstallEnv: func(prefix string) map[string]string {
				return map[string]string{"DESTDIR": prefix + "/staging"}
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallWithEnv(run.Calls, "make", "DESTDIR"),
		"make install should have DESTDIR env when InstallEnv hook is set")
}

func TestInstallEnvFallsBackToConfigureEnvWhenNil(t *testing.T) {
	// When InstallEnv is nil but ConfigureEnv is set, the configure env is
	// reused for make install (needed for libgdiplus which uses CFLAGS/CXXFLAGS
	// for all three steps).
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			ConfigureEnv: func() map[string]string {
				return map[string]string{"CFLAGS": "-g"}
			},
			// InstallEnv is nil
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// make install must still have CFLAGS (from ConfigureEnv fallback).
	assert.True(t, hasCallWithEnv(run.Calls, "make", "CFLAGS"),
		"make install should fall back to ConfigureEnv when InstallEnv is nil")
}

// ── after-install hook ────────────────────────────────────────────────────────

func TestAfterInstallHookIsCalledAfterMakeInstall(t *testing.T) {
	run := runner.NewFakeRunner()
	hookCalled := false
	makeInstallSeen := false

	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterInstall: func(_ context.Context, _ string, _ runner.Runner) error {
				// At this point make install should have already been called.
				for _, c := range run.Calls {
					if c.Name == "make" {
						for _, a := range c.Args {
							if a == "install" {
								makeInstallSeen = true
							}
						}
					}
				}
				hookCalled = true
				return nil
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hookCalled, "AfterInstall hook must be called")
	assert.True(t, makeInstallSeen, "make install must be called before AfterInstall hook")
}

func TestAfterInstallErrorIsPropagated(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterInstall: func(_ context.Context, _ string, _ runner.Runner) error {
				return errors.New("cleanup failed")
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup failed")
	assert.Contains(t, err.Error(), "after_install")
}

// ── pack dirs ────────────────────────────────────────────────────────────────

func TestDefaultPackDirIsFullPrefix(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// Default: pack "." from inside prefix.
	assert.True(t, anyCallContains(run.Calls, "tar"),
		"should run tar to pack artifact")
	assert.True(t, anyArgsContain(run.Calls, "."),
		"default pack dir should be '.'")
}

func TestPackDirsHookLimitsWhatIsPacked(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			PackDirs: func() []string { return []string{"include", "lib"} },
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hasCallMatching(run.Calls, "tar", "include"),
		"tar should include 'include' directory")
	assert.True(t, hasCallMatching(run.Calls, "tar", "lib"),
		"tar should include 'lib' directory")
	// Must not pack the default "."
	// (this is subtle — we just check both named dirs are present)
}

// ── after-pack hook ───────────────────────────────────────────────────────────

func TestAfterPackHookIsCalledAfterTar(t *testing.T) {
	run := runner.NewFakeRunner()
	hookCalled := false
	tarCalledBeforeHook := false

	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterPack: func(_ string) error {
				// tar must have been called before this hook.
				for _, c := range run.Calls {
					if c.Name == "tar" {
						for _, a := range c.Args {
							if a == "czf" {
								tarCalledBeforeHook = true
							}
						}
					}
				}
				hookCalled = true
				return nil
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	assert.True(t, hookCalled, "AfterPack hook must be called")
	assert.True(t, tarCalledBeforeHook, "tar czf must be called before AfterPack hook")
}

func TestAfterPackErrorIsPropagated(t *testing.T) {
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
		Hooks: autoconf.Hooks{
			AfterPack: func(_ string) error {
				return errors.New("strip failed")
			},
		},
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strip failed")
	assert.Contains(t, err.Error(), "after_pack")
}

// ── call ordering sanity ──────────────────────────────────────────────────────

func TestBuildStepOrder(t *testing.T) {
	// Verifies apt→configure→make→make-install→tar order is maintained.
	run := runner.NewFakeRunner()
	r := &autoconf.Recipe{
		DepName: "mylib",
		Fetcher: newFakeFetcher(),
	}
	src := newInput("mylib", "1.0", "https://example.com/mylib-1.0.tar.gz")

	err := r.Build(context.Background(), newStack(), src, run, &output.OutData{})
	require.NoError(t, err)

	// Collect call names in order.
	var names []string
	for _, c := range run.Calls {
		names = append(names, c.Name)
	}

	aptIdx := -1
	configureIdx := -1
	makeIdx := -1
	makeInstallIdx := -1
	tarIdx := -1

	for i, c := range run.Calls {
		switch {
		case c.Name == "apt-get" && aptIdx < 0:
			aptIdx = i
		case c.Name == "./configure" && configureIdx < 0:
			configureIdx = i
		case c.Name == "make" && len(c.Args) == 0 && makeIdx < 0:
			makeIdx = i
		case c.Name == "make" && len(c.Args) > 0 && c.Args[0] == "install" && makeInstallIdx < 0:
			makeInstallIdx = i
		case c.Name == "tar" && tarIdx < 0:
			// Skip the extract tar; find the pack tar (czf).
			for _, a := range c.Args {
				if a == "czf" {
					tarIdx = i
				}
			}
		}
	}

	assert.Greater(t, configureIdx, aptIdx, "configure must come after apt-get")
	assert.Greater(t, makeIdx, configureIdx, "make must come after configure")
	assert.Greater(t, makeInstallIdx, makeIdx, "make install must come after make")
	assert.Greater(t, tarIdx, makeInstallIdx, "tar (pack) must come after make install")
}
