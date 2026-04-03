package php_test

import (
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
