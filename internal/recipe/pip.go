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

func (p *PipRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&BundleRecipe{
		DepName: "pip",
		Meta:    ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher: p.Fetcher,
		MainPackage: func(version string) string {
			return fmt.Sprintf("pip==%s", version)
		},
		DownloadArgs: []string{"--no-binary", ":all:"},
		// pip's extra steps: download source tarball, strip top-level dir, then extract.
		ExtraSteps: func(ctx context.Context, tmpDir string, src *source.Input, f fetch.Fetcher, r runner.Runner) error {
			pipSrcTar := fmt.Sprintf("%s/pip-%s.tar.gz", tmpDir, src.Version)
			if err := f.Download(ctx, src.URL, pipSrcTar, src.PrimaryChecksum()); err != nil {
				return fmt.Errorf("downloading pip source: %w", err)
			}
			if err := archive.StripTopLevelDir(pipSrcTar); err != nil {
				return fmt.Errorf("stripping top-level dir from pip source: %w", err)
			}
			if err := r.RunInDir(tmpDir, "tar", "zxf", fmt.Sprintf("pip-%s.tar.gz", src.Version)); err != nil {
				return fmt.Errorf("extracting pip source: %w", err)
			}
			return nil
		},
		ExtraDeps: []string{
			"setuptools",
			"wheel>=0.46.2", // CVE-2026-24049
		},
	}).Build(ctx, s, src, r, out)
}

// PipenvRecipe builds pipenv: pip3 download + bundle 7 dependencies.
type PipenvRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PipenvRecipe) Name() string { return "pipenv" }
func (p *PipenvRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (p *PipenvRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, out *output.OutData) error {
	return (&BundleRecipe{
		DepName: "pipenv",
		Meta:    ArtifactMeta{OS: "linux", Arch: "noarch"},
		Fetcher: p.Fetcher,
		MainPackage: func(version string) string {
			return fmt.Sprintf("pipenv==%s", version)
		},
		// Do NOT pass --no-binary :all: for the main pipenv package: recent pipenv
		// sdists declare name="unknown" in their metadata, which pip 22.x (cflinuxfs4)
		// rejects. The source tarball is fetched explicitly in ExtraSteps anyway.
		DownloadArgs: []string{"--no-cache-dir"},
		// pipenv: also download the source tarball into tmpDir for bundling.
		ExtraSteps: func(ctx context.Context, tmpDir string, src *source.Input, f fetch.Fetcher, r runner.Runner) error {
			pipenvTar := fmt.Sprintf("pipenv-%s.tar.gz", src.Version)
			return f.Download(ctx, src.URL, fmt.Sprintf("%s/%s", tmpDir, pipenvTar), src.PrimaryChecksum())
		},
		ExtraDeps: []string{
			"pytest-runner",
			"setuptools_scm",
			"parver",
			"wheel>=0.46.2", // CVE-2026-24049
			"invoke",
			"flit_core",
			"hatch-vcs",
		},
		// pipenv output path has a 'v' prefix: /tmp/pipenv-v{version}.tgz
		OutputPath: func(version string) string {
			return fmt.Sprintf("/tmp/pipenv-v%s.tgz", version)
		},
	}).Build(ctx, s, src, r, out)
}

// setupPythonAndPip installs the Python interpreter and pip via apt.
// The packages to install are read from s.AptPackages["pip_build"]
// (stacks/*.yaml) so they can be adjusted per stack without modifying Go source.
// We rely on the apt-installed versions of pip and setuptools directly — attempting
// to upgrade them via pip3 fails on Ubuntu 24.04 (PEP 668 / no RECORD file).
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
	return nil
}
