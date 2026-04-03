package recipe

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/autoconf"
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

func (n *NginxRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return newNginxAutoconf(n.Fetcher, false).Build(ctx, s, src, run, out)
}

// NginxStaticRecipe builds nginx with PIE flags and a minimal module set.
type NginxStaticRecipe struct {
	Fetcher fetch.Fetcher
}

func (n *NginxStaticRecipe) Name() string { return "nginx-static" }
func (n *NginxStaticRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (n *NginxStaticRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return newNginxAutoconf(n.Fetcher, true).Build(ctx, s, src, run, out)
}

// newNginxAutoconf constructs the AutoconfRecipe for nginx or nginx-static.
//
// isStatic=true → PIE flags, minimal modules (nginx-static)
// isStatic=false → PIC flags + dynamic modules (nginx)
//
// Install layout: --prefix=/ with DESTDIR=<prefix>/nginx causes the install
// tree to land at <prefix>/nginx/{sbin,modules,...}.
// AfterPack strips the top-level dir so the final artifact's root is nginx/.
func newNginxAutoconf(fetcher fetch.Fetcher, isStatic bool) *autoconf.Recipe {
	depName := "nginx"
	if isStatic {
		depName = "nginx-static"
	}

	return &autoconf.Recipe{
		DepName: depName,
		Fetcher: fetcher,
		Hooks: autoconf.Hooks{
			// No apt packages needed for nginx.
			AptPackages: func(_ *stack.Stack) []string { return nil },

			// BeforeDownload verifies the GPG signature of the source tarball.
			BeforeDownload: func(ctx context.Context, src *source.Input, r runner.Runner) error {
				sigURL := src.URL + ".asc"
				return gpg.VerifySignature(ctx, src.URL, sigURL, nginxGPGKeys, r)
			},

			// SourceProvider downloads and extracts the nginx source tarball.
			SourceProvider: func(ctx context.Context, src *source.Input, f fetch.Fetcher, r runner.Runner) (string, error) {
				version := src.Version
				srcTarball := fmt.Sprintf("/tmp/nginx-%s.tar.gz", version)
				if err := f.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
					return "", fmt.Errorf("downloading source: %w", err)
				}
				if err := r.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
					return "", fmt.Errorf("extracting source: %w", err)
				}
				return fmt.Sprintf("/tmp/nginx-%s", src.Version), nil
			},

			// AfterExtract creates the DESTDIR install directory before make install.
			AfterExtract: func(_ context.Context, _, prefix string, r runner.Runner) error {
				nginxInstallDir := fmt.Sprintf("%s/nginx", prefix)
				return r.Run("mkdir", "-p", nginxInstallDir)
			},

			// ConfigureArgs returns the full nginx configure flags for this variant.
			// Base args include --prefix=/ (Ruby parity); variant args follow.
			ConfigureArgs: func(_, _ string) []string {
				var variantArgs []string
				if isStatic {
					variantArgs = []string{
						"--with-cc-opt=-fPIE -pie",
						"--with-ld-opt=-fPIE -pie -z now",
					}
				} else {
					variantArgs = []string{
						"--with-cc-opt=-fPIC -pie",
						"--with-ld-opt=-fPIC -pie -z now",
						"--with-compat",
						"--with-mail=dynamic",
						"--with-mail_ssl_module",
						"--with-stream=dynamic",
						"--with-http_sub_module",
					}
				}
				// Use slices.Concat to avoid appending to the package-level slice's backing array.
				return slices.Concat(nginxBaseConfigureArgs, variantArgs)
			},

			// InstallEnv sets DESTDIR so --prefix=/ resolves relative to <prefix>/nginx.
			InstallEnv: func(prefix string) map[string]string {
				return map[string]string{"DESTDIR": fmt.Sprintf("%s/nginx", prefix)}
			},

			// AfterInstall removes html/ and conf/ from the nginx install tree.
			AfterInstall: func(_ context.Context, prefix string, r runner.Runner) error {
				nginxInstallDir := fmt.Sprintf("%s/nginx", prefix)
				return removeNginxRuntimeDirs(r, nginxInstallDir)
			},

			// AfterPack strips the top-level directory from the artifact so the
			// archive root is nginx/ (matching the Ruby builder output).
			AfterPack: func(artifactPath string) error {
				return archive.StripTopLevelDir(artifactPath)
			},
		},
	}
}

// removeNginxRuntimeDirs removes html/ and conf/ from nginxDir, then recreates
// an empty conf/. Shared between nginx and openresty post-install cleanup.
func removeNginxRuntimeDirs(run runner.Runner, nginxDir string) error {
	if err := run.Run("rm", "-rf",
		fmt.Sprintf("%s/html", nginxDir),
		fmt.Sprintf("%s/conf", nginxDir),
	); err != nil {
		return fmt.Errorf("removing html/conf: %w", err)
	}
	if err := run.Run("mkdir", "-p", fmt.Sprintf("%s/conf", nginxDir)); err != nil {
		return fmt.Errorf("recreating conf: %w", err)
	}
	return nil
}
