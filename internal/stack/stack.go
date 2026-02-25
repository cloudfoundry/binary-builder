// Package stack provides the Stack configuration struct and YAML loader.
// All Ubuntu-version-specific values live in stack YAML files — no stack
// names appear in Go source code.
package stack

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GfortranConfig holds gfortran compiler settings for a specific stack.
type GfortranConfig struct {
	Version  int      `yaml:"version"`
	Bin      string   `yaml:"bin"`
	LibPath  string   `yaml:"lib_path"`
	Packages []string `yaml:"packages"`
}

// GCCConfig holds GCC/g++ compiler settings for a specific stack.
type GCCConfig struct {
	Version  int      `yaml:"version"`
	Packages []string `yaml:"packages"`
	PPA      string   `yaml:"ppa"`
}

// CompilerConfig groups all compiler configurations.
type CompilerConfig struct {
	Gfortran GfortranConfig `yaml:"gfortran"`
	GCC      GCCConfig      `yaml:"gcc"`
}

// RubyBootstrap holds the pre-built Ruby binary used to bootstrap builds.
type RubyBootstrap struct {
	URL        string `yaml:"url"`
	SHA256     string `yaml:"sha256"`
	InstallDir string `yaml:"install_dir"`
}

// JRubyConfig holds JDK settings for JRuby builds.
type JRubyConfig struct {
	JDKURL        string `yaml:"jdk_url"`
	JDKSHA256     string `yaml:"jdk_sha256"`
	JDKInstallDir string `yaml:"jdk_install_dir"`
}

// PythonConfig holds Python-specific build settings.
type PythonConfig struct {
	TCLVersion  string `yaml:"tcl_version"`
	UseForceYes bool   `yaml:"use_force_yes"`
}

// Symlink represents a filesystem symlink to create during builds.
type Symlink struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
}

// Stack holds all configuration for a specific Ubuntu stack (cflinuxfs4, cflinuxfs5, etc.).
// Every Ubuntu-version-specific value lives here — recipes read from this struct
// and never contain hardcoded stack names or version numbers.
type Stack struct {
	Name           string              `yaml:"name"`
	UbuntuVersion  string              `yaml:"ubuntu_version"`
	UbuntuCodename string              `yaml:"ubuntu_codename"`
	DockerImage    string              `yaml:"docker_image"`
	RubyBootstrap  RubyBootstrap       `yaml:"ruby_bootstrap"`
	Compilers      CompilerConfig      `yaml:"compilers"`
	AptPackages    map[string][]string `yaml:"apt_packages"`
	PHPSymlinks    []Symlink           `yaml:"php_symlinks"`
	JRuby          JRubyConfig         `yaml:"jruby"`
	Python         PythonConfig        `yaml:"python"`
}

// Load reads a stack YAML file from stacksDir for the given stack name.
// Returns an error if the file does not exist or cannot be parsed.
func Load(stacksDir, name string) (*Stack, error) {
	path := filepath.Join(stacksDir, name+".yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading stack %q: %w", name, err)
	}

	var s Stack
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing stack %q: %w", name, err)
	}

	if s.Name == "" {
		return nil, fmt.Errorf("stack %q: name field is empty", name)
	}

	if s.Name != name {
		return nil, fmt.Errorf("stack file %q declares name %q (expected %q)", path, s.Name, name)
	}

	return &s, nil
}
