package stack_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stacksDir(t *testing.T) string {
	t.Helper()
	// Walk up to find the stacks/ directory relative to the repo root.
	dir, err := filepath.Abs("../../stacks")
	require.NoError(t, err)
	return dir
}

func TestLoadCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Equal(t, "cflinuxfs4", s.Name)
	assert.Equal(t, "22.04", s.UbuntuVersion)
	assert.Equal(t, "jammy", s.UbuntuCodename)
	assert.Equal(t, "cloudfoundry/cflinuxfs4", s.DockerImage)
}

func TestLoadCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.Equal(t, "cflinuxfs5", s.Name)
	assert.Equal(t, "24.04", s.UbuntuVersion)
	assert.Equal(t, "noble", s.UbuntuCodename)
	assert.Equal(t, "cloudfoundry/cflinuxfs5", s.DockerImage)
}

func TestGfortranVersionCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Equal(t, 11, s.Compilers.Gfortran.Version)
	assert.Equal(t, "/usr/bin/x86_64-linux-gnu-gfortran-11", s.Compilers.Gfortran.Bin)
	assert.Equal(t, "/usr/lib/gcc/x86_64-linux-gnu/11", s.Compilers.Gfortran.LibPath)
}

func TestGfortranVersionCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.Equal(t, 14, s.Compilers.Gfortran.Version)
	assert.Equal(t, "/usr/bin/x86_64-linux-gnu-gfortran-14", s.Compilers.Gfortran.Bin)
	assert.Equal(t, "/usr/lib/gcc/x86_64-linux-gnu/14", s.Compilers.Gfortran.LibPath)
}

func TestGCCPPACflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Equal(t, "ppa:ubuntu-toolchain-r/test", s.Compilers.GCC.PPA)
	assert.Equal(t, 12, s.Compilers.GCC.Version)
	assert.Contains(t, s.Compilers.GCC.Packages, "gcc-12")
	assert.Contains(t, s.Compilers.GCC.Packages, "g++-12")
}

func TestGCCPPACflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.Equal(t, "", s.Compilers.GCC.PPA)
	assert.Equal(t, 14, s.Compilers.GCC.Version)
	assert.Contains(t, s.Compilers.GCC.Packages, "gcc-14")
	assert.Contains(t, s.Compilers.GCC.Packages, "g++-14")
}

func TestPHPSymlinksCflinuxfs4HasLibldapR(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	found := false
	for _, sym := range s.PHPSymlinks {
		if sym.Dst == "/usr/lib/libldap_r.so" {
			found = true
			break
		}
	}
	assert.True(t, found, "cflinuxfs4 should have libldap_r.so symlink")
}

func TestPHPSymlinksCflinuxfs5NoLibldapR(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	for _, sym := range s.PHPSymlinks {
		assert.NotEqual(t, "/usr/lib/libldap_r.so", sym.Dst,
			"cflinuxfs5 should NOT have libldap_r.so symlink (dropped in OpenLDAP 2.6)")
	}
}

func TestPythonUseForceYesCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.True(t, s.Python.UseForceYes)
}

func TestPythonUseForceYesCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.False(t, s.Python.UseForceYes)
}

func TestPHPBuildPackagesCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	pkgs := s.AptPackages["php_build"]
	assert.Contains(t, pkgs, "libdb-dev")
	assert.NotContains(t, pkgs, "libdb5.3-dev")
	assert.Contains(t, pkgs, "libzookeeper-mt-dev")
}

func TestPHPBuildPackagesCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	pkgs := s.AptPackages["php_build"]
	assert.Contains(t, pkgs, "libdb5.3-dev")
	assert.NotContains(t, pkgs, "libdb-dev")
	assert.NotContains(t, pkgs, "libzookeeper-mt-dev")
}

func TestRBuildPackagesCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	pkgs := s.AptPackages["r_build"]
	assert.Contains(t, pkgs, "libpcre++-dev")
	assert.Contains(t, pkgs, "libtiff5-dev")
}

func TestRBuildPackagesCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	pkgs := s.AptPackages["r_build"]
	assert.NotContains(t, pkgs, "libpcre++-dev")
	assert.Contains(t, pkgs, "libtiff-dev")
	assert.NotContains(t, pkgs, "libtiff5-dev")
}

func TestLoadMissingFile(t *testing.T) {
	_, err := stack.Load(stacksDir(t), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestLoadMalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte("{{invalid yaml"), 0644)
	require.NoError(t, err)

	_, err = stack.Load(tmpDir, "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestLoadEmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "empty.yaml"), []byte("ubuntu_version: '22.04'\n"), 0644)
	require.NoError(t, err)

	_, err = stack.Load(tmpDir, "empty")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name field is empty")
}

func TestLoadNameMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "wrong.yaml"), []byte("name: other\nubuntu_version: '22.04'\n"), 0644)
	require.NoError(t, err)

	_, err = stack.Load(tmpDir, "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected \"wrong\"")
}

func TestJRubyConfigCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Contains(t, s.JRuby.JDKURL, "bionic")
	assert.Equal(t, "/opt/java", s.JRuby.JDKInstallDir)
}

func TestJRubyConfigCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.Contains(t, s.JRuby.JDKURL, "noble")
	assert.Equal(t, "/opt/java", s.JRuby.JDKInstallDir)
}

func TestRubyBootstrapCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Contains(t, s.RubyBootstrap.URL, "cflinuxfs4")
	assert.Equal(t, "/opt/ruby", s.RubyBootstrap.InstallDir)
}

func TestGoBootstrapCflinuxfs4(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Contains(t, s.Go.BootstrapURL, "go.dev/dl/")
	assert.Contains(t, s.Go.BootstrapURL, "linux-amd64.tar.gz")
}

func TestGoBootstrapCflinuxfs5(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs5")
	require.NoError(t, err)

	assert.Contains(t, s.Go.BootstrapURL, "go.dev/dl/")
	assert.Contains(t, s.Go.BootstrapURL, "linux-amd64.tar.gz")
}

func TestPythonTCLVersion(t *testing.T) {
	s, err := stack.Load(stacksDir(t), "cflinuxfs4")
	require.NoError(t, err)

	assert.Equal(t, "8.6", s.Python.TCLVersion)
}
