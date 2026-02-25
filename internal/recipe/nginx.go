package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/gpg"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// nginxGPGKeys are the 6 nginx signing keys used for tarball verification.
var nginxGPGKeys = []string{
	"http://nginx.org/keys/maxim.key",
	"http://nginx.org/keys/arut.key",
	"https://nginx.org/keys/pluknet.key",
	"http://nginx.org/keys/sb.key",
	"http://nginx.org/keys/thresh.key",
	"https://nginx.org/keys/nginx_signing.key",
}

// nginxBaseConfigureArgs are the configure options shared between nginx and nginx-static.
// These match the Ruby build_nginx_helper base_nginx_options exactly.
var nginxBaseConfigureArgs = []string{
	"--prefix=/",
	"--error-log-path=stderr",
	"--with-http_ssl_module",
	"--with-http_v2_module",
	"--with-http_realip_module",
	"--with-http_gunzip_module",
	"--with-http_gzip_static_module",
	"--with-http_auth_request_module",
	"--with-http_random_index_module",
	"--with-http_secure_link_module",
	"--with-http_stub_status_module",
	"--without-http_uwsgi_module",
	"--without-http_scgi_module",
	"--with-pcre",
	"--with-pcre-jit",
	"--with-debug",
}

// NginxRecipe builds nginx with PIC flags and dynamic modules.
type NginxRecipe struct {
	Fetcher fetch.Fetcher
}

func (n *NginxRecipe) Name() string { return "nginx" }
func (n *NginxRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (n *NginxRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	return buildNginxVariant(ctx, src, run, n.Fetcher, false)
}

// buildNginxVariant implements the shared nginx/nginx-static build logic.
//
// Ruby uses `--prefix=/` with `DESTDIR={destDir}/nginx` for make install, so the
// install tree ends up at {destDir}/nginx/{sbin,modules,…}.
// Then html/ and conf/ are removed from that tree and an empty conf/ recreated.
//
// Packing differs by variant:
//   - nginx-static: tars the `nginx` directory (from inside destDir), producing
//     an archive whose top-level entry is `nginx/`. The outer builder strips that.
//   - nginx (non-static): tars `.` from inside destDir (which contains `nginx/`),
//     producing an archive whose top-level entry is `nginx/`. The outer builder strips that.
//
// isStatic=true uses PIE flags; isStatic=false uses PIC flags + dynamic modules.
func buildNginxVariant(ctx context.Context, src *source.Input, run runner.Runner, fetcher fetch.Fetcher, isStatic bool) error {
	version := src.Version
	srcTarURL := src.URL
	sigURL := src.URL + ".asc"

	// Verify GPG signature of the tarball.
	if err := gpg.VerifySignature(ctx, srcTarURL, sigURL, nginxGPGKeys, run); err != nil {
		return fmt.Errorf("nginx: GPG verification: %w", err)
	}

	srcDir := fmt.Sprintf("/tmp/nginx-%s", version)
	// destDir is the DESTDIR for make install. With --prefix=/ the nginx tree
	// is installed to {destDir}/nginx/{sbin,modules,...}.
	destDir := fmt.Sprintf("/tmp/nginx-install-%s", version)
	nginxInstallDir := fmt.Sprintf("%s/nginx", destDir)

	// Download and extract source.
	srcTarball := fmt.Sprintf("/tmp/nginx-%s.tar.gz", version)
	if err := fetcher.Download(ctx, srcTarURL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("nginx: downloading source: %w", err)
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
		return fmt.Errorf("nginx: extracting source: %w", err)
	}

	// Build configure args. Base args include --prefix=/ (Ruby parity).
	// Custom args are prepended before the base args (matching Ruby's options order).
	var customArgs []string
	if isStatic {
		// nginx-static: PIE flags, minimal modules (no dynamic modules).
		customArgs = []string{
			"--with-cc-opt=-fPIE -pie",
			"--with-ld-opt=-fPIE -pie -z now",
		}
	} else {
		// nginx: PIC flags + dynamic modules.
		customArgs = []string{
			"--with-cc-opt=-fPIC -pie",
			"--with-ld-opt=-fPIC -pie -z now",
			"--with-compat",
			"--with-mail=dynamic",
			"--with-mail_ssl_module",
			"--with-stream=dynamic",
			"--with-http_sub_module",
		}
	}
	configureArgs := append(nginxBaseConfigureArgs, customArgs...)

	if err := run.RunInDir(srcDir, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("nginx: configure: %w", err)
	}

	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("nginx: make: %w", err)
	}

	if err := run.Run("mkdir", "-p", nginxInstallDir); err != nil {
		return err
	}

	// make install with DESTDIR so --prefix=/ resolves relative to destDir.
	if err := run.RunWithEnv(
		map[string]string{"DESTDIR": nginxInstallDir},
		"make", "-C", srcDir, "install",
	); err != nil {
		return fmt.Errorf("nginx: make install: %w", err)
	}

	// Remove html/ and conf/ from install tree, recreate empty conf/.
	if err := run.Run("rm", "-rf",
		fmt.Sprintf("%s/html", nginxInstallDir),
		fmt.Sprintf("%s/conf", nginxInstallDir),
	); err != nil {
		return fmt.Errorf("nginx: removing html/conf: %w", err)
	}
	if err := run.Run("mkdir", "-p", fmt.Sprintf("%s/conf", nginxInstallDir)); err != nil {
		return err
	}

	// Pack the artifact.
	// Both variants produce an archive with `nginx/` as the top-level dir.
	// Recipes write artifacts to an absolute path in CWD so main.go can find them.
	var artifactBasename string
	if isStatic {
		artifactBasename = fmt.Sprintf("nginx-static-%s-linux-x64.tgz", version)
	} else {
		artifactBasename = fmt.Sprintf("nginx-%s-linux-x64.tgz", version)
	}
	artifactPath := filepath.Join(mustCwd(), artifactBasename)

	if isStatic {
		// Pack `nginx` dir from inside destDir → top-level entry is nginx/.
		if err := run.RunInDir(destDir, "tar", "czf", artifactPath, "nginx"); err != nil {
			return fmt.Errorf("nginx-static: packing artifact: %w", err)
		}
	} else {
		// Pack `.` from inside destDir → top-level entry is nginx/ (only dir present).
		if err := run.RunInDir(destDir, "tar", "czf", artifactPath, "."); err != nil {
			return fmt.Errorf("nginx: packing artifact: %w", err)
		}
	}

	return archive.StripTopLevelDir(artifactPath)
}
