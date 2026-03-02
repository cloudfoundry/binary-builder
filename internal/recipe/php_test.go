package recipe_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/recipe"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPHPRecipeName(t *testing.T) {
	r := &recipe.PHPRecipe{}
	assert.Equal(t, "php", r.Name())
}

func TestPHPRecipeArtifact(t *testing.T) {
	r := &recipe.PHPRecipe{}
	meta := r.Artifact()
	assert.Equal(t, "linux", meta.OS)
	assert.Equal(t, "x64", meta.Arch)
}

func TestPHPRecipeBuildInstallsAptPackages(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name: "cflinuxfs4",
		AptPackages: map[string][]string{
			"php_build": {"libssl-dev", "libxml2-dev", "libbz2-dev"},
		},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	assert.True(t, hasCallMatching(fakeRun.Calls, "apt-get", "libssl-dev"), "should apt-get install libssl-dev")
	assert.True(t, hasCallMatching(fakeRun.Calls, "apt-get", "libxml2-dev"), "should apt-get install libxml2-dev")
}

func TestPHPRecipeBuildCreatesSymlinks(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name: "cflinuxfs4",
		AptPackages: map[string][]string{
			"php_build": {},
		},
		PHPSymlinks: []stack.Symlink{
			{Src: "/usr/include/x86_64-linux-gnu/curl", Dst: "/usr/local/include/curl"},
			{Src: "/usr/include/x86_64-linux-gnu/gmp.h", Dst: "/usr/include/gmp.h"},
		},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	assert.True(t, hasCallMatching(fakeRun.Calls, "ln", "/usr/local/include/curl"), "should create curl symlink")
	assert.True(t, hasCallMatching(fakeRun.Calls, "ln", "/usr/include/gmp.h"), "should create gmp.h symlink")
}

func TestPHPRecipeBuildConfigureFlags(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// The configure command is run via bash -c with LIBS=-lz prefix.
	found := false
	for _, c := range fakeRun.Calls {
		if c.Name == "bash" {
			for _, arg := range c.Args {
				if strings.Contains(arg, "LIBS=-lz") && strings.Contains(arg, "--disable-static") {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "should run configure with LIBS=-lz and --disable-static")
}

func TestPHPRecipeBuildPopulatesSubDependencies(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// SubDependencies should be populated from the embedded extension YAML.
	// Check a representative sample of well-known extensions.
	require.NotNil(t, outData.SubDependencies)
	assert.Contains(t, outData.SubDependencies, "apcu", "apcu should be in sub-dependencies")
	assert.Contains(t, outData.SubDependencies, "rabbitmq", "rabbitmq should be in sub-dependencies")
	assert.Greater(t, len(outData.SubDependencies), 20, "should have many sub-dependencies from the embedded YAML")
}

func TestPHPRecipeBuildDownloadsSource(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// Should wget PHP source.
	assert.True(t, hasCallMatching(fakeRun.Calls, "wget", "php-8.3.2.tar.gz"), "should download PHP source")
}

func TestPHPRecipeBuildInvalidVersion(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "invalid"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{}
	err := r.Build(context.Background(), s, src, fakeRun, outData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}
