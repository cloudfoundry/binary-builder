package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/apt"
	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/php"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// phpConfigureFlags are the PHP core ./configure flags (not stack-specific).
var phpConfigureFlags = []string{
	"--disable-static",
	"--enable-shared",
	"--enable-ftp=shared",
	"--enable-sockets=shared",
	"--enable-soap=shared",
	"--enable-fileinfo=shared",
	"--enable-bcmath",
	"--enable-calendar",
	"--enable-intl",
	"--with-kerberos",
	"--with-bz2=shared",
	"--with-curl=shared",
	"--enable-dba=shared",
	"--with-password-argon2=/usr/lib/x86_64-linux-gnu",
	"--with-cdb",
	"--with-gdbm",
	"--with-mysqli=shared",
	"--enable-pdo=shared",
	"--with-pdo-sqlite=shared,/usr",
	"--with-pdo-mysql=shared,mysqlnd",
	"--with-pdo-pgsql=shared",
	"--with-pgsql=shared",
	"--with-pspell=shared",
	"--with-gettext=shared",
	"--with-gmp=shared",
	"--with-imap=shared",
	"--with-imap-ssl=shared",
	"--with-ldap=shared",
	"--with-ldap-sasl",
	"--with-zlib=shared",
	"--with-xsl=shared",
	"--with-snmp=shared",
	"--enable-mbstring=shared",
	"--enable-mbregex",
	"--enable-exif=shared",
	"--with-openssl=shared",
	"--enable-fpm",
	"--enable-pcntl=shared",
	"--enable-sysvsem=shared",
	"--enable-sysvshm=shared",
	"--enable-sysvmsg=shared",
	"--enable-shmop=shared",
}

// PHPRecipe builds PHP and all of its extensions.
// Extension data (which PECL extensions and native modules to build) is embedded
// directly in the internal/php package — see internal/php/extensions.go and the
// YAML files alongside it.
type PHPRecipe struct {
	Fetcher fetch.Fetcher
}

func (p *PHPRecipe) Name() string { return "php" }
func (p *PHPRecipe) Artifact() ArtifactMeta {
	// Stack is appended at runtime from src.Stack.
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

func (p *PHPRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, outData *output.OutData) error {
	// Parse version: "8.3.2" → major="8", minor="3"
	parts := strings.SplitN(src.Version, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("php: invalid version %q", src.Version)
	}
	phpMajor := parts[0]
	phpMinor := parts[1]

	// Load extension set (base + patch) from embedded YAML data in internal/php/.
	extSet, err := php.Load(phpMajor, phpMinor)
	if err != nil {
		return fmt.Errorf("php: loading extensions: %w", err)
	}

	// Step 1: apt install php_build packages.
	a := apt.New(run)
	if err := a.Install(ctx, s.AptPackages["php_build"]...); err != nil {
		return fmt.Errorf("php: apt install php_build: %w", err)
	}

	// Step 2: create symlinks from stack config.
	for _, sym := range s.PHPSymlinks {
		if err := run.Run("ln", "-sf", sym.Src, sym.Dst); err != nil {
			return fmt.Errorf("php: symlink %s → %s: %w", sym.Src, sym.Dst, err)
		}
	}

	// Step 3: download and extract PHP source.
	// Use the URL and checksum provided by go-depwatcher (src.URL / src.PrimaryChecksum).
	phpTarball := fmt.Sprintf("/tmp/php-%s.tar.gz", src.Version)
	if err := p.Fetcher.Download(ctx, src.URL, phpTarball, src.PrimaryChecksum()); err != nil {
		return fmt.Errorf("php: download source: %w", err)
	}
	if err := run.Run("tar", "xzf", phpTarball, "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php: extract source: %w", err)
	}

	phpSrcDir := fmt.Sprintf("/tmp/php-%s", src.Version)
	phpInstallPath := fmt.Sprintf("/app/vendor/php-%s", src.Version)

	// Step 4: configure + make + make install PHP core.
	configureArgs := append([]string{fmt.Sprintf("--prefix=%s", phpInstallPath)}, phpConfigureFlags...)
	configureCmd := "LIBS=-lz ./configure " + strings.Join(configureArgs, " ")
	if err := run.RunInDir(phpSrcDir, "bash", "-c", configureCmd); err != nil {
		return fmt.Errorf("php: configure: %w", err)
	}
	if err := run.RunInDir(phpSrcDir, "make"); err != nil {
		return fmt.Errorf("php: make: %w", err)
	}
	if err := run.RunInDir(phpSrcDir, "make", "install"); err != nil {
		return fmt.Errorf("php: make install: %w", err)
	}

	ec := php.ExtensionContext{
		PHPPath:      phpInstallPath,
		PHPSourceDir: phpSrcDir,
		PHPMajor:     phpMajor,
		PHPMinor:     phpMinor,
		Fetcher:      p.Fetcher,
	}

	// Step 5: build native modules (in order — some later extensions depend on them).
	for _, mod := range extSet.NativeModules {
		recipe, err := php.RecipeFor(mod.Klass)
		if err != nil {
			return fmt.Errorf("php: native module %s: %w", mod.Name, err)
		}
		if err := recipe.Build(ctx, mod, ec, run); err != nil {
			return fmt.Errorf("php: native module %s: %w", mod.Name, err)
		}
		// Wire up paths that later extensions depend on.
		switch mod.Name {
		case "hiredis":
			ec.HiredisPath = fmt.Sprintf("/tmp/hiredis-install-%s", mod.Version)
		case "libsodium":
			ec.LibSodiumPath = fmt.Sprintf("/tmp/libsodium-install-%s", mod.Version)
		case "lua":
			ec.LuaPath = fmt.Sprintf("/tmp/lua-install-%s", mod.Version)
		case "rabbitmq":
			// cmake installs headers to /usr/local/include and libs to
			// /usr/local/lib/x86_64-linux-gnu; pass /usr/local so that
			// --with-librabbitmq-dir finds amqp.h and librabbitmq.so.
			ec.RabbitMQPath = "/usr/local"
		}
	}

	// Step 6: build PHP extensions.
	// Skip oracle extensions (oci8, pdo_oci) when /oracle is not mounted —
	// matches Ruby's should_cook? check (OraclePeclRecipe.oracle_sdk?).
	_, oracleErr := os.Stat("/oracle")
	oraclePresent := oracleErr == nil
	for _, ext := range extSet.Extensions {
		if (ext.Name == "oci8" || ext.Name == "pdo_oci") && !oraclePresent {
			continue
		}
		recipe, err := php.RecipeFor(ext.Klass)
		if err != nil {
			return fmt.Errorf("php: extension %s: %w", ext.Name, err)
		}
		if err := recipe.Build(ctx, ext, ec, run); err != nil {
			return fmt.Errorf("php: extension %s: %w", ext.Name, err)
		}
		// Wire up ioncube path after it's downloaded (ioncube is an extension, not a native module).
		if ext.Name == "ioncube" {
			ec.IonCubePath = fmt.Sprintf("/tmp/ioncube-%s", ext.Version)
		}
	}

	// Step 7: setup_tar — copy bundled shared libs into PHP prefix.
	if err := p.setupTar(ec, run); err != nil {
		return fmt.Errorf("php: setup_tar: %w", err)
	}

	// Step 8: populate sub-dependencies.
	outData.SubDependencies = make(map[string]output.SubDependency)
	for _, ext := range append(extSet.NativeModules, extSet.Extensions...) {
		outData.SubDependencies[ext.Name] = output.SubDependency{Version: ext.Version}
	}

	// Step 9: pack artifact.
	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("php-%s-linux-x64-%s.tgz", src.Version, s.Name))
	if err := run.Run("tar", "czf", artifactPath, "-C", "/app/vendor", fmt.Sprintf("php-%s", src.Version)); err != nil {
		return fmt.Errorf("php: packing artifact: %w", err)
	}
	// Strip the top-level directory (php-{version}/) from the artifact so paths
	// start with "./" — matching Ruby's Archive.strip_top_level_directory_from_tar.
	if err := archive.StripTopLevelDir(artifactPath); err != nil {
		return fmt.Errorf("php: strip top-level dir: %w", err)
	}

	return nil
}

// setupTar copies bundled shared libraries into the PHP install prefix.
func (p *PHPRecipe) setupTar(ec php.ExtensionContext, run runner.Runner) error {
	phpPath := ec.PHPPath
	libDir := "/usr/lib/x86_64-linux-gnu"

	copies := []string{
		fmt.Sprintf("cp -a /usr/local/lib/x86_64-linux-gnu/librabbitmq.so* %s/lib/", phpPath),
		fmt.Sprintf("cp -a /usr/lib/libc-client.so* %s/lib/", phpPath),
		fmt.Sprintf("cp -a %s/libmcrypt.so* %s/lib", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libaspell.so* %s/lib", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libpspell.so* %s/lib", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libmemcached.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libuv.so* %s/lib", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libargon2.so* %s/lib", libDir, phpPath),
		fmt.Sprintf("cp -a /usr/lib/librdkafka.so* %s/lib/", phpPath),
		fmt.Sprintf("cp -a %s/libzip.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libGeoIP.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libgpgme.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libassuan.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libgpg-error.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libtidy*.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libenchant*.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libfbclient.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/librecode.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libtommath.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libmaxminddb.so* %s/lib/", libDir, phpPath),
		fmt.Sprintf("cp -a %s/libssh2.so* %s/lib/", libDir, phpPath),
	}

	// Hiredis libs (only if hiredis was built).
	if ec.HiredisPath != "" {
		copies = append(copies, fmt.Sprintf("cp -a %s/lib/libhiredis.so* %s/lib/", ec.HiredisPath, phpPath))
	}

	// IonCube loader (copy to PHP extensions zts dir).
	// Ruby uses major_version = version.match(/^(\d+\.\d+)/)[1], e.g. "8.1" for PHP 8.1.32.
	// The ioncube archive contains ioncube_loader_lin_8.1.so (major.minor), not ioncube_loader_lin_8.so.
	if ec.IonCubePath != "" {
		phpMajorMinor := ec.PHPMajor + "." + ec.PHPMinor
		copies = append(copies, fmt.Sprintf(
			"cp %s/ioncube/ioncube_loader_lin_%s.so $(find %s/lib/php/extensions -name 'no-debug-non-zts-*' -type d | head -1)/ioncube.so",
			ec.IonCubePath, phpMajorMinor, phpPath,
		))
	}

	// Run all copy commands.
	for _, cmd := range copies {
		if err := run.Run("sh", "-c", cmd); err != nil {
			return fmt.Errorf("setup_tar: %s: %w", cmd, err)
		}
	}

	// Fix any absolute symlinks in lib/ produced by `cp -a` from build hosts
	// where system library symlinks are absolute (e.g. libgpg-error.so →
	// /lib/x86_64-linux-gnu/libgpg-error.so.0 on cflinuxfs4). libbuildpack
	// rejects absolute symlinks as a security measure, causing staging to fail
	// with "cannot link to an absolute path when extracting archives".
	fixSymlinks := fmt.Sprintf(
		`find "%s/lib" -maxdepth 1 -type l | while IFS= read -r link; do
			target=$(readlink "$link")
			if [ "${target#/}" != "$target" ]; then
				ln -sf "$(basename "$target")" "$link"
			fi
		done`,
		phpPath,
	)
	if err := run.Run("sh", "-c", fixSymlinks); err != nil {
		return fmt.Errorf("setup_tar: fix absolute symlinks: %w", err)
	}

	// Cleanup.
	cleanup := fmt.Sprintf(
		`rm -f "%s/etc/php-fpm.conf.default" && rm -f "%s/bin/php-cgi" && find "%s/lib/php/extensions" -name "*.a" -type f -delete`,
		phpPath, phpPath, phpPath,
	)
	if err := run.Run("sh", "-c", cleanup); err != nil {
		return fmt.Errorf("setup_tar: cleanup: %w", err)
	}

	return nil
}
