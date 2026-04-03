package php

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
)

// PeclRecipe downloads an extension from pecl.php.net and builds it with
// phpize + ./configure + make + make install.
//
// URL: http://pecl.php.net/get/{name}-{version}.tgz
type PeclRecipe struct{}

func (p *PeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	return buildPecl(ctx, ext.Name, ext.Version, url, checksum, ec, p.configureOptions(ec), run)
}

func (p *PeclRecipe) configureOptions(ec ExtensionContext) []string {
	return []string{fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath)}
}

// AmqpPeclRecipe builds the amqp extension against the rabbitmq-c library.
type AmqpPeclRecipe struct{}

func (a *AmqpPeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		"--with-amqp",
		fmt.Sprintf("--with-librabbitmq-dir=%s", ec.RabbitMQPath),
	}
	return buildPecl(ctx, ext.Name, ext.Version, url, checksum, ec, opts, run)
}

// MaxMindRecipe builds the maxminddb extension; the work dir is maxminddb-{version}/ext.
type MaxMindRecipe struct{}

func (m *MaxMindRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	opts := []string{fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath)}
	return buildPeclInSubdir(ctx, ext.Name, ext.Version, url, checksum, fmt.Sprintf("maxminddb-%s/ext", ext.Version), ec, opts, run)
}

// RedisPeclRecipe builds the redis extension with igbinary and lzf support.
type RedisPeclRecipe struct{}

func (r *RedisPeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		"--enable-redis-igbinary",
		"--enable-redis-lzf",
		"--with-liblzf=no",
	}
	return buildPecl(ctx, ext.Name, ext.Version, url, checksum, ec, opts, run)
}

// MemcachedPeclRecipe builds the memcached extension with all optional features.
type MemcachedPeclRecipe struct{}

func (m *MemcachedPeclRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("http://pecl.php.net/get/%s-%s.tgz", ext.Name, ext.Version)
	checksum := source.Checksum{Algorithm: "md5", Value: ext.MD5}
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		"--with-libmemcached-dir",
		"--enable-memcached-sasl",
		"--enable-memcached-msgpack",
		"--enable-memcached-igbinary",
		"--enable-memcached-json",
	}
	return buildPecl(ctx, ext.Name, ext.Version, url, checksum, ec, opts, run)
}

// TidewaysXhprofRecipe builds the tideways_xhprof extension from GitHub.
type TidewaysXhprofRecipe struct{}

func (t *TidewaysXhprofRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/tideways/php-xhprof-extension/archive/v%s.tar.gz", ext.Version)
	opts := []string{fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath)}
	// GitHub archive extracts as "php-xhprof-extension-{version}", not "tideways_xhprof-{version}".
	// No MD5 available for GitHub archive downloads — checksum.Value is empty, so verification is skipped.
	return buildPeclInSubdir(ctx, ext.Name, ext.Version, url, source.Checksum{}, fmt.Sprintf("php-xhprof-extension-%s", ext.Version), ec, opts, run)
}

// PHPIRedisRecipe builds the phpiredis extension from GitHub against hiredis.
type PHPIRedisRecipe struct{}

func (p *PHPIRedisRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/nrk/phpiredis/archive/v%s.tar.gz", ext.Version)
	opts := []string{
		fmt.Sprintf("--with-php-config=%s/bin/php-config", ec.PHPPath),
		"--enable-phpiredis",
		fmt.Sprintf("--with-hiredis-dir=%s", ec.HiredisPath),
	}
	// No MD5 available for GitHub archive downloads — checksum.Value is empty, so verification is skipped.
	return buildPecl(ctx, ext.Name, ext.Version, url, source.Checksum{}, ec, opts, run)
}

// buildPecl is the shared PECL build helper: download → phpize → configure → make → make install.
func buildPecl(ctx context.Context, name, version, url string, checksum source.Checksum, ec ExtensionContext, opts []string, run runner.Runner) error {
	return buildPeclInSubdir(ctx, name, version, url, checksum, fmt.Sprintf("%s-%s", name, version), ec, opts, run)
}

// buildPeclInSubdir is like buildPecl but uses a custom subdirectory inside the extracted archive.
func buildPeclInSubdir(ctx context.Context, name, version, url string, checksum source.Checksum, subdir string, ec ExtensionContext, opts []string, run runner.Runner) error {
	archiveName := fmt.Sprintf("%s-%s.tgz", name, version)
	workDir := fmt.Sprintf("/tmp/php-ext-build/%s", subdir)

	// Download with checksum verification via Fetcher.
	// If checksum.Value is empty (GitHub archives without MD5), verification is skipped.
	if err := ec.Fetcher.Download(ctx, url, fmt.Sprintf("/tmp/%s", archiveName), checksum); err != nil {
		return fmt.Errorf("php/%s: download: %w", name, err)
	}

	// Ensure the extraction directory exists.
	if err := run.Run("mkdir", "-p", "/tmp/php-ext-build/"); err != nil {
		return fmt.Errorf("php/%s: mkdir ext-build: %w", name, err)
	}

	// Extract.
	if err := run.Run("tar", "xzf", fmt.Sprintf("/tmp/%s", archiveName), "-C", "/tmp/php-ext-build/"); err != nil {
		return fmt.Errorf("php/%s: extract: %w", name, err)
	}

	// phpize.
	if err := run.RunInDir(workDir, fmt.Sprintf("%s/bin/phpize", ec.PHPPath)); err != nil {
		return fmt.Errorf("php/%s: phpize: %w", name, err)
	}

	// configure.
	configureArgs := append([]string{"./configure"}, opts...)
	if err := run.RunInDir(workDir, "sh", configureArgs...); err != nil {
		return fmt.Errorf("php/%s: configure: %w", name, err)
	}

	// make.
	if err := run.RunInDir(workDir, "make"); err != nil {
		return fmt.Errorf("php/%s: make: %w", name, err)
	}

	// make install.
	if err := run.RunInDir(workDir, "make", "install"); err != nil {
		return fmt.Errorf("php/%s: make install: %w", name, err)
	}

	return nil
}
