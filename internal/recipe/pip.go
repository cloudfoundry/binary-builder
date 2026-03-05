package recipe

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// PipRecipe builds pip: pip3 download + bundle setuptools + wheel.
type PipRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PipRecipe) Name() string { return "pip" }
func (p *PipRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (p *PipRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	// Setup python and pip.
	if err := setupPythonAndPip(ctx, s, r); err != nil {
		return fmt.Errorf("pip: setup python: %w", err)
	}

	outputPath := fmt.Sprintf("/tmp/pip-%s.tgz", src.Version)

	// Create temp dir, download pip and dependencies.
	tmpDir := fmt.Sprintf("/tmp/pip-build-%s", src.Version)
	if err := r.Run("mkdir", "-p", tmpDir); err != nil {
		return err
	}

	// pip3 download pip itself (gets the pip .tar.gz into tmpDir).
	if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", "--no-binary", ":all:", fmt.Sprintf("pip==%s", src.Version)); err != nil {
		return fmt.Errorf("pip3 download pip: %w", err)
	}

	// Download pip source tarball with checksum verification into tmpDir.
	pipSrcTar := fmt.Sprintf("%s/pip-%s.tar.gz", tmpDir, src.Version)
	if err := p.Fetcher.Download(ctx, src.URL, pipSrcTar, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("downloading pip source: %w", err)
	}

	// Strip the top-level dir from the source tarball (pip-24.0/ → ./),
	// then extract it into tmpDir — matching Ruby's Archive.strip_top_level_directory_from_tar
	// + tar zxf step which adds the pip source tree to the bundle.
	if err := archive.StripTopLevelDir(pipSrcTar); err != nil {
		return fmt.Errorf("stripping top-level dir from pip source: %w", err)
	}
	if err := r.RunInDir(tmpDir, "tar", "zxf", fmt.Sprintf("pip-%s.tar.gz", src.Version)); err != nil {
		return fmt.Errorf("extracting pip source: %w", err)
	}

	// Download setuptools.
	if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", "--no-binary", ":all:", "setuptools"); err != nil {
		return fmt.Errorf("pip3 download setuptools: %w", err)
	}

	// Download wheel with CVE-2026-24049 pin.
	if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", "--no-binary", ":all:", "wheel>=0.46.2"); err != nil {
		return fmt.Errorf("pip3 download wheel: %w", err)
	}

	// Bundle everything.
	if err := r.RunInDir(tmpDir, "tar", "zcvf", outputPath, "."); err != nil {
		return fmt.Errorf("creating pip tarball: %w", err)
	}

	return nil
}

// PipenvRecipe builds pipenv: pip3 download + bundle 7 dependencies.
type PipenvRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PipenvRecipe) Name() string { return "pipenv" }
func (p *PipenvRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (p *PipenvRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	if err := setupPythonAndPip(ctx, s, r); err != nil {
		return fmt.Errorf("pipenv: setup python: %w", err)
	}

	outputPath := fmt.Sprintf("/tmp/pipenv-v%s.tgz", src.Version)

	tmpDir := fmt.Sprintf("/tmp/pipenv-build-%s", src.Version)
	if err := r.Run("mkdir", "-p", tmpDir); err != nil {
		return err
	}

	// Download pipenv.
	if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", "--no-cache-dir", "--no-binary", ":all:", fmt.Sprintf("pipenv==%s", src.Version)); err != nil {
		return fmt.Errorf("pip3 download pipenv: %w", err)
	}

	// Download pipenv source with checksum.
	pipenvTar := fmt.Sprintf("pipenv-%s.tar.gz", src.Version)
	if err := p.Fetcher.Download(ctx, src.URL, fmt.Sprintf("%s/%s", tmpDir, pipenvTar), src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("downloading pipenv source: %w", err)
	}

	// Bundle 7 additional dependencies.
	bundledDeps := []string{
		"pytest-runner",
		"setuptools_scm",
		"parver",
		"wheel>=0.46.2", // CVE-2026-24049
		"invoke",
		"flit_core",
		"hatch-vcs",
	}

	for _, dep := range bundledDeps {
		if err := r.RunInDir(tmpDir, "/usr/bin/pip3", "download", "--no-binary", ":all:", dep); err != nil {
			return fmt.Errorf("pip3 download %s: %w", dep, err)
		}
	}

	// Bundle everything.
	if err := r.RunInDir(tmpDir, "tar", "zcvf", outputPath, "."); err != nil {
		return fmt.Errorf("creating pipenv tarball: %w", err)
	}

	return nil
}

// setupPythonAndPip installs the Python interpreter and pip via apt, then
// upgrades pip and setuptools. The packages to install are read from
// s.AptPackages["pip_build"] (stacks/*.yaml) so they can be adjusted per
// stack without modifying Go source.
func setupPythonAndPip(ctx context.Context, s *stack.Stack, r runner.Runner) error {
	if err := r.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", "update",
	); err != nil {
		return err
	}
	installArgs := append([]string{"install", "-y"}, s.AptPackages["pip_build"]...)
	if err := r.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", installArgs...,
	); err != nil {
		return err
	}
	return r.Run("pip3", "install", "--upgrade", "pip", "setuptools")
}
