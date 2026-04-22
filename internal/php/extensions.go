// Package php provides extension loading and recipe implementations for PHP builds.
//
// # Extension data files
//
// The YAML files that define which PHP extensions and native modules are built
// live in assets/ alongside the Go source:
//
//	internal/php/assets/
//	  php8-base-extensions.yml      — full list for all PHP 8.x (native modules + PECL extensions)
//	  php81-extensions-patch.yml    — overrides/exclusions specific to PHP 8.1.x
//	  php82-extensions-patch.yml    — overrides/exclusions specific to PHP 8.2.x
//	  php83-extensions-patch.yml    — overrides/exclusions specific to PHP 8.3.x
//
// # Adding a new PHP minor version (e.g. 8.4)
//
//  1. Create assets/php84-extensions-patch.yml with any additions or exclusions
//     relative to the PHP 8 base (an empty patch is valid: "---\nextensions:\n").
//
// No code changes are required — the file is discovered automatically via the
// embedded FS glob.
//
// # Adding a new PHP major version (e.g. 9)
//
//  1. Create assets/php9-base-extensions.yml with the full extension list for PHP 9.x.
//  2. Create a patch file for each shipped minor version (e.g. assets/php90-extensions-patch.yml).
//
// Again, no code changes are required.
//
// # File naming convention (drives auto-discovery)
//
//   - Base files:  php{major}-base-extensions.yml   (e.g. php8-base-extensions.yml)
//   - Patch files: php{major}{minor}-extensions-patch.yml (e.g. php84-extensions-patch.yml)
//
// Important: only .yml files are embedded (the glob is assets/*.yml). A file with a
// .yaml extension would be silently ignored.
package php

import (
	"embed"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed assets/*.yml
var assetsFS embed.FS

var (
	// baseFileRE matches e.g. "assets/php8-base-extensions.yml" and captures the major version ("8").
	baseFileRE = regexp.MustCompile(`^assets/php(\d+)-base-extensions\.yml$`)
	// patchFileRE matches e.g. "assets/php83-extensions-patch.yml" and captures major+minor ("83").
	// The \d{2,} quantifier is intentional: patch filenames must encode both the major AND minor
	// digit together (e.g. "php81", "php90"), never a bare major digit. This prevents a file named
	// "php8-extensions-patch.yml" from matching (that would be a base file), and ensures a future
	// PHP 9 patch is named "php90-extensions-patch.yml", not "php9-extensions-patch.yml" (which
	// would be silently ignored).
	patchFileRE = regexp.MustCompile(`^assets/php(\d{2,})-extensions-patch\.yml$`)
)

// embeddedBases maps PHP major version (e.g. "8") → base YAML bytes.
// embeddedPatches maps PHP major+minor (e.g. "83") → patch YAML bytes.
// Both are populated once at init() from the embedded FS.
var (
	embeddedBases   map[string][]byte
	embeddedPatches map[string][]byte
)

func init() {
	embeddedBases = make(map[string][]byte)
	embeddedPatches = make(map[string][]byte)

	entries, err := assetsFS.ReadDir("assets")
	if err != nil {
		panic("php/extensions: cannot read embedded assets: " + err.Error())
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		path := "assets/" + e.Name()
		data, err := assetsFS.ReadFile(path)
		if err != nil {
			panic("php/extensions: cannot read embedded file " + path + ": " + err.Error())
		}

		if m := baseFileRE.FindStringSubmatch(path); m != nil {
			embeddedBases[m[1]] = data
		} else if m := patchFileRE.FindStringSubmatch(path); m != nil {
			embeddedPatches[m[1]] = data
		}
	}
}

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
// the embedded YAML data compiled into this package.
//
// It loads the base file for the major version (e.g. "8" → php8-base-extensions.yml),
// then applies the patch file for the specific minor version if one exists
// (e.g. "8"+"2" → php82-extensions-patch.yml).
//
// Merge rules (applied by applyPatch):
//   - Exclusions run first. If an exclusion has a matching addition, the addition
//     replaces the entry in-place (preserving list position). Otherwise the entry
//     is removed.
//   - Remaining additions (no matching exclusion) override by name if present,
//     or are appended.
func Load(phpMajor, phpMinor string) (*ExtensionSet, error) {
	baseData, ok := embeddedBases[phpMajor]
	if !ok {
		return nil, fmt.Errorf("php/extensions: no base extensions file for PHP major version %q", phpMajor)
	}

	var set ExtensionSet
	if err := yaml.Unmarshal(baseData, &set); err != nil {
		return nil, fmt.Errorf("php/extensions: parsing base file for PHP %s: %w", phpMajor, err)
	}

	patchData, ok := embeddedPatches[phpMajor+phpMinor]
	if !ok {
		// No patch for this minor version — return base as-is.
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
