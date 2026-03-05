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

// GoToolRecipe implements the common pattern for building Go CLI tools (dep, glide, godep):
//
//  1. Download the source tarball to /tmp/<name>-<version>.tar.gz
//  2. mkdir -p <tmpPath>
//  3. tar xzf to <tmpPath>
//  4. Rename <tmpPath>/<name>-* → <tmpPath>/<name>
//  5. Run BuildCmd (sh -c "cd <srcDir> && GOPATH=... go get/build ...")
//  6. Move binary + license to /tmp
//  7. [optional] extra staging step (e.g. glide's mkdir /tmp/bin + copy)
//  8. tar czf artifact from PackFiles
type GoToolRecipe struct {
	ToolName    string // "dep", "glide", "godep"
	OrgPath     string // GitHub org path, e.g. "github.com/golang"
	LicenseName string // "LICENSE" or "License" (godep uses capital-L, no-E)
	// BuildCmd returns the shell command string executed via sh -c inside srcDir.
	BuildCmd func(srcDir, version string) string
	// PackFiles returns the paths to pack into the artifact tarball, relative to /tmp.
	PackFiles func(name string) []string
	// ExtraStaging runs after move and before packing; used by glide to stage /tmp/bin.
	// May be nil.
	ExtraStaging func(ctx context.Context, name string, run runner.Runner) error
	Fetcher      fetch.Fetcher
}

func (g *GoToolRecipe) Name() string { return g.ToolName }
func (g *GoToolRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (g *GoToolRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	name := g.ToolName
	version := src.Version
	tmpPath := fmt.Sprintf("/tmp/src/%s", g.OrgPath)
	srcDir := fmt.Sprintf("%s/%s", tmpPath, name)
	srcTarball := fmt.Sprintf("/tmp/%s-%s.tar.gz", name, version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("%s-v%s-linux-x64.tgz", name, version))

	if err := g.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("%s: downloading source: %w", name, err)
	}
	if err := run.Run("mkdir", "-p", tmpPath); err != nil {
		return fmt.Errorf("%s: mkdir: %w", name, err)
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", tmpPath); err != nil {
		return fmt.Errorf("%s: extracting source: %w", name, err)
	}
	if err := run.Run("sh", "-c",
		fmt.Sprintf("mv %s/%s-* %s", tmpPath, name, srcDir)); err != nil {
		return fmt.Errorf("%s: renaming source dir: %w", name, err)
	}
	if err := run.Run("sh", "-c", g.BuildCmd(srcDir, version)); err != nil {
		return fmt.Errorf("%s: build: %w", name, err)
	}

	// Move binary to /tmp/<name>.
	if err := run.Run("mv", fmt.Sprintf("%s/%s", srcDir, name), fmt.Sprintf("/tmp/%s", name)); err != nil {
		return fmt.Errorf("%s: moving binary: %w", name, err)
	}
	// Move license file to /tmp/<license>.
	if err := run.Run("mv", fmt.Sprintf("%s/%s", srcDir, g.LicenseName), fmt.Sprintf("/tmp/%s", g.LicenseName)); err != nil {
		return fmt.Errorf("%s: moving license: %w", name, err)
	}

	if g.ExtraStaging != nil {
		if err := g.ExtraStaging(ctx, name, run); err != nil {
			return fmt.Errorf("%s: extra staging: %w", name, err)
		}
	}

	packArgs := append([]string{"czf", artifactPath}, g.PackFiles(name)...)
	if err := run.RunInDir("/tmp", "tar", packArgs...); err != nil {
		return fmt.Errorf("%s: packing artifact: %w", name, err)
	}
	return nil
}

// DepRecipe builds the `dep` Go dependency manager tool.
//
// Ruby layout (dep.rb):
//
//	tmp_path  = /tmp/src/github.com/golang
//	source extracted to {tmp_path}/dep-VERSION/, renamed to {tmp_path}/dep/
//	GOPATH = {tmp_path}/dep/deps/_workspace:/tmp
//	`go get -asmflags -trimpath ./...` run inside {tmp_path}/dep/
//	binary lands at /tmp/bin/dep  (second GOPATH entry /tmp → /tmp/bin/)
//	archive_files  = ['/tmp/bin/dep', '/tmp/LICENSE']
//	archive_path_name = 'bin'  → artifact contains bin/dep + bin/LICENSE
type DepRecipe struct {
	Fetcher fetch.Fetcher
}

func (d *DepRecipe) Name() string { return "dep" }
func (d *DepRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (d *DepRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return (&GoToolRecipe{
		ToolName:    "dep",
		OrgPath:     "github.com/golang",
		LicenseName: "LICENSE",
		BuildCmd: func(srcDir, _ string) string {
			gopath := fmt.Sprintf("%s/deps/_workspace:/tmp", srcDir)
			return fmt.Sprintf("cd %s && GOPATH=%s /usr/local/go/bin/go get -asmflags -trimpath ./...", srcDir, gopath)
		},
		// dep: binary lands at /tmp/bin/dep via GOPATH; pack directly from /tmp.
		PackFiles: func(name string) []string {
			return []string{"bin/dep", "bin/LICENSE"}
		},
		Fetcher: d.Fetcher,
	}).Build(ctx, s, src, run, out)
}

// GlideRecipe builds the `glide` Go package manager tool.
//
// Ruby layout (glide.rb):
//
//	tmp_path  = /tmp/src/github.com/Masterminds
//	source extracted to {tmp_path}/glide-VERSION/, renamed to {tmp_path}/glide/
//	GOPATH = /tmp
//	`go build` run inside {tmp_path}/glide/
//	binary built at {tmp_path}/glide/glide, moved to /tmp/glide
//	LICENSE moved to /tmp/LICENSE
//	archive_files  = ['/tmp/glide', '/tmp/LICENSE']
//	archive_path_name = 'bin'  → artifact contains bin/glide + bin/LICENSE
type GlideRecipe struct {
	Fetcher fetch.Fetcher
}

func (g *GlideRecipe) Name() string { return "glide" }
func (g *GlideRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (g *GlideRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return (&GoToolRecipe{
		ToolName:    "glide",
		OrgPath:     "github.com/Masterminds",
		LicenseName: "LICENSE",
		BuildCmd: func(srcDir, _ string) string {
			return fmt.Sprintf("cd %s && GOPATH=/tmp /usr/local/go/bin/go build", srcDir)
		},
		PackFiles: func(name string) []string {
			return []string{"bin/glide", "bin/LICENSE"}
		},
		// glide: binary built in srcDir, not GOPATH bin; must stage manually.
		ExtraStaging: func(_ context.Context, name string, run runner.Runner) error {
			if err := run.Run("mkdir", "-p", "/tmp/bin"); err != nil {
				return err
			}
			if err := run.Run("cp", fmt.Sprintf("/tmp/%s", name), fmt.Sprintf("/tmp/bin/%s", name)); err != nil {
				return err
			}
			if err := run.Run("cp", "/tmp/LICENSE", "/tmp/bin/LICENSE"); err != nil {
				return err
			}
			return nil
		},
		Fetcher: g.Fetcher,
	}).Build(ctx, s, src, run, out)
}

// GodepRecipe builds the `godep` Go vendoring tool.
//
// Ruby layout (godep.rb):
//
//	tmp_path  = /tmp/src/github.com/tools
//	source extracted to {tmp_path}/godep-VERSION/, renamed to {tmp_path}/godep/
//	GOPATH = {tmp_path}/godep/Godeps/_workspace:/tmp
//	`go get ./...` run inside {tmp_path}/godep/
//	binary lands at /tmp/bin/godep
//	License (capital L, no E) moved to /tmp/License
//	archive_files  = ['/tmp/bin/godep', '/tmp/License']
//	archive_path_name = 'bin'  → artifact contains bin/godep + bin/License
type GodepRecipe struct {
	Fetcher fetch.Fetcher
}

func (g *GodepRecipe) Name() string { return "godep" }
func (g *GodepRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (g *GodepRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, out *output.OutData) error {
	return (&GoToolRecipe{
		ToolName:    "godep",
		OrgPath:     "github.com/tools",
		LicenseName: "License",
		BuildCmd: func(srcDir, _ string) string {
			gopath := fmt.Sprintf("%s/Godeps/_workspace:/tmp", srcDir)
			return fmt.Sprintf("cd %s && GOPATH=%s /usr/local/go/bin/go get ./...", srcDir, gopath)
		},
		// godep: binary lands at /tmp/bin/godep via GOPATH; pack directly from /tmp.
		PackFiles: func(_ string) []string {
			return []string{"bin/godep", "bin/License"}
		},
		Fetcher: g.Fetcher,
	}).Build(ctx, s, src, run, out)
}
