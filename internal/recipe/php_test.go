package recipe_test

import (
	"context"
	"os"
	"path/filepath"
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

// phpTestdataDir is the absolute path to the extension YAML testdata files.
// Must be resolved at init time before any test calls useTempWorkDir() which
// changes the working directory.
var phpTestdataDir string

func init() {
	abs, err := filepath.Abs("../php/testdata")
	if err != nil {
		panic("php_test: cannot resolve testdata path: " + err.Error())
	}
	phpTestdataDir = abs
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

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: phpTestdataDir}
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

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: phpTestdataDir}
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

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: phpTestdataDir}
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

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: phpTestdataDir}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// SubDependencies should be populated from the extension YAML testdata.
	// Check a representative sample of well-known extensions.
	require.NotNil(t, outData.SubDependencies)
	assert.Contains(t, outData.SubDependencies, "apcu", "apcu should be in sub-dependencies")
	assert.Contains(t, outData.SubDependencies, "rabbitmq", "rabbitmq should be in sub-dependencies")
	assert.Greater(t, len(outData.SubDependencies), 20, "should have many sub-dependencies from the testdata YAML")
}

func TestPHPRecipeBuildDownloadsSource(t *testing.T) {
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{
		Version: "8.3.2",
		URL:     "https://www.php.net/distributions/php-8.3.2.tar.gz",
		SHA256:  "abc123",
	}
	outData := &output.OutData{}

	f := newFakeFetcher()
	r := &recipe.PHPRecipe{Fetcher: f, ExtensionsDir: phpTestdataDir}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// Should download PHP source via Fetcher (not wget).
	assert.True(t, hasDownload(f, src.URL), "should download PHP source via Fetcher")
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

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: phpTestdataDir}
	err := r.Build(context.Background(), s, src, fakeRun, outData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}

func TestPHPRecipeBuildExtensionsDirDispatch(t *testing.T) {
	// When ExtensionsDir is set, PHPRecipe should load extensions from
	// that directory via LoadFromDir rather than using the embedded defaults.
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	// Create a minimal extensions directory with a single extension.
	extDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		extDir+"/php8-base-extensions.yml",
		[]byte("---\nextensions:\n- name: testonly\n  version: 1.0.0\n  md5: abc\n  klass: PeclRecipe\n"),
		0644,
	))

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: extDir}
	_ = r.Build(context.Background(), s, src, fakeRun, outData)

	// SubDependencies should contain "testonly" from our custom dir,
	// NOT the full set from the embedded YAML.
	require.NotNil(t, outData.SubDependencies)
	assert.Contains(t, outData.SubDependencies, "testonly",
		"should load extensions from ExtensionsDir, not embedded defaults")
	assert.NotContains(t, outData.SubDependencies, "apcu",
		"should NOT contain embedded extensions when ExtensionsDir is set")
}

func TestPHPRecipeBuildExtensionsDirMissingBaseFile(t *testing.T) {
	// When ExtensionsDir points to an empty directory, Build should fail
	// with a clear error about the missing base file.
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	emptyDir := t.TempDir()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: emptyDir}
	err := r.Build(context.Background(), s, src, fakeRun, outData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extensions")
}

func TestPHPRecipeBuildRequiresExtensionsDir(t *testing.T) {
	// ExtensionsDir is required. Build should fail with a clear error when
	// it is not set (empty string).
	useTempWorkDir(t)
	fakeRun := runner.NewFakeRunner()

	s := &stack.Stack{
		Name:        "cflinuxfs4",
		AptPackages: map[string][]string{"php_build": {}},
		PHPSymlinks: []stack.Symlink{},
	}
	src := &source.Input{Version: "8.3.2"}
	outData := &output.OutData{}

	r := &recipe.PHPRecipe{Fetcher: newFakeFetcher(), ExtensionsDir: ""}
	err := r.Build(context.Background(), s, src, fakeRun, outData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--extensions-dir is required")
}
