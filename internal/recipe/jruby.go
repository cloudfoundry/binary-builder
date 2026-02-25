package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// JRubyRecipe builds JRuby by:
//  1. Downloading the JDK from the stack-specific URL.
//  2. Building Maven from source (pinned version).
//  3. Compiling JRuby via Maven.
//  4. Stripping incorrect_words.yaml from the resulting jar files.
//
// The input version is the raw JRuby version (e.g. "9.4.5.0").
// Internally the recipe computes the Ruby compatibility version and
// produces a full version of the form "9.4.5.0-ruby-3.1" which is
// used in the artifact filename.
type JRubyRecipe struct {
	Fetcher fetch.Fetcher
}

func (j *JRubyRecipe) Name() string { return "jruby" }
func (j *JRubyRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""}
}

// jrubyToRubyVersion maps a JRuby major.minor prefix to the Ruby compatibility version.
// This mapping is JRuby's own versioning scheme — not stack-specific.
var jrubyToRubyVersion = map[string]string{
	"9.3": "2.6",
	"9.4": "3.1",
}

// mavenVersion is the pinned Maven version used for the JRuby build.
// SHA512 matches the official Apache Maven 3.6.3 binary distribution.
const (
	mavenVersion = "3.6.3"
	mavenSHA512  = "c35a1803a6e70a126e80b2b3ae33eed961f83ed74d18fcd16909b2d44d7dada3203f1ffe726c17ef8dcca2dcaa9fca676987befeadc9b9f759967a8cb77181c0"
)

func (j *JRubyRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, outData *output.OutData) error {
	jrubyVersion := src.Version

	// Step 1: Determine Ruby compatibility version from JRuby version prefix.
	rubyVersion, err := jrubyRubyVersion(jrubyVersion)
	if err != nil {
		return fmt.Errorf("jruby: %w", err)
	}

	// Full version used in artifact filename: e.g. "9.4.5.0-ruby-3.1"
	fullVersion := fmt.Sprintf("%s-ruby-%s", jrubyVersion, rubyVersion)

	// Set ArtifactVersion so the artifact filename uses the full version
	// (e.g. "9.4.14.0-ruby-3.1") rather than just the raw JRuby version.
	// outData.Version (the raw "9.4.14.0") is preserved for dep-metadata and
	// builds JSON, matching Ruby builder behaviour where out_data[:version] =
	// source_input.version (never overwritten by build_jruby).
	outData.ArtifactVersion = fullVersion

	// Step 2: Download and install JDK from stack-specific URL.
	jdkDir := s.JRuby.JDKInstallDir
	jdkTar := fmt.Sprintf("%s/openjdk-8-jdk.tar.gz", jdkDir)

	if err := run.Run("mkdir", "-p", jdkDir); err != nil {
		return fmt.Errorf("jruby: creating JDK dir: %w", err)
	}

	jdkChecksum := source.Checksum{Algorithm: "sha256", Value: s.JRuby.JDKSHA256}
	if err := j.Fetcher.Download(ctx, s.JRuby.JDKURL, jdkTar, jdkChecksum); err != nil {
		return fmt.Errorf("jruby: downloading JDK: %w", err)
	}

	if err := run.Run("tar", "xvf", jdkTar, "-C", jdkDir); err != nil {
		return fmt.Errorf("jruby: extracting JDK: %w", err)
	}

	// Step 3: Download and set up Maven.
	mavenURL := fmt.Sprintf("https://archive.apache.org/dist/maven/maven-3/%s/binaries/apache-maven-%s-bin.tar.gz", mavenVersion, mavenVersion)
	mavenTar := fmt.Sprintf("/tmp/apache-maven-%s-bin.tar.gz", mavenVersion)
	mavenChecksum := source.Checksum{Algorithm: "sha512", Value: mavenSHA512}

	if err := j.Fetcher.Download(ctx, mavenURL, mavenTar, mavenChecksum); err != nil {
		return fmt.Errorf("jruby: downloading Maven: %w", err)
	}

	mavenInstallDir := fmt.Sprintf("/tmp/apache-maven-%s", mavenVersion)
	if err := run.Run("tar", "xf", mavenTar, "-C", "/tmp"); err != nil {
		return fmt.Errorf("jruby: extracting Maven: %w", err)
	}

	// Step 4: Download JRuby source zip.
	//
	// NOTE: The Ruby builder (builder.rb build_jruby) historically had a bug where it
	// passed @source_input.sha256 into the sha512 slot of the inner SourceInput, and
	// @source_input.git_commit_sha (always empty) into the sha256 slot. This caused the
	// inner binary-builder.rb to be invoked with --sha256= (empty), failing verification.
	// Fixed in builder.rb by passing nil for sha512 and @source_input.sha256 for sha256.
	//
	// The Go builder uses PrimaryChecksum() which correctly prefers sha512 > sha256,
	// so data.json should supply both sha256 and sha512 fields.
	jrubyURL := fmt.Sprintf("https://repo1.maven.org/maven2/org/jruby/jruby-dist/%s/jruby-dist-%s-src.zip", jrubyVersion, jrubyVersion)
	jrubySrcZip := fmt.Sprintf("/tmp/jruby-dist-%s-src.zip", jrubyVersion)
	jrubyChecksum := src.PrimaryChecksum()

	if err := j.Fetcher.Download(ctx, jrubyURL, jrubySrcZip, jrubyChecksum); err != nil {
		return fmt.Errorf("jruby: downloading source: %w", err)
	}

	// Extract source zip.
	if err := run.Run("unzip", "-o", jrubySrcZip, "-d", "/tmp"); err != nil {
		return fmt.Errorf("jruby: extracting source: %w", err)
	}

	srcDir := fmt.Sprintf("/tmp/jruby-%s", jrubyVersion)

	// Step 5: Compile JRuby via Maven inside srcDir.
	// JAVA_HOME and PATH are set so Maven and the JDK are on the PATH.
	// We use sh -c "cd {srcDir} && mvn ..." because RunWithEnv does not
	// accept a working directory argument.
	mvnCmd := fmt.Sprintf(
		"cd %s && mvn clean package -P '!truffle' -Djruby.default.ruby.version=%s",
		srcDir, rubyVersion,
	)
	buildEnv := map[string]string{
		"JAVA_HOME": jdkDir,
		"PATH":      fmt.Sprintf("%s/bin:%s/bin:/usr/bin:/bin", jdkDir, mavenInstallDir),
	}
	if err := run.RunWithEnv(buildEnv, "sh", "-c", mvnCmd); err != nil {
		return fmt.Errorf("jruby: mvn build: %w", err)
	}

	// Step 6: Pack artifact — mirror Ruby's compress! exactly.
	// compress! creates a tmpdir, cp -r's bin/ and lib/ into it, writes sources.yml
	// into the tmpdir root, then runs: ls -A tmpdir | xargs tar czf archive -C tmpdir
	// which produces ./bin/... ./lib/... ./sources.yml paths with the ./ prefix.
	// We reproduce this by creating a tmpdir, copying bin/ and lib/, writing
	// sources.yml, and tarring from there.
	jrubySrcSHA256, err := fileSHA256(jrubySrcZip)
	if err != nil {
		return fmt.Errorf("jruby: computing source SHA256: %w", err)
	}
	sourcesContent := buildSourcesYAML([]SourceEntry{{URL: jrubyURL, SHA256: jrubySrcSHA256}})

	packDir, err := os.MkdirTemp("", "jruby-pack-*")
	if err != nil {
		return fmt.Errorf("jruby: creating pack tmpdir: %w", err)
	}
	defer os.RemoveAll(packDir)

	if err := run.Run("cp", "-r", filepath.Join(srcDir, "bin"), filepath.Join(srcDir, "lib"), packDir); err != nil {
		return fmt.Errorf("jruby: copying bin/lib to pack dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "sources.yml"), sourcesContent, 0644); err != nil {
		return fmt.Errorf("jruby: writing sources.yml: %w", err)
	}

	artifactPath := filepath.Join(mustCwd(), fmt.Sprintf("jruby-%s-linux-x64.tgz", fullVersion))

	// Use `tar czf ... -C packDir .` — the trailing `.` argument makes tar emit
	// a root `./` entry and `./`-prefixed paths for all members, exactly matching
	// what Ruby's Archive.strip_incorrect_words_yaml_from_tar re-archives with
	// `tar -C dir -czf filename .`.
	if err := run.Run("tar", "czf", artifactPath, "-C", packDir, "."); err != nil {
		return fmt.Errorf("jruby: packing artifact: %w", err)
	}

	// Step 7: Strip incorrect_words.yaml from jars in the artifact.
	if err := archive.StripIncorrectWordsYAML(artifactPath); err != nil {
		return fmt.Errorf("jruby: stripping incorrect_words.yaml: %w", err)
	}

	return nil
}

// jrubyRubyVersion maps a JRuby version string to the Ruby compatibility version.
func jrubyRubyVersion(jrubyVersion string) (string, error) {
	// Extract major.minor prefix (first two parts).
	parts := strings.Split(jrubyVersion, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected jruby version format %q", jrubyVersion)
	}
	prefix := strings.Join(parts[:2], ".")

	rubyVer, ok := jrubyToRubyVersion[prefix]
	if !ok {
		return "", fmt.Errorf("unknown JRuby version %q — cannot determine Ruby compatibility version (supported prefixes: 9.3, 9.4)", jrubyVersion)
	}
	return rubyVer, nil
}
