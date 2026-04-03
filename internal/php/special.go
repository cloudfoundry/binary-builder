package php

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
)

// IonCubeRecipe downloads a pre-built ioncube loader binary (no compile step).
// The loader is placed at {ioncubePath}/ioncube/ioncube_loader_lin_{major}.so
// and later copied to {ztsPath}/ioncube.so by PHPRecipe.setup_tar.
type IonCubeRecipe struct{}

func (i *IonCubeRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_x86-64_%s.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("ioncube-%s.tar.gz", ext.Version)
	installPath := fmt.Sprintf("/tmp/ioncube-%s", ext.Version)

	// IonCube provides no checksum — skip verification (checksum.Value == "").
	if err := ec.Fetcher.Download(ctx, url, fmt.Sprintf("/tmp/%s", archiveName), source.Checksum{}); err != nil {
		return fmt.Errorf("php/ioncube: download: %w", err)
	}
	if err := run.Run("mkdir", "-p", installPath); err != nil {
		return fmt.Errorf("php/ioncube: mkdir: %w", err)
	}
	if err := run.Run("tar", "xzf", fmt.Sprintf("/tmp/%s", archiveName), "-C", installPath); err != nil {
		return fmt.Errorf("php/ioncube: extract: %w", err)
	}
	return nil
}

// OraclePeclRecipe builds the oci8 PECL extension against Oracle Instant Client.
// Requires /oracle to be mounted with the Oracle SDK.
type OraclePeclRecipe struct{}

func (o *OraclePeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		"--with-oci8=shared,instantclient,/oracle",
	}
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	if err := buildPecl(ctx, ext.Name, ext.Version, url, checksum, ec, opts, run); err != nil {
		return err
	}
	// Copy Oracle libs into PHP prefix.
	return run.Run("sh", "-c", fmt.Sprintf(`
cp -an /oracle/libclntshcore.so.12.1 %s/lib
cp -an /oracle/libclntsh.so %s/lib
cp -an /oracle/libclntsh.so.12.1 %s/lib
cp -an /oracle/libipc1.so %s/lib
cp -an /oracle/libmql1.so %s/lib
cp -an /oracle/libnnz12.so %s/lib
cp -an /oracle/libociicus.so %s/lib
cp -an /oracle/libons.so %s/lib
`, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath))
}

// OraclePdoRecipe builds the pdo_oci extension (FakePecl) against Oracle Instant Client.
// Detects the Oracle version from /oracle/libclntsh.so.*.
type OraclePdoRecipe struct{}

func (o *OraclePdoRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	// Detect oracle version from the libclntsh.so.{version} symlink.
	oracleVersion, err := detectOracleVersion(run)
	if err != nil {
		return fmt.Errorf("php/pdo_oci: detect oracle version: %w", err)
	}

	opts := []string{
		fmt.Sprintf("--with-pdo-oci=shared,instantclient,/oracle,%s", oracleVersion),
	}
	if err := buildFakePecl(ctx, "pdo_oci", ec, opts, run); err != nil {
		return err
	}
	// Copy Oracle libs into PHP prefix.
	return run.Run("sh", "-c", fmt.Sprintf(`
cp -an /oracle/libclntshcore.so.12.1 %s/lib
cp -an /oracle/libclntsh.so %s/lib
cp -an /oracle/libclntsh.so.12.1 %s/lib
cp -an /oracle/libipc1.so %s/lib
cp -an /oracle/libmql1.so %s/lib
cp -an /oracle/libnnz12.so %s/lib
cp -an /oracle/libociicus.so %s/lib
cp -an /oracle/libons.so %s/lib
`, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath, ec.PHPPath))
}

// detectOracleVersion returns the version suffix from the first libclntsh.so.{version} file found.
func detectOracleVersion(run runner.Runner) (string, error) {
	out, err := run.Output("sh", "-c", `ls /oracle/libclntsh.so.* 2>/dev/null | head -1 | sed 's|.*libclntsh\.so\.||'`)
	if err != nil {
		return "", fmt.Errorf("listing oracle libs: %w", err)
	}
	return out, nil
}
