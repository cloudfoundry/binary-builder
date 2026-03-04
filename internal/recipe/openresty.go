package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// OpenrestyRecipe builds OpenResty (nginx + Lua) via configure/make.
// No GPG verification (known gap, carried forward from Ruby code as a TODO).
type OpenrestyRecipe struct {
	Fetcher fetch.Fetcher
}

func (o *OpenrestyRecipe) Name() string { return "openresty" }
func (o *OpenrestyRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (o *OpenrestyRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version
	srcTarball := fmt.Sprintf("/tmp/openresty-%s.tar.gz", version)
	srcDir := fmt.Sprintf("/tmp/openresty-%s", version)
	destDir := fmt.Sprintf("/tmp/openresty-install-%s", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("openresty-%s-linux-x64.tgz", version))

	// Download source via Fetcher (no GPG verification — TODO: add when keys are published).
	if err := o.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("openresty: downloading source: %w", err)
	}

	if err := run.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
		return fmt.Errorf("openresty: extracting source: %w", err)
	}

	// Configure with PIC flags and -j2 parallelism.
	// Flags match Ruby builder.rb build_openresty exactly.
	prefix := fmt.Sprintf("%s/openresty", destDir)
	configureArgs := []string{
		fmt.Sprintf("--prefix=%s", prefix),
		"-j2",
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
		"--with-cc-opt=-fPIC -pie",
		"--with-ld-opt=-fPIC -pie -z now",
		"--with-compat",
		"--with-mail=dynamic",
		"--with-mail_ssl_module",
		"--with-stream=dynamic",
	}

	if err := run.RunInDir(srcDir, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("openresty: configure: %w", err)
	}

	if err := run.RunInDir(srcDir, "make", "-j2"); err != nil {
		return fmt.Errorf("openresty: make: %w", err)
	}

	if err := run.Run("mkdir", "-p", prefix); err != nil {
		return err
	}

	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("openresty: make install: %w", err)
	}

	// Clean up: remove nginx/html, nginx/conf, bin/openresty; recreate nginx/conf.
	nginxDir := fmt.Sprintf("%s/nginx", prefix)
	if err := run.Run("rm", "-rf",
		fmt.Sprintf("%s/html", nginxDir),
		fmt.Sprintf("%s/conf", nginxDir),
		fmt.Sprintf("%s/bin/openresty", prefix),
	); err != nil {
		return fmt.Errorf("openresty: cleanup: %w", err)
	}
	if err := run.Run("mkdir", "-p", fmt.Sprintf("%s/conf", nginxDir)); err != nil {
		return err
	}

	// Pack from inside the openresty install dir.
	if err := run.Run("tar", "czf", artifactPath, "-C", prefix, "."); err != nil {
		return fmt.Errorf("openresty: packing artifact: %w", err)
	}

	return nil
}
