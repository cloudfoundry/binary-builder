// Package compiler provides GCC and gfortran setup helpers.
// All version numbers and paths come from the injected stack config —
// no hardcoded compiler versions in this package.
package compiler

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// GCC manages GCC/g++ installation and update-alternatives setup.
type GCC struct {
	Config stack.GCCConfig
	APT    *apt.APT
	Runner runner.Runner
}

// NewGCC creates a GCC instance from stack config.
func NewGCC(config stack.GCCConfig, a *apt.APT, r runner.Runner) *GCC {
	return &GCC{Config: config, APT: a, Runner: r}
}

// Setup installs GCC, optionally adds a PPA, and sets up update-alternatives.
// On cflinuxfs4: adds PPA, installs gcc-12/g++-12.
// On cflinuxfs5: skips PPA (empty string), installs gcc-14/g++-14 (native).
func (g *GCC) Setup(ctx context.Context) error {
	// Install software-properties-common for add-apt-repository.
	if err := g.APT.Install(ctx, "software-properties-common"); err != nil {
		return fmt.Errorf("installing software-properties-common: %w", err)
	}

	// Add PPA only when configured (cflinuxfs4 needs it, cflinuxfs5 does not).
	if err := g.APT.AddPPA(ctx, g.Config.PPA); err != nil {
		return fmt.Errorf("adding GCC PPA: %w", err)
	}

	// Install GCC packages.
	if err := g.APT.Install(ctx, g.Config.Packages...); err != nil {
		return fmt.Errorf("installing GCC packages: %w", err)
	}

	// Set up update-alternatives so gcc/g++ point to the correct version.
	gccBin := fmt.Sprintf("/usr/bin/gcc-%d", g.Config.Version)
	gppBin := fmt.Sprintf("/usr/bin/g++-%d", g.Config.Version)

	return g.Runner.Run(
		"update-alternatives",
		"--install", "/usr/bin/gcc", "gcc", gccBin, "60",
		"--slave", "/usr/bin/g++", "g++", gppBin,
	)
}

// Gfortran manages gfortran installation and library copying.
type Gfortran struct {
	Config stack.GfortranConfig
	APT    *apt.APT
	Runner runner.Runner
}

// NewGfortran creates a Gfortran instance from stack config.
func NewGfortran(config stack.GfortranConfig, a *apt.APT, r runner.Runner) *Gfortran {
	return &Gfortran{Config: config, APT: a, Runner: r}
}

// Setup installs gfortran packages for the stack.
func (g *Gfortran) Setup(ctx context.Context) error {
	return g.APT.Install(ctx, g.Config.Packages...)
}

// CopyLibs copies the stack-specific gfortran libraries into the target directory.
// targetLib receives .a and .so files; targetBin receives the gfortran binary and f951.
//
// On cflinuxfs4: copies from /usr/lib/gcc/x86_64-linux-gnu/11/
// On cflinuxfs5: copies from /usr/lib/gcc/x86_64-linux-gnu/14/
func (g *Gfortran) CopyLibs(_ context.Context, targetLib, targetBin string) error {
	libPath := g.Config.LibPath

	// Copy gfortran binary.
	if err := g.Runner.Run("cp", "-L", g.Config.Bin, fmt.Sprintf("%s/gfortran", targetBin)); err != nil {
		return fmt.Errorf("copying gfortran binary: %w", err)
	}

	// Copy f951 compiler frontend.
	if err := g.Runner.Run("cp", "-L", fmt.Sprintf("%s/f951", libPath), fmt.Sprintf("%s/f951", targetBin)); err != nil {
		return fmt.Errorf("copying f951: %w", err)
	}

	// Copy libraries.
	libs := []string{"libcaf_single.a", "libgfortran.a", "libgfortran.so"}
	for _, lib := range libs {
		src := fmt.Sprintf("%s/%s", libPath, lib)
		dst := fmt.Sprintf("%s/%s", targetLib, lib)
		if err := g.Runner.Run("cp", "-L", src, dst); err != nil {
			return fmt.Errorf("copying %s: %w", lib, err)
		}
	}

	// Copy system libpcre2.
	if err := g.Runner.Run("cp", "-L", "/usr/lib/x86_64-linux-gnu/libpcre2-8.so.0", fmt.Sprintf("%s/libpcre2-8.so.0", targetLib)); err != nil {
		return fmt.Errorf("copying libpcre2: %w", err)
	}

	return nil
}
