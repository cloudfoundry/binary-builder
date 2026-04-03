package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// HWCRecipe cross-compiles the Hostable Web Core (HWC) for Windows.
// The cross-compiler apt packages (e.g. mingw-w64) are read from
// s.AptPackages["hwc_build"] so they can be overridden per stack in
// stacks/*.yaml without modifying Go source.
//
// Ruby recipe (hwc.rb) builds two Windows binaries:
//   - hwc-windows-amd64  (via release-binaries.bash amd64)  → /tmp/hwc.exe
//   - hwc-windows-386    (via release-binaries.bash 386)    → /tmp/hwc_x86.exe
//
// Both are zipped into the artifact named hwc_{version}_windows_x86-64_any-stack.zip.
// ArchiveRecipe uses zip because archive_filename ends in .zip.
type HWCRecipe struct {
	Fetcher fetch.Fetcher
}

func (h *HWCRecipe) Name() string { return "hwc" }
func (h *HWCRecipe) Artifact() ArtifactMeta {
	// Windows binary — arch is x86-64, stack is any-stack, extension is .zip.
	return ArtifactMeta{OS: "windows", Arch: "x86-64", Stack: "any-stack"}
}

func (h *HWCRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	version := src.Version
	tmpPath := "/tmp/src/code.cloudfoundry.org"
	srcDir := fmt.Sprintf("%s/hwc", tmpPath)
	srcTarball := fmt.Sprintf("/tmp/hwc-%s.tar.gz", version)
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("hwc-%s-windows-x86-64.zip", version))

	// Install cross-compiler packages (e.g. mingw-w64). The package list lives in
	// stacks/*.yaml under apt_packages.hwc_build so it can be adjusted per stack
	// without touching Go source.
	if err := apt.New(run).Install(ctx, s.AptPackages["hwc_build"]...); err != nil {
		return fmt.Errorf("hwc: installing hwc_build packages: %w", err)
	}

	// Download source.
	if err := h.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("hwc: downloading source: %w", err)
	}

	// Extract source.
	if err := run.Run("mkdir", "-p", tmpPath); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", srcTarball, "-C", tmpPath); err != nil {
		return fmt.Errorf("hwc: extracting source: %w", err)
	}
	// Rename hwc-VERSION → hwc.
	if err := run.Run("sh", "-c",
		fmt.Sprintf("mv %s/hwc-* %s", tmpPath, srcDir)); err != nil {
		return fmt.Errorf("hwc: renaming source dir: %w", err)
	}

	// Cross-compile for Windows amd64 using mingw-w64.
	// Must run from inside srcDir (where go.mod lives).
	// Matches release-binaries.bash: CGO_ENABLED=1 GO_EXTLINK_ENABLED=1 CC=... GOARCH=... GOOS=windows go build -o OUTPUT -ldflags "-X main.version=VERSION"
	hwcExePath := fmt.Sprintf("%s/hwc-windows-amd64", srcDir)
	if err := run.RunInDirWithEnv(srcDir,
		map[string]string{
			"GOOS":               "windows",
			"GOARCH":             "amd64",
			"CGO_ENABLED":        "1",
			"GO_EXTLINK_ENABLED": "1",
			"CC":                 "x86_64-w64-mingw32-gcc",
			"CXX":                "x86_64-w64-mingw32-g++",
		},
		"go", "build", "-o", hwcExePath, "-ldflags", fmt.Sprintf("-X main.version=%s", version),
	); err != nil {
		return fmt.Errorf("hwc: go build amd64: %w", err)
	}

	// Cross-compile for Windows 386 using mingw-w64.
	hwcX86ExePath := fmt.Sprintf("%s/hwc-windows-386", srcDir)
	if err := run.RunInDirWithEnv(srcDir,
		map[string]string{
			"GOOS":               "windows",
			"GOARCH":             "386",
			"CGO_ENABLED":        "1",
			"GO_EXTLINK_ENABLED": "1",
			"CC":                 "i686-w64-mingw32-gcc",
			"CXX":                "i686-w64-mingw32-g++",
		},
		"go", "build", "-o", hwcX86ExePath, "-ldflags", fmt.Sprintf("-X main.version=%s", version),
	); err != nil {
		return fmt.Errorf("hwc: go build 386: %w", err)
	}

	// Move binaries to /tmp, renamed to hwc.exe and hwc_x86.exe.
	if err := run.Run("mv", hwcExePath, "/tmp/hwc.exe"); err != nil {
		return fmt.Errorf("hwc: moving amd64 binary: %w", err)
	}
	if err := run.Run("mv", hwcX86ExePath, "/tmp/hwc_x86.exe"); err != nil {
		return fmt.Errorf("hwc: moving 386 binary: %w", err)
	}

	// Write sources.yml to /tmp — matches Ruby's YAMLPresenter which writes
	// sources.yml into the tmpdir before zipping (zip -r . includes it).
	srcSHA256, err := fileSHA256(srcTarball)
	if err != nil {
		return fmt.Errorf("hwc: computing source sha256: %w", err)
	}
	sourcesContent := buildSourcesYAML([]SourceEntry{{URL: src.URL, SHA256: srcSHA256}})
	if err := os.WriteFile("/tmp/sources.yml", sourcesContent, 0644); err != nil {
		return fmt.Errorf("hwc: writing sources.yml: %w", err)
	}

	// Pack as zip.
	// Ruby: zip archive.zip -r . from inside tmpdir containing hwc.exe, hwc_x86.exe, sources.yml.
	if err := run.RunInDir("/tmp", "zip", artifactPath, "hwc.exe", "hwc_x86.exe", "sources.yml"); err != nil {
		return fmt.Errorf("hwc: creating zip: %w", err)
	}

	return nil
}
