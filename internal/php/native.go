package php

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// LuaRecipe downloads and builds Lua from lua.org.
// Compiles with `make linux MYCFLAGS=-fPIC` and installs with `make install INSTALL_TOP={path}`.
type LuaRecipe struct{}

func (l *LuaRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://www.lua.org/ftp/lua-%s.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("lua-%s.tar.gz", ext.Version)
	srcDir := fmt.Sprintf("/tmp/lua-%s", ext.Version)
	installPath := fmt.Sprintf("/tmp/lua-install-%s", ext.Version)

	if err := run.Run("wget", "-O", fmt.Sprintf("/tmp/%s", archiveName), url); err != nil {
		return fmt.Errorf("php/lua: download: %w", err)
	}
	if err := run.Run("tar", "xzf", fmt.Sprintf("/tmp/%s", archiveName), "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php/lua: extract: %w", err)
	}
	if err := run.RunInDir(srcDir, "bash", "-c", "make linux MYCFLAGS=-fPIC"); err != nil {
		return fmt.Errorf("php/lua: make: %w", err)
	}
	installCmd := fmt.Sprintf("make install INSTALL_TOP=%s", installPath)
	if err := run.RunInDir(srcDir, "bash", "-c", installCmd); err != nil {
		return fmt.Errorf("php/lua: make install: %w", err)
	}
	return nil
}
