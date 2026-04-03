package php

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// FakePeclRecipe builds built-in PHP extensions that live inside the PHP source tree
// under ext/{name}. It tars up the source directory and then uses the PECL build steps.
type FakePeclRecipe struct{}

func (f *FakePeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	opts := []string{fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath)}
	return buildFakePecl(ctx, ext.Name, ec, opts, run)
}

// SodiumRecipe builds the sodium extension (built-in since PHP 7.2) against libsodium.
type SodiumRecipe struct{}

func (s *SodiumRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		fmt.Sprintf("--with-sodium=%s", ec.LibSodiumPath),
	}
	env := map[string]string{
		"LDFLAGS":         fmt.Sprintf("-L%s/lib", ec.LibSodiumPath),
		"PKG_CONFIG_PATH": fmt.Sprintf("%s/lib/pkgconfig", ec.LibSodiumPath),
	}
	if err := buildFakePeclWithEnv(ctx, ext.Name, ec, opts, env, run); err != nil {
		return err
	}
	// Copy libsodium shared libs into the PHP prefix.
	return run.Run("sh", "-c", fmt.Sprintf("cp -a %s/lib/libsodium.so* %s/lib/", ec.LibSodiumPath, ec.PHPPath))
}

// OdbcRecipe builds the ODBC extension (built-in) with unixODBC.
type OdbcRecipe struct{}

func (o *OdbcRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	workDir := fmt.Sprintf("%s/ext/odbc", ec.PHPSourceDir)
	// Patch config.m4 to add AC_DEFUN before contents.
	patchScript := fmt.Sprintf(
		`cd %s && echo 'AC_DEFUN([PHP_ALWAYS_SHARED],[])dnl' > temp.m4 && echo >> temp.m4 && cat config.m4 >> temp.m4 && mv temp.m4 config.m4`,
		workDir,
	)
	if err := run.Run("sh", "-c", patchScript); err != nil {
		return fmt.Errorf("php/odbc: patch config.m4: %w", err)
	}

	opts := []string{"--with-unixODBC=shared,/usr"}
	if err := buildFakePeclFromDir(ctx, "odbc", workDir, ec, opts, run); err != nil {
		return err
	}
	// Copy odbc libs into PHP prefix.
	return run.Run("sh", "-c", fmt.Sprintf(
		"cp -a /usr/lib/x86_64-linux-gnu/libodbc.so* %s/lib/ && cp -a /usr/lib/x86_64-linux-gnu/libodbcinst.so* %s/lib/",
		ec.PHPPath, ec.PHPPath,
	))
}

// PdoOdbcRecipe builds the PDO ODBC extension (built-in) with unixODBC.
type PdoOdbcRecipe struct{}

func (p *PdoOdbcRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	opts := []string{"--with-pdo-odbc=unixODBC,/usr"}
	if err := buildFakePecl(ctx, "pdo_odbc", ec, opts, run); err != nil {
		return err
	}
	return run.Run("sh", "-c", fmt.Sprintf(
		"cp -a /usr/lib/x86_64-linux-gnu/libodbc.so* %s/lib/ && cp -a /usr/lib/x86_64-linux-gnu/libodbcinst.so* %s/lib/",
		ec.PHPPath, ec.PHPPath,
	))
}

// SnmpRecipe copies SNMP system libraries and mibs into the PHP prefix.
// No external download or compilation — just copies from system paths.
type SnmpRecipe struct{}

func (s *SnmpRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	script := fmt.Sprintf(`
cd %s
mkdir -p mibs
cp -a /usr/lib/x86_64-linux-gnu/libnetsnmp.so* lib/
cp -rL /usr/share/snmp/mibs/iana mibs/
cp -rL /usr/share/snmp/mibs/ietf mibs/
cp /usr/share/snmp/mibs/*.txt mibs/
cp /usr/bin/download-mibs bin/
cp /usr/bin/smistrip bin/
sed -i "s|^CONFDIR=/etc/snmp-mibs-downloader|CONFDIR=\$HOME/php/mibs/conf|" bin/download-mibs
sed -i "s|^SMISTRIP=/usr/bin/smistrip|SMISTRIP=\$HOME/php/bin/smistrip|" bin/download-mibs
cp -R /etc/snmp-mibs-downloader mibs/conf
cp -R /etc/snmp-mibs-downloader mibs/conf
sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/iana.conf
sed -i "s|^DEST=iana|DEST=|" mibs/conf/iana.conf
sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/ianarfc.conf
sed -i "s|^DEST=iana|DEST=|" mibs/conf/ianarfc.conf
sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/rfc.conf
sed -i "s|^DEST=ietf|DEST=|" mibs/conf/rfc.conf
sed -i "s|^BASEDIR=/var/lib/mibs|BASEDIR=\$HOME/php/mibs|" mibs/conf/snmp-mibs-downloader.conf
`, ec.PHPPath)
	if err := run.Run("sh", "-c", script); err != nil {
		return fmt.Errorf("php/snmp: setup: %w", err)
	}
	return nil
}

// Gd74FakePeclRecipe builds the GD extension (PHP 7.4+) using the system libgd.
type Gd74FakePeclRecipe struct{}

func (g *Gd74FakePeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	opts := []string{"--with-external-gd"}
	return buildFakePecl(ctx, "gd", ec, opts, run)
}

// EnchantFakePeclRecipe builds the enchant extension with a source patch.
type EnchantFakePeclRecipe struct{}

func (e *EnchantFakePeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	workDir := fmt.Sprintf("%s/ext/enchant", ec.PHPSourceDir)
	// Patch the broken include path.
	patchScript := fmt.Sprintf(
		`cd %s && sed -i 's|#include "../spl/spl_exceptions.h"|#include <spl/spl_exceptions.h>|' enchant.c`,
		workDir,
	)
	if err := run.Run("sh", "-c", patchScript); err != nil {
		return fmt.Errorf("php/enchant: patch: %w", err)
	}
	opts := []string{fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath)}
	return buildFakePeclFromDir(ctx, "enchant", workDir, ec, opts, run)
}

// buildFakePecl archives ec.PHPSourceDir/ext/{name}, then runs phpize+configure+make+install.
func buildFakePecl(ctx context.Context, name string, ec ExtensionContext, opts []string, run runner.Runner) error {
	workDir := fmt.Sprintf("%s/ext/%s", ec.PHPSourceDir, name)
	return buildFakePeclFromDir(ctx, name, workDir, ec, opts, run)
}

// buildFakePeclWithEnv is like buildFakePecl but runs phpize/configure with extra env vars.
func buildFakePeclWithEnv(ctx context.Context, name string, ec ExtensionContext, opts []string, env map[string]string, run runner.Runner) error {
	workDir := fmt.Sprintf("%s/ext/%s", ec.PHPSourceDir, name)

	// Merge PHP bin dir into PATH so phpize and php-config are found by name.
	mergedEnv := mergePHPBinPath(ec.PHPPath, env)

	if err := run.RunInDirWithEnv(workDir, mergedEnv, fmt.Sprintf("%s/bin/phpize", ec.PHPPath)); err != nil {
		return fmt.Errorf("php/%s: phpize: %w", name, err)
	}

	configureArgs := append([]string{"./configure"}, opts...)
	if err := run.RunInDirWithEnv(workDir, mergedEnv, "sh", configureArgs...); err != nil {
		return fmt.Errorf("php/%s: configure: %w", name, err)
	}

	if err := run.RunInDir(workDir, "make"); err != nil {
		return fmt.Errorf("php/%s: make: %w", name, err)
	}

	if err := run.RunInDir(workDir, "make", "install"); err != nil {
		return fmt.Errorf("php/%s: make install: %w", name, err)
	}

	return nil
}

// buildFakePeclFromDir runs phpize + configure + make + make install in the given directory.
func buildFakePeclFromDir(ctx context.Context, name, workDir string, ec ExtensionContext, opts []string, run runner.Runner) error {
	// Prepend PHP bin dir to PATH so phpize and php-config are found by name.
	phpBinEnv := mergePHPBinPath(ec.PHPPath, nil)

	if err := run.RunInDirWithEnv(workDir, phpBinEnv, fmt.Sprintf("%s/bin/phpize", ec.PHPPath)); err != nil {
		return fmt.Errorf("php/%s: phpize: %w", name, err)
	}

	configureArgs := append([]string{"./configure"}, opts...)
	if err := run.RunInDirWithEnv(workDir, phpBinEnv, "sh", configureArgs...); err != nil {
		return fmt.Errorf("php/%s: configure: %w", name, err)
	}

	if err := run.RunInDir(workDir, "make"); err != nil {
		return fmt.Errorf("php/%s: make: %w", name, err)
	}

	if err := run.RunInDir(workDir, "make", "install"); err != nil {
		return fmt.Errorf("php/%s: make install: %w", name, err)
	}

	return nil
}

// mergePHPBinPath returns an env map with PATH prepended with phpPath/bin.
// If extra is non-nil, its entries are merged in too.
func mergePHPBinPath(phpPath string, extra map[string]string) map[string]string {
	env := map[string]string{
		"PATH": fmt.Sprintf("%s/bin:%s", phpPath, os.Getenv("PATH")),
	}
	for k, v := range extra {
		env[k] = v
	}
	return env
}
