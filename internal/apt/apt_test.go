package apt_test

import (
	"context"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallPackages(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.Install(context.Background(), "pkg1", "pkg2")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"install", "-y", "pkg1", "pkg2"}, f.Calls[0].Args)
	assert.Equal(t, "noninteractive", f.Calls[0].Env["DEBIAN_FRONTEND"])
}

func TestInstallNoPackages(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.Install(context.Background())
	require.NoError(t, err)

	assert.Empty(t, f.Calls)
}

func TestUpdate(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.Update(context.Background())
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"update"}, f.Calls[0].Args)
	assert.Equal(t, "noninteractive", f.Calls[0].Env["DEBIAN_FRONTEND"])
}

func TestAddPPANonEmpty(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.AddPPA(context.Background(), "ppa:ubuntu-toolchain-r/test")
	require.NoError(t, err)

	require.Len(t, f.Calls, 2)
	// First call: add-apt-repository
	assert.Equal(t, "add-apt-repository", f.Calls[0].Name)
	assert.Equal(t, []string{"-y", "ppa:ubuntu-toolchain-r/test"}, f.Calls[0].Args)
	// Second call: apt-get update
	assert.Equal(t, "apt-get", f.Calls[1].Name)
	assert.Equal(t, []string{"update"}, f.Calls[1].Args)
}

func TestAddPPAEmpty(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.AddPPA(context.Background(), "")
	require.NoError(t, err)

	assert.Empty(t, f.Calls, "empty PPA should be a no-op")
}

func TestInstallReinstallWithForceYes(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.InstallReinstall(context.Background(), true, "libtcl8.6", "libtk8.6", "libxss1")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"--force-yes", "-d", "install", "--reinstall", "libtcl8.6", "libtk8.6", "libxss1"}, f.Calls[0].Args)
}

func TestInstallReinstallWithoutForceYes(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.InstallReinstall(context.Background(), false, "libtcl8.6", "libtk8.6", "libxss1")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"--yes", "-d", "install", "--reinstall", "libtcl8.6", "libtk8.6", "libxss1"}, f.Calls[0].Args)
	// Verify --force-yes is NOT present
	for _, arg := range f.Calls[0].Args {
		assert.NotEqual(t, "--force-yes", arg)
	}
}

func TestInstallReinstallNoPackages(t *testing.T) {
	f := runner.NewFakeRunner()
	a := apt.New(f)

	err := a.InstallReinstall(context.Background(), true)
	require.NoError(t, err)

	assert.Empty(t, f.Calls)
}
