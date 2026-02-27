package recipe

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/compiler"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/portile"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// RRecipe builds R from source and installs devtools + 4 R packages
// (forecast, plumber, Rserve, shiny).  Sub-dependency inputs are read
// from well-known Concourse resource directories alongside the working
// directory.
type RRecipe struct {
	Fetcher fetch.Fetcher
}

func (r *RRecipe) Name() string { return "r" }
func (r *RRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

// rPackage holds a sub-dependency name alongside the dir that contains its
// source/data.json Concourse resource.
type rPackage struct {
	name   string // R package name (passed to devtools::install_version)
	srcDir string // e.g. "source-forecast-latest"
}

// rSubDeps lists the R packages to install, in the same order as the Ruby builder:
// Rserve, forecast, shiny, plumber.
var rSubDeps = []rPackage{
	{name: "Rserve", srcDir: "source-rserve-latest"},
	{name: "forecast", srcDir: "source-forecast-latest"},
	{name: "shiny", srcDir: "source-shiny-latest"},
	{name: "plumber", srcDir: "source-plumber-latest"},
}

func (r *RRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, outData *output.OutData) error {
	a := apt.New(run)

	// Step 1: Install R build dependencies from stack config (gfortran, libpcre, etc.)
	if err := a.Install(ctx, s.AptPackages["r_build"]...); err != nil {
		return fmt.Errorf("r: apt install r_build: %w", err)
	}

	// Step 2: Set up gfortran (stack-driven version).
	gf := compiler.NewGfortran(s.Compilers.Gfortran, a, run)
	if err := gf.Setup(ctx); err != nil {
		return fmt.Errorf("r: gfortran setup: %w", err)
	}

	// Step 3: Download R source.
	srcTarball := fmt.Sprintf("/tmp/R-%s.tar.gz", src.Version)
	if err := r.Fetcher.Download(ctx, src.URL, srcTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("r: downloading source: %w", err)
	}

	// Compute SHA256 of the downloaded tarball (matches Ruby's git_commit_sha field).
	sourceSHA, err := fileSHA256(srcTarball)
	if err != nil {
		return fmt.Errorf("r: computing source sha256: %w", err)
	}

	// Step 4: Build via portile (configure + make + make install).
	installPrefix := fmt.Sprintf("/usr/local")
	pt := &portile.Portile{
		Name:    "R",
		Version: src.Version,
		URL:     src.URL,
		// We already downloaded to srcTarball — portile fetches independently, so
		// we pass checksum for portile's own download/verify path.
		Checksum: src.PrimaryChecksum(),
		Prefix:   installPrefix,
		Options: []string{
			"--with-readline=no",
			"--with-x=no",
			"--enable-R-shlib",
		},
		Runner:  run,
		Fetcher: r.Fetcher,
	}
	if err := pt.Cook(ctx); err != nil {
		return fmt.Errorf("r: portile cook: %w", err)
	}

	// Step 5: Install devtools (required for install_version below).
	devtoolsCmd := `/usr/local/lib/R/bin/R --vanilla -e 'install.packages("devtools", repos="https://cran.r-project.org")'`
	if err := run.Run("sh", "-c", devtoolsCmd); err != nil {
		return fmt.Errorf("r: installing devtools: %w", err)
	}

	// Step 6: Read sub-dependency source inputs and install R packages.
	// Install order matches Ruby: Rserve, forecast, shiny, plumber.
	subDeps := make(map[string]output.SubDependency)
	for _, pkg := range rSubDeps {
		subInput, err := source.FromFile(fmt.Sprintf("%s/data.json", pkg.srcDir))
		if err != nil {
			return fmt.Errorf("r: reading sub-dep source for %s: %w", pkg.name, err)
		}

		// Format version for Rserve: "1.8.15" → "1.8-15"
		pkgVersion := subInput.Version
		if pkg.name == "Rserve" {
			pkgVersion = formatRserveVersion(subInput.Version)
		}

		// Install via devtools::install_version with dependencies=TRUE and type='source',
		// matching the Ruby builder behaviour.
		rCmd := fmt.Sprintf(
			`/usr/local/lib/R/bin/R --vanilla -e "require('devtools'); devtools::install_version('%s', '%s', repos='https://cran.r-project.org', type='source', dependencies=TRUE)"`,
			pkg.name, pkgVersion,
		)
		if err := run.Run("sh", "-c", rCmd); err != nil {
			return fmt.Errorf("r: installing R package %s: %w", pkg.name, err)
		}

		subDeps[pkg.name] = output.SubDependency{
			Source: &output.SubDepSource{
				URL:    subInput.URL,
				SHA256: subInput.SHA256,
			},
			Version: subInput.Version,
		}
	}

	// Step 7: Remove devtools after use.
	removeDevtoolsCmd := `/usr/local/lib/R/bin/R --vanilla -e 'remove.packages("devtools")'`
	if err := run.Run("sh", "-c", removeDevtoolsCmd); err != nil {
		return fmt.Errorf("r: removing devtools: %w", err)
	}

	// Step 8: Copy gfortran libs into R install.
	// Ruby copies gfortran/f951 into /usr/local/lib/R/bin (R's own bin dir)
	// and gfortran libs into /usr/local/lib/R/lib, so they end up inside the
	// packed artifact (which is tarred from /usr/local/lib/R).
	rLibDir := "/usr/local/lib/R/lib"
	rBinDir := "/usr/local/lib/R/bin"
	if err := gf.CopyLibs(ctx, rLibDir, rBinDir); err != nil {
		return fmt.Errorf("r: copying gfortran libs: %w", err)
	}

	// Step 9: Create f95 symlink → gfortran (relative, matching Ruby's ln -s ./gfortran ./bin/f95).
	if err := run.RunInDir(rBinDir, "ln", "-s", "./gfortran", "./f95"); err != nil {
		return fmt.Errorf("r: creating f95 symlink: %w", err)
	}

	// Step 10: Pack the R install from /usr/local/lib/R.
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("r-%s.tgz", src.Version))
	if err := run.RunInDir("/usr/local/lib/R", "tar", "zcvf", artifactPath, "."); err != nil {
		return fmt.Errorf("r: packing artifact: %w", err)
	}

	// Populate sub-dependencies and git_commit_sha in outData.
	outData.SubDependencies = subDeps
	outData.GitCommitSHA = sourceSHA

	return nil
}

// fileSHA256 returns the hex-encoded SHA256 digest of the file at path.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// formatRserveVersion converts "1.8.14" → "1.8-14"
// (first two dot-separated parts joined by '.', remainder joined by '-').
func formatRserveVersion(v string) string {
	parts := strings.Split(v, ".")
	if len(parts) <= 2 {
		return v
	}
	prefix := strings.Join(parts[:2], ".")
	suffix := strings.Join(parts[2:], ".")
	return fmt.Sprintf("%s-%s", prefix, suffix)
}
