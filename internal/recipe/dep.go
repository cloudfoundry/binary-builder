package recipe

import (
	"path/filepath"
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

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

func (d *DepRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version
	tmpPath := "/tmp/src/github.com/golang"
	srcDir := fmt.Sprintf("%s/dep", tmpPath)
	srcTarball := fmt.Sprintf("/tmp/dep-%s.tar.gz", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("dep-v%s-linux-x64.tgz", version))

	if err := d.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("dep: downloading source: %w", err)
	}

	if err := run.Run("mkdir", "-p", tmpPath); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", tmpPath); err != nil {
		return fmt.Errorf("dep: extracting source: %w", err)
	}
	// Rename dep-VERSION → dep (matching Ruby's FileUtils.mv(Dir.glob("dep-*").first, "dep")).
	if err := run.Run("sh", "-c",
		fmt.Sprintf("mv %s/dep-* %s", tmpPath, srcDir)); err != nil {
		return fmt.Errorf("dep: renaming source dir: %w", err)
	}

	// go get with GOPATH workspace, run inside srcDir.
	// Binary lands at /tmp/bin/dep (GOPATH second entry /tmp, bin subdir).
	gopath := fmt.Sprintf("%s/deps/_workspace:/tmp", srcDir)
	if err := run.Run("sh", "-c",
		fmt.Sprintf("cd %s && GOPATH=%s /usr/local/go/bin/go get -asmflags -trimpath ./...", srcDir, gopath)); err != nil {
		return fmt.Errorf("dep: go get: %w", err)
	}

	// Move LICENSE to /tmp/LICENSE.
	if err := run.Run("mv", fmt.Sprintf("%s/LICENSE", srcDir), "/tmp/LICENSE"); err != nil {
		return fmt.Errorf("dep: moving LICENSE: %w", err)
	}

	// Pack: bin/dep + bin/LICENSE inside the tgz.
	// /tmp/bin/dep already exists from go get; /tmp/LICENSE was just moved there.
	if err := run.RunInDir("/tmp", "tar", "czf", artifactPath,
		"bin/dep", "bin/LICENSE"); err != nil {
		return fmt.Errorf("dep: packing artifact: %w", err)
	}

	return nil
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

func (g *GlideRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version
	tmpPath := "/tmp/src/github.com/Masterminds"
	srcDir := fmt.Sprintf("%s/glide", tmpPath)
	srcTarball := fmt.Sprintf("/tmp/glide-%s.tar.gz", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("glide-v%s-linux-x64.tgz", version))

	if err := g.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("glide: downloading source: %w", err)
	}

	if err := run.Run("mkdir", "-p", tmpPath); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", tmpPath); err != nil {
		return fmt.Errorf("glide: extracting source: %w", err)
	}
	// Rename glide-VERSION → glide.
	if err := run.Run("sh", "-c",
		fmt.Sprintf("mv %s/glide-* %s", tmpPath, srcDir)); err != nil {
		return fmt.Errorf("glide: renaming source dir: %w", err)
	}

	// go build with GOPATH=/tmp, run inside srcDir.
	if err := run.Run("sh", "-c",
		fmt.Sprintf("cd %s && GOPATH=/tmp /usr/local/go/bin/go build", srcDir)); err != nil {
		return fmt.Errorf("glide: go build: %w", err)
	}

	// Move binary and LICENSE to /tmp.
	if err := run.Run("mv", fmt.Sprintf("%s/glide", srcDir), "/tmp/glide"); err != nil {
		return fmt.Errorf("glide: moving binary: %w", err)
	}
	if err := run.Run("mv", fmt.Sprintf("%s/LICENSE", srcDir), "/tmp/LICENSE"); err != nil {
		return fmt.Errorf("glide: moving LICENSE: %w", err)
	}

	// Pack: bin/glide + bin/LICENSE inside the tgz.
	// ArchiveRecipe copies archive_files into {tmpdir}/bin/ then tars.
	// We replicate: mkdir /tmp/bin, copy, tar.
	if err := run.Run("mkdir", "-p", "/tmp/bin"); err != nil {
		return err
	}
	if err := run.Run("cp", "/tmp/glide", "/tmp/bin/glide"); err != nil {
		return fmt.Errorf("glide: copying binary: %w", err)
	}
	if err := run.Run("cp", "/tmp/LICENSE", "/tmp/bin/LICENSE"); err != nil {
		return fmt.Errorf("glide: copying LICENSE: %w", err)
	}
	if err := run.RunInDir("/tmp", "tar", "czf", artifactPath,
		"bin/glide", "bin/LICENSE"); err != nil {
		return fmt.Errorf("glide: packing artifact: %w", err)
	}

	return nil
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

func (g *GodepRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version
	tmpPath := "/tmp/src/github.com/tools"
	srcDir := fmt.Sprintf("%s/godep", tmpPath)
	srcTarball := fmt.Sprintf("/tmp/godep-%s.tar.gz", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("godep-v%s-linux-x64.tgz", version))

	if err := g.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("godep: downloading source: %w", err)
	}

	if err := run.Run("mkdir", "-p", tmpPath); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", tmpPath); err != nil {
		return fmt.Errorf("godep: extracting source: %w", err)
	}
	// Rename godep-VERSION → godep.
	if err := run.Run("sh", "-c",
		fmt.Sprintf("mv %s/godep-* %s", tmpPath, srcDir)); err != nil {
		return fmt.Errorf("godep: renaming source dir: %w", err)
	}

	// go get with GOPATH workspace, run inside srcDir.
	// Binary lands at /tmp/bin/godep.
	gopath := fmt.Sprintf("%s/Godeps/_workspace:/tmp", srcDir)
	if err := run.Run("sh", "-c",
		fmt.Sprintf("cd %s && GOPATH=%s /usr/local/go/bin/go get ./...", srcDir, gopath)); err != nil {
		return fmt.Errorf("godep: go get: %w", err)
	}

	// Move License (capital L, no E — matches Ruby source) to /tmp/License.
	if err := run.Run("mv", fmt.Sprintf("%s/License", srcDir), "/tmp/License"); err != nil {
		return fmt.Errorf("godep: moving License: %w", err)
	}

	// Pack: bin/godep + bin/License inside the tgz.
	// /tmp/bin/godep already exists from go get.
	if err := run.RunInDir("/tmp", "tar", "czf", artifactPath,
		"bin/godep", "bin/License"); err != nil {
		return fmt.Errorf("godep: packing artifact: %w", err)
	}

	return nil
}
