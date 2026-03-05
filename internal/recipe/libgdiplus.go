package recipe

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/autoconf"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// LibgdiplusRecipe builds libgdiplus from source via git clone + autogen + make.
type LibgdiplusRecipe struct{}

func (l *LibgdiplusRecipe) Name() string { return "libgdiplus" }
func (l *LibgdiplusRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (l *LibgdiplusRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return newLibgdiplusAutoconf().Build(ctx, s, src, run, out)
}

// newLibgdiplusAutoconf constructs the AutoconfRecipe for libgdiplus.
func newLibgdiplusAutoconf() *autoconf.Recipe {
	return &autoconf.Recipe{
		DepName: "libgdiplus",
		// No Fetcher needed: SourceProvider uses git clone.
		Hooks: autoconf.Hooks{
			AptPackages: func(s *stack.Stack) []string {
				return s.AptPackages["libgdiplus_build"]
			},

			// SourceProvider clones the repository at the given version tag.
			SourceProvider: func(ctx context.Context, src *source.Input, _ fetch.Fetcher, r runner.Runner) (string, error) {
				version := src.Version
				cloneDir := fmt.Sprintf("libgdiplus-%s", version)
				repoURL := fmt.Sprintf("https://github.com/%s", src.Repo)
				if err := r.Run("git", "clone", "--single-branch", "--branch", version, repoURL, cloneDir); err != nil {
					return "", fmt.Errorf("git clone: %w", err)
				}
				// Return an absolute path so RunInDir works correctly from any CWD.
				absCloneDir := filepath.Join(mustCwd(), cloneDir)
				return absCloneDir, nil
			},

			// AfterExtract runs autogen.sh (which generates and runs ./configure for libgdiplus).
			// AutoconfRecipe will also call ./configure after this hook; that second call is
			// a harmless reconfigure since autogen.sh already generated the script.
			AfterExtract: func(ctx context.Context, srcDir, prefix string, r runner.Runner) error {
				buildEnv := map[string]string{
					"CFLAGS":   "-g -Wno-maybe-uninitialized",
					"CXXFLAGS": "-g -Wno-maybe-uninitialized",
				}
				return r.RunWithEnv(buildEnv, "sh", "-c",
					fmt.Sprintf("cd %s && ./autogen.sh --prefix=%s", srcDir, prefix))
			},

			// ConfigureEnv sets warning-suppression flags for configure, make, and make install.
			ConfigureEnv: func() map[string]string {
				return map[string]string{
					"CFLAGS":   "-g -Wno-maybe-uninitialized",
					"CXXFLAGS": "-g -Wno-maybe-uninitialized",
				}
			},

			PackDirs: func() []string { return []string{"lib"} },
		},
	}
}
