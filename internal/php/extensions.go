// Package php provides extension loading and recipe implementations for PHP builds.
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

// Load reads the base YAML for the given PHP major version (e.g. "8"), applies
// the patch YAML for the given major+minor (e.g. "8"+"3"), and returns the
// merged ExtensionSet.
//
// Base file:  {extensionsDir}/php{major}-base-extensions.yml
// Patch file: {extensionsDir}/php{major}{minor}-extensions-patch.yml
//
// Merge rules:
//   - For each addition: if name already exists → replace; otherwise → append.
//   - For each exclusion: remove by name.
func Load(extensionsDir, phpMajor, phpMinor string) (*ExtensionSet, error) {
	basePath := filepath.Join(extensionsDir, fmt.Sprintf("php%s-base-extensions.yml", phpMajor))

	data, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("php/extensions: reading base file %s: %w", basePath, err)
	}

	var set ExtensionSet
	if err := yaml.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("php/extensions: parsing base file %s: %w", basePath, err)
	}

	patchPath := filepath.Join(extensionsDir, fmt.Sprintf("php%s%s-extensions-patch.yml", phpMajor, phpMinor))

	patchData, err := os.ReadFile(patchPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No patch file — return base as-is.
			return &set, nil
		}
		return nil, fmt.Errorf("php/extensions: reading patch file %s: %w", patchPath, err)
	}

	var patch patchYAML
	if err := yaml.Unmarshal(patchData, &patch); err != nil {
		return nil, fmt.Errorf("php/extensions: parsing patch file %s: %w", patchPath, err)
	}

	applyPatch(&set.NativeModules, patch.NativeModules)
	applyPatch(&set.Extensions, patch.Extensions)

	return &set, nil
}

// applyPatch applies additions (override by name or append) and exclusions
// (remove by name) from a patch category to a slice of extensions.
func applyPatch(list *[]Extension, cat *patchCategory) {
	if cat == nil {
		return
	}

	for _, add := range cat.Additions {
		if idx := indexByName(*list, add.Name); idx >= 0 {
			(*list)[idx] = add
		} else {
			*list = append(*list, add)
		}
	}

	for _, excl := range cat.Exclusions {
		if idx := indexByName(*list, excl.Name); idx >= 0 {
			*list = append((*list)[:idx], (*list)[idx+1:]...)
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
