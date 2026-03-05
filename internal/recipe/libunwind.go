package recipe

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/autoconf"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// LibunwindRecipe builds libunwind from a pre-downloaded source tarball.
// The Concourse github-releases depwatcher has already placed the tarball in source/.
// Only the include/ and lib/ directories are packed into the artifact.
type LibunwindRecipe struct{}

func (l *LibunwindRecipe) Name() string { return "libunwind" }
func (l *LibunwindRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (l *LibunwindRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return newLibunwindAutoconf().Build(ctx, s, src, run, out)
}

// newLibunwindAutoconf constructs the AutoconfRecipe for libunwind.
func newLibunwindAutoconf() *autoconf.Recipe {
	return &autoconf.Recipe{
		DepName: "libunwind",
		// No Fetcher: SourceProvider reads from source/ instead.
		Hooks: autoconf.Hooks{
			AptPackages: func(s *stack.Stack) []string {
				return s.AptPackages["libunwind_build"]
			},

			// SourceProvider reads the pre-downloaded tarball from source/ and
			// extracts it to /tmp, returning the extracted directory path.
			SourceProvider: func(ctx context.Context, src *source.Input, _ fetch.Fetcher, r runner.Runner) (string, error) {
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

				srcTarball := fmt.Sprintf("source/%s", filename)
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
