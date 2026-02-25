package php_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/php"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile is a helper to create a YAML file in dir with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}

func TestLoad_BaseOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `
native_modules:
  - name: rabbitmq
    version: "0.11.0"
    md5: abc123
    klass: RabbitMQRecipe
  - name: lua
    version: "5.4.6"
    md5: def456
    klass: LuaRecipe
extensions:
  - name: apcu
    version: "5.1.23"
    md5: ghi789
    klass: PeclRecipe
`)

	// No patch file exists — Load should succeed with base data.
	set, err := php.Load(dir, "8", "4")
	require.NoError(t, err)

	require.Len(t, set.NativeModules, 2)
	assert.Equal(t, "rabbitmq", set.NativeModules[0].Name)
	assert.Equal(t, "0.11.0", set.NativeModules[0].Version)
	assert.Equal(t, "lua", set.NativeModules[1].Name)

	require.Len(t, set.Extensions, 1)
	assert.Equal(t, "apcu", set.Extensions[0].Name)
}

func TestLoad_PatchAddsExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `
native_modules: []
extensions:
  - name: apcu
    version: "5.1.23"
    md5: abc
    klass: PeclRecipe
`)
	writeFile(t, dir, "php83-extensions-patch.yml", `
extensions:
  additions:
    - name: newext
      version: "1.0.0"
      md5: xyz
      klass: PeclRecipe
`)

	set, err := php.Load(dir, "8", "3")
	require.NoError(t, err)

	require.Len(t, set.Extensions, 2)
	assert.Equal(t, "apcu", set.Extensions[0].Name)
	assert.Equal(t, "newext", set.Extensions[1].Name)
}

func TestLoad_PatchOverridesVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `
native_modules: []
extensions:
  - name: oci8
    version: "3.3.0"
    md5: oldmd5
    klass: OraclePeclRecipe
`)
	writeFile(t, dir, "php81-extensions-patch.yml", `
extensions:
  additions:
    - name: oci8
      version: "3.2.1"
      md5: newmd5
      klass: OraclePeclRecipe
`)

	set, err := php.Load(dir, "8", "1")
	require.NoError(t, err)

	require.Len(t, set.Extensions, 1)
	assert.Equal(t, "3.2.1", set.Extensions[0].Version)
	assert.Equal(t, "newmd5", set.Extensions[0].MD5)
}

func TestLoad_PatchRemovesExclusion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `
native_modules: []
extensions:
  - name: yaf
    version: "3.3.5"
    md5: abc
    klass: PeclRecipe
  - name: apcu
    version: "5.1.23"
    md5: def
    klass: PeclRecipe
`)
	writeFile(t, dir, "php82-extensions-patch.yml", `
extensions:
  exclusions:
    - name: yaf
      version: "3.3.5"
      md5: abc
      klass: PeclRecipe
`)

	set, err := php.Load(dir, "8", "2")
	require.NoError(t, err)

	require.Len(t, set.Extensions, 1)
	assert.Equal(t, "apcu", set.Extensions[0].Name)
}

func TestLoad_PatchNativeModuleAddition(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `
native_modules:
  - name: rabbitmq
    version: "0.11.0"
    md5: abc
    klass: RabbitMQRecipe
extensions: []
`)
	writeFile(t, dir, "php83-extensions-patch.yml", `
native_modules:
  additions:
    - name: newlib
      version: "2.0.0"
      md5: xyz
      klass: SomeRecipe
`)

	set, err := php.Load(dir, "8", "3")
	require.NoError(t, err)

	require.Len(t, set.NativeModules, 2)
	assert.Equal(t, "newlib", set.NativeModules[1].Name)
}

func TestLoad_MissingBaseFile(t *testing.T) {
	dir := t.TempDir()
	_, err := php.Load(dir, "8", "3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base file")
}

func TestLoad_InvalidBaseYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "php8-base-extensions.yml", `not: valid: yaml: [`)
	_, err := php.Load(dir, "8", "3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing base file")
}

func TestLoad_RealPhp8BaseFile(t *testing.T) {
	// Smoke-test: load the real php8-base-extensions.yml shipped in this repo.
	set, err := php.Load("../../php_extensions", "8", "4")
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
	set, err := php.Load("../../php_extensions", "8", "1")
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
	set, err := php.Load("../../php_extensions", "8", "2")
	require.NoError(t, err)

	// php82 patch removes yaf
	for _, e := range set.Extensions {
		assert.NotEqual(t, "yaf", e.Name, "yaf should be excluded in php82")
	}
}

func TestLoad_RealPhp83Patch(t *testing.T) {
	set, err := php.Load("../../php_extensions", "8", "3")
	require.NoError(t, err)

	// php83 patch also removes yaf
	for _, e := range set.Extensions {
		assert.NotEqual(t, "yaf", e.Name, "yaf should be excluded in php83")
	}
}
