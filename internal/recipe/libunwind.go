package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/autoconf"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// LibunwindRecipe builds libunwind from source.
// The Concourse github_releases depwatcher only emits metadata in source/data.json;
// it does not pre-download the tarball. We fetch it ourselves via Fetcher.
// Only the include/ and lib/ directories are packed into the artifact.
type LibunwindRecipe struct {
	Fetcher fetch.Fetcher
}

func (l *LibunwindRecipe) Name() string { return "libunwind" }
func (l *LibunwindRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (l *LibunwindRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return newLibunwindAutoconf(l.Fetcher).Build(ctx, s, src, run, out)
}

// newLibunwindAutoconf constructs the AutoconfRecipe for libunwind.
func newLibunwindAutoconf(fetcher fetch.Fetcher) *autoconf.Recipe {
	return &autoconf.Recipe{
		DepName: "libunwind",
		Fetcher: fetcher,
		Hooks: autoconf.Hooks{
			AptPackages: func(s *stack.Stack) []string {
				return s.AptPackages["libunwind_build"]
			},

			// SourceProvider downloads the tarball via Fetcher (the github_releases
			// depwatcher only places data.json in source/, not the tarball itself),
			// extracts it to /tmp, and returns the extracted directory path.
			SourceProvider: func(ctx context.Context, src *source.Input, f fetch.Fetcher, r runner.Runner) (string, error) {
				parts := strings.Split(src.URL, "/")
				filename := parts[len(parts)-1]
				tag := strings.TrimSuffix(strings.TrimSuffix(filename, ".tar.gz"), ".tgz")
				// Two URL styles:
				//   refs/tags/v1.6.2.tar.gz   → tag="v1.6.2"         → extracts to libunwind-1.6.2/
				//   libunwind-1.6.2.tar.gz    → tag="libunwind-1.6.2" → extracts to libunwind-1.6.2/
				var dirName string
				if strings.HasPrefix(tag, "libunwind-") {
					dirName = tag
				} else {
					dirName = "libunwind-" + strings.TrimPrefix(tag, "v")
				}

				srcTarball := filepath.Join(os.TempDir(), filename)
				if err := f.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
					return "", fmt.Errorf("downloading libunwind source: %w", err)
				}
				defer os.Remove(srcTarball)
				if err := r.Run("tar", "xzf", srcTarball, "-C", "/tmp"); err != nil {
					return "", fmt.Errorf("extracting source: %w", err)
				}
				return fmt.Sprintf("/tmp/%s", dirName), nil
			},

			// AfterExtract regenerates ./configure from configure.ac.
			// GitHub source archives only contain autotools sources, not the generated script.
			AfterExtract: func(ctx context.Context, srcDir, _ string, r runner.Runner) error {
				return r.RunInDir(srcDir, "autoreconf", "-i")
			},

			PackDirs: func() []string { return []string{"include", "lib"} },
		},
	}
}
