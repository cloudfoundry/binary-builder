package recipe

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/autoconf"
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

func (o *OpenrestyRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return o.newAutoconf().Build(ctx, s, src, run, out)
}

// newAutoconf constructs the AutoconfRecipe for openresty.
func (o *OpenrestyRecipe) newAutoconf() *autoconf.Recipe {
	return &autoconf.Recipe{
		DepName: "openresty",
		Fetcher: o.Fetcher,
		Hooks: autoconf.Hooks{
			// No apt packages needed for openresty.
			AptPackages: func(_ *stack.Stack) []string { return nil },

			// SourceProvider downloads and extracts the openresty source tarball.
			SourceProvider: func(ctx context.Context, src *source.Input, f fetch.Fetcher, r runner.Runner) (string, error) {
				version := src.Version
				srcTarball := fmt.Sprintf("/tmp/openresty-%s.tar.gz", version)
				if err := f.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
					return "", fmt.Errorf("downloading source: %w", err)
				}
				if err := r.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
					return "", fmt.Errorf("extracting source: %w", err)
				}
				return fmt.Sprintf("/tmp/openresty-%s", version), nil
			},

			// ConfigureArgs returns the full openresty configure flags.
			// Flags match Ruby builder.rb build_openresty exactly.
			ConfigureArgs: func(_, prefix string) []string {
				return []string{
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
			},

			MakeArgs: func() []string { return []string{"-j2"} },

			// AfterInstall removes the bin/openresty symlink and cleans nginx runtime dirs.
			AfterInstall: func(ctx context.Context, prefix string, r runner.Runner) error {
				if err := r.Run("rm", "-rf", fmt.Sprintf("%s/bin/openresty", prefix)); err != nil {
					return fmt.Errorf("removing bin/openresty: %w", err)
				}
				nginxDir := fmt.Sprintf("%s/nginx", prefix)
				return removeNginxRuntimeDirs(r, nginxDir)
			},
		},
	}
}
