// Package php provides extension loading and recipe implementations for PHP builds.
//
// # Extension data files
//
// Extension YAML files define which PHP extensions and native modules are compiled
// into PHP binaries. In production, these files are owned by the php-buildpack
// repository (config/extensions/) and passed to binary-builder at build time via
// the --extensions-dir CLI flag.
//
// The testdata/ directory in this package contains a copy of the extension YAML
// files for use in unit tests. These are NOT compiled into the binary.
//
// # Loading extensions
//
// [LoadFromDir] is the primary API. It reads base + patch YAML files from a
// directory on the filesystem and returns a merged [ExtensionSet].
//
// [Load] is a convenience wrapper for tests that delegates to LoadFromDir
// using the testdata/ directory relative to the caller's working directory.
// It should NOT be used in production code paths.
//
// # File naming convention
//
//   - Base files:  php{major}-base-extensions.yml   (e.g. php8-base-extensions.yml)
//   - Patch files: php{major}{minor}-extensions-patch.yml (e.g. php84-extensions-patch.yml)
package php

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Extension represents a single PHP extension or native module.
type Extension struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	MD5     string `yaml:"md5"`
	Klass   string `yaml:"klass"`
}

// ExtensionSet holds the full set of native modules and extensions for a PHP build.
type ExtensionSet struct {
	NativeModules []Extension `yaml:"native_modules"`
	Extensions    []Extension `yaml:"extensions"`
}

// patchYAML is the structure of a php{major}{minor}-extensions-patch.yml file.
type patchYAML struct {
	NativeModules *patchCategory `yaml:"native_modules"`
	Extensions    *patchCategory `yaml:"extensions"`
}

type patchCategory struct {
	Additions  []Extension `yaml:"additions"`
	Exclusions []Extension `yaml:"exclusions"`
}

// Load returns the ExtensionSet for the given PHP major+minor version by reading
// from the testdata/ directory in this package. This is a convenience function
// for tests only — production code should use [LoadFromDir] with the path provided
// via the --extensions-dir CLI flag.
//
// Merge rules (applied by applyPatch):
//   - Exclusions run first. If an exclusion has a matching addition, the addition
//     replaces the entry in-place (preserving list position). Otherwise the entry
//     is removed.
//   - Remaining additions (no matching exclusion) override by name if present,
//     or are appended.
func Load(phpMajor, phpMinor string) (*ExtensionSet, error) {
	return LoadFromDir("testdata", phpMajor, phpMinor)
}

// LoadFromDir returns the ExtensionSet for the given PHP major+minor version by
// reading YAML files from a directory on the filesystem. The directory must use
// the following naming convention:
//
//   - Base files:  php{major}-base-extensions.yml   (e.g. php8-base-extensions.yml)
//   - Patch files: php{major}{minor}-extensions-patch.yml (e.g. php84-extensions-patch.yml)
//
// This is the primary API for loading extension configuration. In production,
// the directory is provided by the php-buildpack repo via the --extensions-dir
// CLI flag.
func LoadFromDir(dir, phpMajor, phpMinor string) (*ExtensionSet, error) {
	basePath := filepath.Join(dir, fmt.Sprintf("php%s-base-extensions.yml", phpMajor))
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("php/extensions: reading base file %s: %w", basePath, err)
	}

	var patchData []byte
	patchPath := filepath.Join(dir, fmt.Sprintf("php%s%s-extensions-patch.yml", phpMajor, phpMinor))
	switch pd, err := os.ReadFile(patchPath); {
	case err == nil:
		patchData = pd
	case os.IsNotExist(err):
		// Missing patch file is not an error — not all minor versions have patches.
	default:
		return nil, fmt.Errorf("php/extensions: reading patch file %s: %w", patchPath, err)
	}

	return loadFromData(baseData, patchData, phpMajor, phpMinor)
}

// loadFromData is the shared implementation. It parses the base YAML, optionally
// applies a patch, and returns the merged ExtensionSet.
// When patchData is nil or empty, the base set is returned as-is.
func loadFromData(baseData, patchData []byte, phpMajor, phpMinor string) (*ExtensionSet, error) {
	var set ExtensionSet
	if err := yaml.Unmarshal(baseData, &set); err != nil {
		return nil, fmt.Errorf("php/extensions: parsing base file for PHP %s: %w", phpMajor, err)
	}

	if len(patchData) == 0 {
		return &set, nil
	}

	var patch patchYAML
	if err := yaml.Unmarshal(patchData, &patch); err != nil {
		return nil, fmt.Errorf("php/extensions: parsing patch file for PHP %s.%s: %w", phpMajor, phpMinor, err)
	}

	applyPatch(&set.NativeModules, patch.NativeModules)
	applyPatch(&set.Extensions, patch.Extensions)

	return &set, nil
}

// applyPatch applies exclusions (remove by name) and then additions
// (override by name or append) from a patch category to a slice of extensions.
//
// When an exclusion removes an entry that has a matching addition (same name),
// the addition is inserted at the original position so that build-order
// dependencies (e.g. memcached depending on igbinary) are preserved.
// Additions with no matching exclusion are appended at the end.
func applyPatch(list *[]Extension, cat *patchCategory) {
	if cat == nil {
		return
	}

	// Build a lookup of additions by name for O(1) access.
	addByName := make(map[string]Extension, len(cat.Additions))
	for _, add := range cat.Additions {
		addByName[add.Name] = add
	}

	// Apply exclusions. When the excluded name has a corresponding addition,
	// replace in-place to preserve position; otherwise remove.
	replaced := make(map[string]bool)
	for _, excl := range cat.Exclusions {
		if idx := indexByName(*list, excl.Name); idx >= 0 {
			if add, ok := addByName[excl.Name]; ok {
				(*list)[idx] = add
				replaced[excl.Name] = true
			} else {
				*list = append((*list)[:idx], (*list)[idx+1:]...)
			}
		}
	}

	// Apply remaining additions (those not already placed by the exclusion loop).
	for _, add := range cat.Additions {
		if replaced[add.Name] {
			continue
		}
		if idx := indexByName(*list, add.Name); idx >= 0 {
			(*list)[idx] = add
		} else {
			*list = append(*list, add)
		}
	}
}

func indexByName(list []Extension, name string) int {
	for i, e := range list {
		if e.Name == name {
			return i
		}
	}
	return -1
}
