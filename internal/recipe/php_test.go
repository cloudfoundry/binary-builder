package recipe_test

import (
	"context"
	"os"
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

// fakePHPExtensionsDir returns a temp directory with minimal PHP extension YAMLs.
// The minimal YAML has a single native module and a single extension so tests
// can assert the recipe wires them up correctly without loading the full ~45-extension file.
func fakePHPExtensionsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	base := `
native_modules:
  - name: hiredis
    version: "1.2.0"
    md5: abc123
    klass: HiredisRecipe
extensions:
  - name: apcu
    version: "5.1.23"
    md5: def456
    klass: PeclRecipe
`
	if err := writeExtFile(t, dir, "php8-base-extensions.yml", base); err != nil {
		t.Fatalf("fakePHPExtensionsDir: %v", err)
	}
	return dir
}

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
	extDir := fakePHPExtensionsDir(t)

	r := &recipe.PHPRecipe{ExtensionsDir: extDir}
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
	extDir := fakePHPExtensionsDir(t)

	r := &recipe.PHPRecipe{ExtensionsDir: extDir}
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
	extDir := fakePHPExtensionsDir(t)

	r := &recipe.PHPRecipe{ExtensionsDir: extDir}
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
	extDir := fakePHPExtensionsDir(t)

	r := &recipe.PHPRecipe{ExtensionsDir: extDir}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// Both the native module (hiredis) and extension (apcu) should be in sub-dependencies.
	require.NotNil(t, outData.SubDependencies)
	assert.Contains(t, outData.SubDependencies, "hiredis")
	assert.Contains(t, outData.SubDependencies, "apcu")
	assert.Equal(t, "1.2.0", outData.SubDependencies["hiredis"].Version)
	assert.Equal(t, "5.1.23", outData.SubDependencies["apcu"].Version)
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
	extDir := fakePHPExtensionsDir(t)

	r := &recipe.PHPRecipe{ExtensionsDir: extDir}
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

	r := &recipe.PHPRecipe{ExtensionsDir: t.TempDir()}
	err := r.Build(context.Background(), s, src, fakeRun, outData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}

// writeExtFile creates a YAML file at dir/name with the given content.
func writeExtFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	return os.WriteFile(dir+"/"+name, []byte(content), 0644)
}
