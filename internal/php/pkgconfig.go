package php

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
)

// HiredisRecipe downloads and builds the hiredis C library from GitHub.
// Uses LIBRARY_PATH=lib PREFIX={path} make install (no autoconf configure).
type HiredisRecipe struct{}

func (h *HiredisRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/redis/hiredis/archive/v%s.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("hiredis-%s.tar.gz", ext.Version)
	dest := fmt.Sprintf("/tmp/%s", archiveName)
	srcDir := fmt.Sprintf("/tmp/hiredis-%s", ext.Version)
	installPath := fmt.Sprintf("/tmp/hiredis-install-%s", ext.Version)

	if err := ec.Fetcher.Download(ctx, url, dest, source.Checksum{}); err != nil {
		return fmt.Errorf("php/hiredis: download: %w", err)
	}
	if err := run.Run("tar", "xzf", dest, "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php/hiredis: extract: %w", err)
	}
	installCmd := fmt.Sprintf("LIBRARY_PATH=lib PREFIX='%s' make install", installPath)
	if err := run.RunInDir(srcDir, "bash", "-c", installCmd); err != nil {
		return fmt.Errorf("php/hiredis: make install: %w", err)
	}
	// Expose install path via ec (caller sets ec.HiredisPath = installPath).
	_ = installPath
	return nil
}

// RabbitMQRecipe downloads and builds rabbitmq-c from GitHub using cmake.
type RabbitMQRecipe struct{}

func (r *RabbitMQRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/alanxz/rabbitmq-c/archive/v%s.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("rabbitmq-%s.tar.gz", ext.Version)
	dest := fmt.Sprintf("/tmp/%s", archiveName)
	srcDir := fmt.Sprintf("/tmp/rabbitmq-c-%s", ext.Version)

	if err := ec.Fetcher.Download(ctx, url, dest, source.Checksum{}); err != nil {
		return fmt.Errorf("php/rabbitmq: download: %w", err)
	}
	if err := run.Run("tar", "xzf", dest, "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php/rabbitmq: extract: %w", err)
	}
	for _, step := range [][]string{
		{"cmake", "."},
		{"cmake", "--build", "."},
		{"cmake", "-DCMAKE_INSTALL_PREFIX=/usr/local", "."},
		{"cmake", "--build", ".", "--target", "install"},
	} {
		if err := run.RunInDir(srcDir, step[0], step[1:]...); err != nil {
			return fmt.Errorf("php/rabbitmq: cmake step %v: %w", step, err)
		}
	}
	return nil
}

// LibRdKafkaRecipe downloads and builds librdkafka from GitHub.
// Uses ./configure --prefix=/usr then make + make install.
type LibRdKafkaRecipe struct{}

func (l *LibRdKafkaRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/edenhill/librdkafka/archive/v%s.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("librdkafka-%s.tar.gz", ext.Version)
	dest := fmt.Sprintf("/tmp/%s", archiveName)
	srcDir := fmt.Sprintf("/tmp/librdkafka-%s", ext.Version)

	if err := ec.Fetcher.Download(ctx, url, dest, source.Checksum{}); err != nil {
		return fmt.Errorf("php/librdkafka: download: %w", err)
	}
	if err := run.Run("tar", "xzf", dest, "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php/librdkafka: extract: %w", err)
	}
	if err := run.RunInDir(srcDir, "bash", "./configure", "--prefix=/usr"); err != nil {
		return fmt.Errorf("php/librdkafka: configure: %w", err)
	}
	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("php/librdkafka: make: %w", err)
	}
	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("php/librdkafka: make install: %w", err)
	}
	return nil
}

// LibSodiumRecipe downloads and builds libsodium from GitHub.
// Uses ./configure + make + make install (standard autoconf).
type LibSodiumRecipe struct{}

func (l *LibSodiumRecipe) Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error {
	url := fmt.Sprintf("https://github.com/jedisct1/libsodium/archive/%s-RELEASE.tar.gz", ext.Version)
	archiveName := fmt.Sprintf("libsodium-%s.tar.gz", ext.Version)
	dest := fmt.Sprintf("/tmp/%s", archiveName)
	srcDir := fmt.Sprintf("/tmp/libsodium-%s-RELEASE", ext.Version)
	installPath := fmt.Sprintf("/tmp/libsodium-install-%s", ext.Version)

	if err := ec.Fetcher.Download(ctx, url, dest, source.Checksum{}); err != nil {
		return fmt.Errorf("php/libsodium: download: %w", err)
	}
	if err := run.Run("tar", "xzf", dest, "-C", "/tmp/"); err != nil {
		return fmt.Errorf("php/libsodium: extract: %w", err)
	}
	if err := run.RunInDir(srcDir, "sh", "./configure", fmt.Sprintf("--prefix=%s", installPath)); err != nil {
		return fmt.Errorf("php/libsodium: configure: %w", err)
	}
	if err := run.RunInDir(srcDir, "make"); err != nil {
		return fmt.Errorf("php/libsodium: make: %w", err)
	}
	if err := run.RunInDir(srcDir, "make", "install"); err != nil {
		return fmt.Errorf("php/libsodium: make install: %w", err)
	}
	return nil
}
