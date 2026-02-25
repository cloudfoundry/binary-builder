package runner_test

import (
	"errors"
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeRunnerRecordsCalls(t *testing.T) {
	f := runner.NewFakeRunner()

	err := f.Run("apt-get", "-y", "install", "foo")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"-y", "install", "foo"}, f.Calls[0].Args)
}

func TestFakeRunnerRunWithEnv(t *testing.T) {
	f := runner.NewFakeRunner()
	env := map[string]string{"DEBIAN_FRONTEND": "noninteractive"}

	err := f.RunWithEnv(env, "apt-get", "update")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "apt-get", f.Calls[0].Name)
	assert.Equal(t, []string{"update"}, f.Calls[0].Args)
	assert.Equal(t, "noninteractive", f.Calls[0].Env["DEBIAN_FRONTEND"])
}

func TestFakeRunnerRunInDir(t *testing.T) {
	f := runner.NewFakeRunner()

	err := f.RunInDir("/tmp/build", "make", "install")
	require.NoError(t, err)

	require.Len(t, f.Calls, 1)
	assert.Equal(t, "make", f.Calls[0].Name)
	assert.Equal(t, []string{"install"}, f.Calls[0].Args)
	assert.Equal(t, "/tmp/build", f.Calls[0].Dir)
}

func TestFakeRunnerOutputReturnsConfiguredValue(t *testing.T) {
	f := runner.NewFakeRunner()
	f.OutputMap["git describe --tags"] = "v1.2.3"

	out, err := f.Output("git", "describe", "--tags")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", out)
}

func TestFakeRunnerOutputReturnsEmptyForUnknown(t *testing.T) {
	f := runner.NewFakeRunner()

	out, err := f.Output("unknown", "command")
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestFakeRunnerErrorMapTriggersError(t *testing.T) {
	f := runner.NewFakeRunner()
	f.ErrorMap["make install"] = errors.New("make failed")

	err := f.Run("make", "install")
	require.Error(t, err)
	assert.Equal(t, "make failed", err.Error())
}

func TestFakeRunnerErrorMapForOutput(t *testing.T) {
	f := runner.NewFakeRunner()
	f.ErrorMap["git status"] = errors.New("not a git repo")

	_, err := f.Output("git", "status")
	require.Error(t, err)
	assert.Equal(t, "not a git repo", err.Error())
}

func TestFakeRunnerMultipleCalls(t *testing.T) {
	f := runner.NewFakeRunner()

	_ = f.Run("apt-get", "update")
	_ = f.Run("apt-get", "install", "-y", "gcc")
	_ = f.RunInDir("/build", "make")

	require.Len(t, f.Calls, 3)
	assert.Equal(t, "apt-get update", f.Calls[0].String())
	assert.Equal(t, "apt-get install -y gcc", f.Calls[1].String())
	assert.Equal(t, "make", f.Calls[2].String())
	assert.Equal(t, "/build", f.Calls[2].Dir)
}

func TestCallString(t *testing.T) {
	c := runner.Call{Name: "wget", Args: []string{"-q", "-O", "/tmp/file", "http://example.com"}}
	assert.Equal(t, "wget -q -O /tmp/file http://example.com", c.String())
}

func TestRealRunnerRunSuccess(t *testing.T) {
	r := &runner.RealRunner{}
	err := r.Run("true")
	require.NoError(t, err)
}

func TestRealRunnerRunFailure(t *testing.T) {
	r := &runner.RealRunner{}
	err := r.Run("false")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "false")
}
