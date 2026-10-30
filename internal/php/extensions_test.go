package php_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/php"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_PatchOverridesExisting verifies that an addition whose name already
// exists in the base replaces the existing entry (override path), and that the
// result contains exactly one entry for that name (no duplicates).
// php81-extensions-patch.yml overrides oci8, which is also present in the base.
//
// TODO: the pure-append path (patch adds a name not present in the base at all)
// has no integration-level coverage here; it would require a dedicated patch
// fixture that introduces a genuinely new extension name.
func TestLoad_PatchOverridesExisting(t *testing.T) {
	// php81 patch adds oci8 (as an override — oci8 exists in base with a
	// different version). Verify the result contains exactly one oci8 entry.
	set, err := php.Load("8", "1")
	require.NoError(t, err)

	var oci8Count int
	for _, e := range set.Extensions {
		if e.Name == "oci8" {
			oci8Count++
		}
	}
	assert.Equal(t, 1, oci8Count, "oci8 should appear exactly once after patch")
}

// TestLoad_PatchOverridesVersion verifies that an addition whose name already
// exists in the base replaces the existing entry (version override).
// php81-extensions-patch.yml overrides oci8 to version 3.2.1.
func TestLoad_PatchOverridesVersion(t *testing.T) {
	set, err := php.Load("8", "1")
	require.NoError(t, err)

	var oci8 *php.Extension
	for i := range set.Extensions {
		if set.Extensions[i].Name == "oci8" {
			oci8 = &set.Extensions[i]
		}
	}
	require.NotNil(t, oci8, "oci8 should be present after php81 patch")
	assert.Equal(t, "3.2.1", oci8.Version)
	assert.Equal(t, "309190ef3ede2779a617c9375d32ea7a", oci8.MD5)
}

// TestLoad_PatchRemovesExclusion verifies that an exclusion in a patch file
// removes the named extension from the result.
// php82-extensions-patch.yml excludes yaf.
func TestLoad_PatchRemovesExclusion(t *testing.T) {
	set, err := php.Load("8", "2")
	require.NoError(t, err)

	for _, e := range set.Extensions {
		assert.NotEqual(t, "yaf", e.Name, "yaf should be excluded in php82")
	}
}

// TestLoad_PatchNativeModuleAddition verifies that native modules are
// unaffected when a patch file has no native_modules section.
// php83-extensions-patch.yml has no native_modules section.
func TestLoad_PatchNativeModuleAddition(t *testing.T) {
	set, err := php.Load("8", "3")
	require.NoError(t, err)

	// Native modules from the base file should be present unchanged.
	nativeNames := make([]string, len(set.NativeModules))
	for i, m := range set.NativeModules {
		nativeNames[i] = m.Name
	}
	assert.Contains(t, nativeNames, "rabbitmq")
	assert.Contains(t, nativeNames, "lua")
}

// TestLoad_MissingBaseFile verifies that Load returns an error (mentioning
// "base") when no base file exists for the requested major version.
func TestLoad_MissingBaseFile(t *testing.T) {
	_, err := php.Load("9", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base")
}

// --- Smoke tests against the real embedded YAML files ---

func TestLoad_RealPhp8BaseFile(t *testing.T) {
	// Smoke-test: load the real php8-base-extensions.yml embedded in this package.
	set, err := php.Load("8", "4")
	require.NoError(t, err)

	// The base file must have the expected native modules.
	nativeNames := make([]string, len(set.NativeModules))
	for i, m := range set.NativeModules {
		nativeNames[i] = m.Name
	}
	assert.Contains(t, nativeNames, "rabbitmq")
	assert.Contains(t, nativeNames, "lua")
	assert.Contains(t, nativeNames, "hiredis")

	// Must have a meaningful number of extensions.
	assert.Greater(t, len(set.Extensions), 20)
}

func TestLoad_RealPhp81Patch(t *testing.T) {
	set, err := php.Load("8", "1")
	require.NoError(t, err)

	// php81 patch overrides oci8 version to 3.2.1
	var oci8 *php.Extension
	for i := range set.Extensions {
		if set.Extensions[i].Name == "oci8" {
			oci8 = &set.Extensions[i]
		}
	}
	require.NotNil(t, oci8, "oci8 should be present")
	assert.Equal(t, "3.2.1", oci8.Version)
}

func TestLoad_RealPhp82Patch(t *testing.T) {
	set, err := php.Load("8", "2")
	require.NoError(t, err)

	// php82 patch removes yaf
	for _, e := range set.Extensions {
		assert.NotEqual(t, "yaf", e.Name, "yaf should be excluded in php82")
	}
}

func TestLoad_RealPhp83Patch(t *testing.T) {
	set, err := php.Load("8", "3")
	require.NoError(t, err)

	// php83 patch also removes yaf
	for _, e := range set.Extensions {
		assert.NotEqual(t, "yaf", e.Name, "yaf should be excluded in php83")
	}
}

// --- LoadFromDir tests ---

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

func TestLoadFromDir_BaseOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `---
native_modules:
- name: lua
  version: 5.4.6
  md5: abc123
  klass: LuaRecipe
extensions:
- name: redis
  version: 6.0.2
  md5: def456
  klass: PeclRecipe
- name: mongodb
  version: 1.21.12
  md5: ghi789
  klass: PeclRecipe
`)

	set, err := php.LoadFromDir(dir, "8", "2")
	require.NoError(t, err)

	assert.Len(t, set.NativeModules, 1)
	assert.Equal(t, "lua", set.NativeModules[0].Name)
	assert.Equal(t, "5.4.6", set.NativeModules[0].Version)

	assert.Len(t, set.Extensions, 2)
	assert.Equal(t, "redis", set.Extensions[0].Name)
	assert.Equal(t, "mongodb", set.Extensions[1].Name)
	assert.Equal(t, "1.21.12", set.Extensions[1].Version)
}

func TestLoadFromDir_WithPatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `---
extensions:
- name: redis
  version: 6.0.2
  md5: aaa
  klass: PeclRecipe
- name: mongodb
  version: 1.18.1
  md5: bbb
  klass: PeclRecipe
- name: xdebug
  version: 3.3.2
  md5: ccc
  klass: PeclRecipe
`)
	writeFile(t, dir, "php84-extensions-patch.yml", `---
extensions:
  exclusions:
    - name: mongodb
      version: 1.18.1
      md5: bbb
      klass: PeclRecipe
    - name: xdebug
      version: 3.3.2
      md5: ccc
      klass: PeclRecipe
  additions:
    - name: mongodb
      version: 2.5.5
      md5: ddd
      klass: PeclRecipe
    - name: xdebug
      version: 3.4.0
      md5: eee
      klass: PeclRecipe
`)

	set, err := php.LoadFromDir(dir, "8", "4")
	require.NoError(t, err)

	// redis is unchanged from base.
	assert.Equal(t, "redis", set.Extensions[0].Name)
	assert.Equal(t, "6.0.2", set.Extensions[0].Version)

	// mongodb was excluded and replaced in-place.
	assert.Equal(t, "mongodb", set.Extensions[1].Name)
	assert.Equal(t, "2.5.5", set.Extensions[1].Version)

	// xdebug was excluded and replaced in-place.
	assert.Equal(t, "xdebug", set.Extensions[2].Name)
	assert.Equal(t, "3.4.0", set.Extensions[2].Version)

	assert.Len(t, set.Extensions, 3)
}

func TestLoadFromDir_MissingBaseFile(t *testing.T) {
	dir := t.TempDir()
	// No files — should fail with a clear error.
	_, err := php.LoadFromDir(dir, "8", "4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base file")
}

func TestLoadFromDir_MissingPatchIsOK(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `---
extensions:
- name: redis
  version: 6.0.2
  md5: aaa
  klass: PeclRecipe
`)

	// No patch file for php84 — should succeed with base only.
	set, err := php.LoadFromDir(dir, "8", "4")
	require.NoError(t, err)
	assert.Len(t, set.Extensions, 1)
	assert.Equal(t, "redis", set.Extensions[0].Name)
}

func TestLoadFromDir_MatchesLoad(t *testing.T) {
	// Load() delegates to LoadFromDir("testdata", ...), so both should
	// produce identical results. This verifies the delegation works.
	loadSet, err := php.Load("8", "4")
	require.NoError(t, err)

	// In tests, the working directory is the package directory (internal/php/),
	// so the testdata dir is at "./testdata".
	dirSet, err := php.LoadFromDir("testdata", "8", "4")
	require.NoError(t, err)

	assert.Equal(t, loadSet.NativeModules, dirSet.NativeModules,
		"NativeModules should match between Load and LoadFromDir")
	assert.Equal(t, loadSet.Extensions, dirSet.Extensions,
		"Extensions should match between Load and LoadFromDir")
}

func TestLoadFromDir_PatchFileReadError(t *testing.T) {
	// A patch file that exists but cannot be read (e.g. permission denied)
	// should return an error rather than silently falling back to base-only.
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `---
extensions:
- name: redis
  version: 6.0.2
  md5: aaa
  klass: PeclRecipe
`)
	// Create a patch file with no read permissions.
	patchPath := filepath.Join(dir, "php84-extensions-patch.yml")
	require.NoError(t, os.WriteFile(patchPath, []byte("---\n"), 0000))

	_, err := php.LoadFromDir(dir, "8", "4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "patch file")
}
