package compiler_test

import (
	"context"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/compiler"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCCSetupCflinuxfs4(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GCCConfig{
		Version:      12,
		Packages:     []string{"gcc-12", "g++-12"},
		PPA:          "ppa:ubuntu-toolchain-r/test",
		ToolPackages: []string{"software-properties-common"},
	}

	gcc := compiler.NewGCC(config, a, f)
	err := gcc.Setup(context.Background())
	require.NoError(t, err)

	// Expect: update+install software-properties-common (from ToolPackages), add-apt-repository,
	//         apt-get update (from AddPPA), update+install gcc-12 g++-12, update-alternatives
	var callNames []string
	for _, c := range f.Calls {
		callNames = append(callNames, c.Name)
	}

	// software-properties-common must be installed before add-apt-repository is called.
	var toolInstallIdx, addPPAIdx = -1, -1
	for i, c := range f.Calls {
		if c.Name == "add-apt-repository" && addPPAIdx < 0 {
			addPPAIdx = i
		}
		if c.Name == "apt-get" && toolInstallIdx < 0 {
			for _, arg := range c.Args {
				if arg == "software-properties-common" {
					toolInstallIdx = i
				}
			}
		}
	}
	require.True(t, toolInstallIdx >= 0, "software-properties-common not installed")
	require.True(t, addPPAIdx >= 0, "add-apt-repository not called")
	assert.Less(t, toolInstallIdx, addPPAIdx, "tool packages must be installed before add-apt-repository")

	// Should see add-apt-repository (PPA is non-empty).
	assert.Contains(t, callNames, "add-apt-repository")

	// Should see update-alternatives.
	assert.Contains(t, callNames, "update-alternatives")

	// Find the update-alternatives call and verify version.
	for _, c := range f.Calls {
		if c.Name == "update-alternatives" {
			assert.Contains(t, c.Args, "/usr/bin/gcc-12")
			assert.Contains(t, c.Args, "/usr/bin/g++-12")
			break
		}
	}

	// Find the GCC install call.
	for _, c := range f.Calls {
		if c.Name == "apt-get" {
			for _, arg := range c.Args {
				if arg == "gcc-12" {
					assert.Contains(t, c.Args, "g++-12")
					break
				}
			}
		}
	}
}

func TestGCCSetupCflinuxfs5(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GCCConfig{
		Version:      14,
		Packages:     []string{"gcc-14", "g++-14"},
		PPA:          "", // No PPA needed on cflinuxfs5.
		ToolPackages: []string{"software-properties-common"},
	}

	gcc := compiler.NewGCC(config, a, f)
	err := gcc.Setup(context.Background())
	require.NoError(t, err)

	// ToolPackages should be installed before the GCC packages, even when PPA is empty.
	require.NotEmpty(t, f.Calls)
	var toolInstallIdx2, gccInstallIdx = -1, -1
	for i, c := range f.Calls {
		if c.Name == "apt-get" {
			for _, arg := range c.Args {
				if arg == "software-properties-common" && toolInstallIdx2 < 0 {
					toolInstallIdx2 = i
				}
				if arg == "gcc-14" && gccInstallIdx < 0 {
					gccInstallIdx = i
				}
			}
		}
	}
	require.True(t, toolInstallIdx2 >= 0, "software-properties-common not installed")
	require.True(t, gccInstallIdx >= 0, "gcc-14 not installed")
	assert.Less(t, toolInstallIdx2, gccInstallIdx, "tool packages must be installed before gcc packages")

	// Should NOT see add-apt-repository (PPA is empty).
	for _, c := range f.Calls {
		assert.NotEqual(t, "add-apt-repository", c.Name,
			"cflinuxfs5 should not add a PPA")
	}

	// Should see update-alternatives with gcc-14.
	for _, c := range f.Calls {
		if c.Name == "update-alternatives" {
			assert.Contains(t, c.Args, "/usr/bin/gcc-14")
			assert.Contains(t, c.Args, "/usr/bin/g++-14")
			break
		}
	}
}

func TestGCCSetupNoToolPackages(t *testing.T) {
	// When ToolPackages is empty, no tool install call should be made.
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GCCConfig{
		Version:      12,
		Packages:     []string{"gcc-12", "g++-12"},
		PPA:          "",
		ToolPackages: nil, // explicitly empty
	}

	gcc := compiler.NewGCC(config, a, f)
	require.NoError(t, gcc.Setup(context.Background()))

	// No apt-get call should contain "software-properties-common".
	for _, c := range f.Calls {
		if c.Name == "apt-get" {
			assert.NotContains(t, c.Args, "software-properties-common",
				"no tool package install expected when ToolPackages is empty")
		}
	}
}

func TestGfortranSetupCflinuxfs4(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GfortranConfig{
		Version:  11,
		Bin:      "/usr/bin/x86_64-linux-gnu-gfortran-11",
		LibPath:  "/usr/lib/gcc/x86_64-linux-gnu/11",
		Packages: []string{"gfortran", "libgfortran-12-dev"},
	}

	gf := compiler.NewGfortran(config, a, f)
	err := gf.Setup(context.Background())
	require.NoError(t, err)

	// Should install gfortran packages (update + install = 2 calls).
	require.Len(t, f.Calls, 2)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"update"}, f.Calls[0].Args)
	assert.Equal(t, "apt-get", f.Calls[1].Name)
	assert.Contains(t, f.Calls[1].Args, "gfortran")
	assert.Contains(t, f.Calls[1].Args, "libgfortran-12-dev")
}

func TestGfortranSetupCflinuxfs5(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GfortranConfig{
		Version:     13,
		Bin:         "/usr/bin/x86_64-linux-gnu-gfortran-13",
		LibPath:     "/usr/lib/gcc/x86_64-linux-gnu/13",
		LibexecPath: "/usr/libexec/gcc/x86_64-linux-gnu/13",
		Packages:    []string{"gfortran", "libgfortran-13-dev"},
	}

	gf := compiler.NewGfortran(config, a, f)
	err := gf.Setup(context.Background())
	require.NoError(t, err)

	require.Len(t, f.Calls, 2)
	assert.Equal(t, []string{"update"}, f.Calls[0].Args)
	assert.Contains(t, f.Calls[1].Args, "libgfortran-13-dev")
}

func TestGfortranCopyLibsCflinuxfs4(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GfortranConfig{
		Version: 11,
		Bin:     "/usr/bin/x86_64-linux-gnu-gfortran-11",
		LibPath: "/usr/lib/gcc/x86_64-linux-gnu/11",
	}

	gf := compiler.NewGfortran(config, a, f)
	err := gf.CopyLibs(context.Background(), "/target/lib", "/target/bin")
	require.NoError(t, err)

	// Verify copies from version 11 paths.
	var cpSources []string
	for _, c := range f.Calls {
		if c.Name == "cp" {
			cpSources = append(cpSources, c.Args[1]) // -L is args[0], source is args[1]
		}
	}

	assert.Contains(t, cpSources, "/usr/bin/x86_64-linux-gnu-gfortran-11")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/11/f951")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/11/libcaf_single.a")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/11/libgfortran.a")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/11/libgfortran.so")
	assert.Contains(t, cpSources, "/usr/lib/x86_64-linux-gnu/libpcre2-8.so.0")
}

func TestGfortranCopyLibsCflinuxfs5(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GfortranConfig{
		Version:     13,
		Bin:         "/usr/bin/x86_64-linux-gnu-gfortran-13",
		LibPath:     "/usr/lib/gcc/x86_64-linux-gnu/13",
		LibexecPath: "/usr/libexec/gcc/x86_64-linux-gnu/13",
	}

	gf := compiler.NewGfortran(config, a, f)
	err := gf.CopyLibs(context.Background(), "/target/lib", "/target/bin")
	require.NoError(t, err)

	// Verify copies from version 13 paths.
	var cpSources []string
	for _, c := range f.Calls {
		if c.Name == "cp" {
			cpSources = append(cpSources, c.Args[1])
		}
	}

	assert.Contains(t, cpSources, "/usr/bin/x86_64-linux-gnu-gfortran-13")
	// f951 comes from libexec_path on cflinuxfs5 (noble), not lib_path.
	assert.Contains(t, cpSources, "/usr/libexec/gcc/x86_64-linux-gnu/13/f951")
	assert.NotContains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/13/f951")
	// Libs still come from lib_path.
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/13/libcaf_single.a")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/13/libgfortran.a")
	assert.Contains(t, cpSources, "/usr/lib/gcc/x86_64-linux-gnu/13/libgfortran.so")

	// Verify NO version 11 or 14 paths.
	for _, src := range cpSources {
		assert.NotContains(t, src, "/11/")
	}
}

func TestGfortranCopyLibsTargetPaths(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	config := stack.GfortranConfig{
		Version: 11,
		Bin:     "/usr/bin/x86_64-linux-gnu-gfortran-11",
		LibPath: "/usr/lib/gcc/x86_64-linux-gnu/11",
	}

	gf := compiler.NewGfortran(config, a, f)
	err := gf.CopyLibs(context.Background(), "/r/lib", "/r/bin")
	require.NoError(t, err)

	// Verify target paths.
	var cpDests []string
	for _, c := range f.Calls {
		if c.Name == "cp" {
			cpDests = append(cpDests, c.Args[2]) // -L is args[0], source is args[1], dest is args[2]
		}
	}

	assert.Contains(t, cpDests, "/r/bin/gfortran")
	assert.Contains(t, cpDests, "/r/bin/f951")
	assert.Contains(t, cpDests, "/r/lib/libcaf_single.a")
	assert.Contains(t, cpDests, "/r/lib/libgfortran.a")
	assert.Contains(t, cpDests, "/r/lib/libgfortran.so")
	assert.Contains(t, cpDests, "/r/lib/libpcre2-8.so.0")
}
