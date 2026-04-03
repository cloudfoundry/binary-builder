// Package apt provides a wrapper around apt-get for installing packages.
package apt

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// APT wraps apt-get operations using an injected Runner.
type APT struct {
	Runner runner.Runner
}

// New creates an APT instance with the given runner.
func New(r runner.Runner) *APT {
	return &APT{Runner: r}
}

// Update runs apt-get update.
func (a *APT) Update(_ context.Context) error {
	return a.Runner.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", "update",
	)
}

// Install runs apt-get install -y for the given packages.
// Does nothing if no packages are provided.
func (a *APT) Install(_ context.Context, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := append([]string{"install", "-y"}, packages...)
	return a.Runner.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", args...,
	)
}

// AddPPA adds a PPA repository and runs apt-get update.
// If ppa is empty, this is a no-op (cflinuxfs5 does not need PPAs).
func (a *APT) AddPPA(_ context.Context, ppa string) error {
	if ppa == "" {
		return nil
	}

	if err := a.Runner.Run("add-apt-repository", "-y", ppa); err != nil {
		return fmt.Errorf("adding PPA %s: %w", ppa, err)
	}

	return a.Runner.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", "update",
	)
}

// InstallReinstall runs apt-get -d install --reinstall to download .deb files
// without installing them. Used by the Python recipe for tcl/tk debs.
//
// When useForceYes is true, passes --force-yes (cflinuxfs4 compatibility).
// When false, passes --yes (cflinuxfs5 / modern apt).
func (a *APT) InstallReinstall(_ context.Context, useForceYes bool, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	var forceFlag string
	if useForceYes {
		forceFlag = "--force-yes"
	} else {
		forceFlag = "--yes"
	}

	args := []string{forceFlag, "-d", "install", "--reinstall"}
	args = append(args, packages...)

	return a.Runner.RunWithEnv(
		map[string]string{"DEBIAN_FRONTEND": "noninteractive"},
		"apt-get", args...,
	)
}
