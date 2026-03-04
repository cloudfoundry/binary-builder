package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/portile"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// HTTPDRecipe builds Apache HTTPD along with its full dependency chain:
// APR → APR-Iconv → APR-Util → HTTPD → mod_auth_openidc.
// APR/APR-Iconv/APR-Util/mod_auth_openidc versions, URLs and SHA256s are
// pinned in the stack YAML under httpd_sub_deps.
type HTTPDRecipe struct {
	Fetcher fetch.Fetcher
}

func (h *HTTPDRecipe) Name() string { return "httpd" }
func (h *HTTPDRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (h *HTTPDRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	a := apt.New(run)

	// Step 1: apt install httpd build dependencies from stack config.
	if err := a.Install(ctx, s.AptPackages["httpd_build"]...); err != nil {
		return fmt.Errorf("httpd: apt install httpd_build: %w", err)
	}

	// Step 2: Create /app directory.
	if err := run.Run("mkdir", "-p", "/app"); err != nil {
		return fmt.Errorf("httpd: mkdir /app: %w", err)
	}

	// Step 3: Read sub-dep config from stack YAML.
	aprDep := s.HTTPDSubDeps.APR
	aprIconvDep := s.HTTPDSubDeps.APRIconv
	aprUtilDep := s.HTTPDSubDeps.APRUtil

	// Step 4: Build APR.
	aprPrefix := fmt.Sprintf("/tmp/apr-%s-prefix", aprDep.Version)
	aprPortile := &portile.Portile{
		Name:     "apr",
		Version:  aprDep.Version,
		URL:      aprDep.URL,
		Checksum: source.Checksum{Algorithm: "sha256", Value: aprDep.SHA256},
		Prefix:   aprPrefix,
		Runner:   run,
		Fetcher:  h.Fetcher,
	}
	if err := aprPortile.Cook(ctx); err != nil {
		return fmt.Errorf("httpd: building APR: %w", err)
	}

	// Step 5: Build APR-Iconv (depends on APR).
	aprIconvPrefix := fmt.Sprintf("/tmp/apr-iconv-%s-prefix", aprIconvDep.Version)
	aprIconvPortile := &portile.Portile{
		Name:     "apr-iconv",
		Version:  aprIconvDep.Version,
		URL:      aprIconvDep.URL,
		Checksum: source.Checksum{Algorithm: "sha256", Value: aprIconvDep.SHA256},
		Prefix:   aprIconvPrefix,
		Options: []string{
			fmt.Sprintf("--with-apr=%s/bin/apr-1-config", aprPrefix),
		},
		Runner:  run,
		Fetcher: h.Fetcher,
	}
	if err := aprIconvPortile.Cook(ctx); err != nil {
		return fmt.Errorf("httpd: building APR-Iconv: %w", err)
	}

	// Step 6: Build APR-Util (depends on APR + APR-Iconv).
	aprUtilPrefix := fmt.Sprintf("/tmp/apr-util-%s-prefix", aprUtilDep.Version)
	aprUtilPortile := &portile.Portile{
		Name:     "apr-util",
		Version:  aprUtilDep.Version,
		URL:      aprUtilDep.URL,
		Checksum: source.Checksum{Algorithm: "sha256", Value: aprUtilDep.SHA256},
		Prefix:   aprUtilPrefix,
		Options: []string{
			fmt.Sprintf("--with-apr=%s", aprPrefix),
			fmt.Sprintf("--with-iconv=%s", aprIconvPrefix),
			"--with-crypto",
			"--with-openssl",
			"--with-mysql",
			"--with-pgsql",
			"--with-gdbm",
			"--with-ldap",
		},
		Runner:  run,
		Fetcher: h.Fetcher,
	}
	if err := aprUtilPortile.Cook(ctx); err != nil {
		return fmt.Errorf("httpd: building APR-Util: %w", err)
	}

	// Step 7: Build HTTPD (depends on APR + APR-Iconv + APR-Util).
	httpdPrefix := "/app/httpd"
	httpdPortile := &portile.Portile{
		Name:     "httpd",
		Version:  src.Version,
		URL:      fmt.Sprintf("https://archive.apache.org/dist/httpd/httpd-%s.tar.bz2", src.Version),
		Checksum: src.PrimaryChecksum(),
		Prefix:   httpdPrefix,
		Options: []string{
			fmt.Sprintf("--with-apr=%s", aprPrefix),
			fmt.Sprintf("--with-apr-util=%s", aprUtilPrefix),
			"--with-ssl=/usr/lib/x86_64-linux-gnu",
			"--enable-mpms-shared=worker event",
			"--enable-mods-shared=reallyall",
			"--disable-isapi",
			"--disable-dav",
			"--disable-dialup",
		},
		Runner:  run,
		Fetcher: h.Fetcher,
	}
	if err := httpdPortile.Cook(ctx); err != nil {
		return fmt.Errorf("httpd: building HTTPD: %w", err)
	}

	// Step 7b: apt install mod_auth_openidc dependencies AFTER httpd is built.
	// These must be installed after httpd so that jansson/cjose are NOT available
	// during the HTTPD configure step — otherwise mod_md.so gets compiled (since
	// mod_md depends on jansson), creating a file-list mismatch with the Ruby build.
	if err := a.Install(ctx, s.AptPackages["httpd_mod_auth_build"]...); err != nil {
		return fmt.Errorf("httpd: apt install httpd_mod_auth_build: %w", err)
	}

	// Step 8: Build mod_auth_openidc (depends on HTTPD + APR).
	// APR_LIBS and APR_CFLAGS are set in the environment for ./configure.
	aprLibs := fmt.Sprintf("`%s/bin/apr-1-config --link-ld --libs`", aprPrefix)
	aprCFlags := fmt.Sprintf("`%s/bin/apr-1-config --cflags --includes`", aprPrefix)
	modAuthEnv := map[string]string{
		"APR_LIBS":   aprLibs,
		"APR_CFLAGS": aprCFlags,
	}
	modAuthDep := s.HTTPDSubDeps.ModAuthOpenidc
	modAuthChecksum := source.Checksum{Algorithm: "sha256", Value: modAuthDep.SHA256}
	modAuthPrefix := fmt.Sprintf("/tmp/mod_auth_openidc-%s-prefix", modAuthDep.Version)

	// We need to pass env vars to the configure step. The portile package runs
	// configure via RunInDir without env, so we handle mod_auth_openidc manually:
	// download, extract, configure with env, make, make install.
	if err := h.buildModAuthOpenidc(ctx, run, modAuthDep.URL, modAuthDep.Version, modAuthPrefix, httpdPrefix, modAuthEnv, modAuthChecksum); err != nil {
		return fmt.Errorf("httpd: building mod_auth_openidc: %w", err)
	}

	// Step 9: setup_tar — copy shared libraries into the httpd prefix lib/ dir.
	if err := h.setupTar(run, httpdPrefix, aprPrefix, aprUtilPrefix, aprIconvPrefix); err != nil {
		return fmt.Errorf("httpd: setup_tar: %w", err)
	}

	// Step 10: Pack the httpd prefix into the artifact tarball.
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("httpd-%s-linux-x64.tgz", src.Version))
	if err := run.Run("tar", "czf", artifactPath, "-C", "/app", "httpd"); err != nil {
		return fmt.Errorf("httpd: packing artifact: %w", err)
	}

	// Step 11: Strip top-level directory from the artifact.
	if err := archive.StripTopLevelDir(artifactPath); err != nil {
		return fmt.Errorf("httpd: stripping top-level dir: %w", err)
	}

	return nil
}

// buildModAuthOpenidc manually handles the configure/make/install cycle for
// mod_auth_openidc, passing APR_LIBS and APR_CFLAGS via RunWithEnv.
func (h *HTTPDRecipe) buildModAuthOpenidc(
	ctx context.Context,
	run runner.Runner,
	url, version, prefix, httpdPrefix string,
	env map[string]string,
	checksum source.Checksum,
) error {
	// Download tarball.
	tarball := fmt.Sprintf("/tmp/mod_auth_openidc-%s.tar.gz", version)
	if err := h.Fetcher.Download(ctx, url, tarball, checksum); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Create temp dir and extract.
	tmpDir := fmt.Sprintf("/tmp/mod_auth_openidc-%s-build", version)
	if err := run.Run("mkdir", "-p", tmpDir); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := run.Run("tar", "xf", tarball, "-C", tmpDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	srcDir := fmt.Sprintf("%s/mod_auth_openidc-%s", tmpDir, version)

	// Configure with env (APR_LIBS + APR_CFLAGS), passing --with-apxs2.
	configureCmd := fmt.Sprintf(
		"./configure --prefix=%s --with-apxs2=%s/bin/apxs",
		prefix, httpdPrefix,
	)
	if err := run.RunWithEnv(env, "sh", "-c",
		fmt.Sprintf("cd %s && %s", srcDir, configureCmd)); err != nil {
		return fmt.Errorf("configure: %w", err)
	}

	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("make: %w", err)
	}

	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("make install: %w", err)
	}

	return nil
}

// setupTar copies the runtime shared libraries into the httpd prefix lib/ dir,
// mirroring the Ruby httpd_meal.rb setup_tar method.
func (h *HTTPDRecipe) setupTar(run runner.Runner, httpdPrefix, aprPrefix, aprUtilPrefix, aprIconvPrefix string) error {
	libDir := fmt.Sprintf("%s/lib", httpdPrefix)
	aprUtilLibDir := fmt.Sprintf("%s/lib/apr-util-1", httpdPrefix)
	iconvLibDir := fmt.Sprintf("%s/lib/iconv", httpdPrefix)

	// Remove unneeded directories.
	for _, dir := range []string{"cgi-bin", "error", "icons", "include", "man", "manual", "htdocs"} {
		if err := run.Run("rm", "-rf", fmt.Sprintf("%s/%s", httpdPrefix, dir)); err != nil {
			return fmt.Errorf("rm %s: %w", dir, err)
		}
	}

	// Remove conf files but keep the conf/ directory.
	if err := run.Run("sh", "-c", fmt.Sprintf(
		"rm -rf %s/conf/extra/* %s/conf/httpd.conf %s/conf/httpd.conf.bak %s/conf/magic %s/conf/original",
		httpdPrefix, httpdPrefix, httpdPrefix, httpdPrefix, httpdPrefix,
	)); err != nil {
		return fmt.Errorf("cleaning conf: %w", err)
	}

	// Create lib subdirs.
	for _, dir := range []string{libDir, aprUtilLibDir, iconvLibDir} {
		if err := run.Run("mkdir", "-p", dir); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Copy APR runtime library.
	if err := run.Run("cp", fmt.Sprintf("%s/lib/libapr-1.so.0", aprPrefix), libDir); err != nil {
		return fmt.Errorf("cp libapr: %w", err)
	}

	// Copy APR-Util runtime library.
	if err := run.Run("cp", fmt.Sprintf("%s/lib/libaprutil-1.so.0", aprUtilPrefix), libDir); err != nil {
		return fmt.Errorf("cp libaprutil: %w", err)
	}

	// Copy APR-Util plugins (apr-util-1/*.so).
	if err := run.Run("sh", "-c", fmt.Sprintf(
		"cp %s/lib/apr-util-1/*.so %s/",
		aprUtilPrefix, aprUtilLibDir,
	)); err != nil {
		return fmt.Errorf("cp apr-util-1 plugins: %w", err)
	}

	// Copy APR-Iconv library.
	if err := run.Run("cp", fmt.Sprintf("%s/lib/libapriconv-1.so.0", aprIconvPrefix), libDir); err != nil {
		return fmt.Errorf("cp libapriconv: %w", err)
	}

	// Copy APR-Iconv converters (iconv/*.so).
	if err := run.Run("sh", "-c", fmt.Sprintf(
		"cp %s/lib/iconv/*.so %s/",
		aprIconvPrefix, iconvLibDir,
	)); err != nil {
		return fmt.Errorf("cp iconv plugins: %w", err)
	}

	// Copy system shared libraries (cjose, hiredis, jansson).
	for _, pattern := range []string{
		"/usr/lib/x86_64-linux-gnu/libcjose.so*",
		"/usr/lib/x86_64-linux-gnu/libhiredis.so*",
		"/usr/lib/x86_64-linux-gnu/libjansson.so*",
	} {
		if err := run.Run("sh", "-c", fmt.Sprintf("cp %s %s/", pattern, libDir)); err != nil {
			return fmt.Errorf("cp %s: %w", pattern, err)
		}
	}

	return nil
}
