package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/archive"
	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// jdkSubdir returns the actual JDK directory that the tarball extracted into.
// The bellsoft/openjdk tarballs extract to a subdirectory like "jdk8u452/"
// inside the install dir, not directly into the install dir itself.
func jdkSubdir(jdkInstallDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(jdkInstallDir, "jdk*"))
	if err != nil {
		return "", fmt.Errorf("globbing JDK subdir: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no jdk* subdirectory found in %s", jdkInstallDir)
	}
	return matches[0], nil
}

// JRubyRecipe builds JRuby by:
//  1. Downloading the JDK from the stack-specific URL (JDK 8 for JRuby 9.x,
//     JDK 21 for JRuby 10.x).
//  2. Downloading Maven (pinned version, 3.6.3 for 9.x, 3.9.9 for 10.x).
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
	"9.3":  "2.6",
	"9.4":  "3.1",
	"10.0": "3.4",
	"10.1": "4.0",
}

// jrubyToolchain is the JDK and Maven combination used to build a JRuby version.
// SHA512 values match the official Apache Maven binary distributions.
type jrubyToolchain struct {
	jdk          stack.BootstrapBinary
	jdkTarball   string
	mavenVersion string
	mavenSHA512  string
}

// jrubyToolchainFor returns the toolchain for a JRuby version. JRuby 10.x
// requires Java 21+ and Maven 3.9+; 9.x builds with JDK 8 and Maven 3.6.3.
func jrubyToolchainFor(s *stack.Stack, jrubyVersion string) (jrubyToolchain, error) {
	major, _, _ := strings.Cut(jrubyVersion, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return jrubyToolchain{}, fmt.Errorf("unexpected jruby version format %q", jrubyVersion)
	}
	if n >= 10 {
		return jrubyToolchain{
			jdk:          s.Bootstrap.JRubyJDK21,
			jdkTarball:   "openjdk-21-jdk.tar.gz",
			mavenVersion: "3.9.9",
			mavenSHA512:  "a555254d6b53d267965a3404ecb14e53c3827c09c3b94b5678835887ab404556bfaf78dcfe03ba76fa2508649dca8531c74bca4d5846513522404d48e8c4ac8b",
		}, nil
	}
	return jrubyToolchain{
		jdk:          s.Bootstrap.JRuby,
		jdkTarball:   "openjdk-8-jdk.tar.gz",
		mavenVersion: "3.6.3",
		mavenSHA512:  "c35a1803a6e70a126e80b2b3ae33eed961f83ed74d18fcd16909b2d44d7dada3203f1ffe726c17ef8dcca2dcaa9fca676987befeadc9b9f759967a8cb77181c0",
	}, nil
}

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

	tc, err := jrubyToolchainFor(s, jrubyVersion)
	if err != nil {
		return fmt.Errorf("jruby: %w", err)
	}

	// Step 2: Download and install JDK from stack-specific URL.
	jdkDir := tc.jdk.InstallDir
	jdkTar := fmt.Sprintf("%s/%s", jdkDir, tc.jdkTarball)

	if err := run.Run("mkdir", "-p", jdkDir); err != nil {
		return fmt.Errorf("jruby: creating JDK dir: %w", err)
	}

	jdkChecksum := source.Checksum{Algorithm: "sha256", Value: tc.jdk.SHA256}
	if err := j.Fetcher.Download(ctx, tc.jdk.URL, jdkTar, jdkChecksum); err != nil {
		return fmt.Errorf("jruby: downloading JDK: %w", err)
	}

	if err := run.Run("tar", "xvf", jdkTar, "-C", jdkDir); err != nil {
		return fmt.Errorf("jruby: extracting JDK: %w", err)
	}

	// The JDK tarball extracts into a subdir like "jdk8u452/" inside jdkDir.
	// JAVA_HOME must point to that subdir, not to jdkDir itself.
	javaHome, err := jdkSubdir(jdkDir)
	if err != nil {
		return fmt.Errorf("jruby: locating JDK subdir: %w", err)
	}

	// Step 3: Download and set up Maven.
	mavenURL := fmt.Sprintf("https://archive.apache.org/dist/maven/maven-3/%s/binaries/apache-maven-%s-bin.tar.gz", tc.mavenVersion, tc.mavenVersion)
	mavenTar := fmt.Sprintf("/tmp/apache-maven-%s-bin.tar.gz", tc.mavenVersion)
	mavenChecksum := source.Checksum{Algorithm: "sha512", Value: tc.mavenSHA512}

	if err := j.Fetcher.Download(ctx, mavenURL, mavenTar, mavenChecksum); err != nil {
		return fmt.Errorf("jruby: downloading Maven: %w", err)
	}

	mavenInstallDir := fmt.Sprintf("/tmp/apache-maven-%s", tc.mavenVersion)
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
	// We embed export statements directly in the shell command so that all
	// child processes (including Maven's polyglot plugin which invokes bin/jruby
	// as a subprocess) inherit JAVA_HOME, JAVACMD, and PATH.
	// JAVACMD is set explicitly so bin/jruby does not need to discover java via
	// PATH lookup — without it the subprocess fails with "Permission denied"
	// (exit 126) because JAVACMD is empty and `exec ""` fails.
	javacmd := fmt.Sprintf("%s/bin/java", javaHome)
	mvnCmd := fmt.Sprintf(
		"export JAVA_HOME=%s && export JAVACMD=%s && export PATH=%s/bin:%s/bin:/usr/bin:/bin && cd %s && mvn clean package -P '!truffle' -Djruby.default.ruby.version=%s",
		javaHome, javacmd, javaHome, mavenInstallDir, srcDir, rubyVersion,
	)
	if err := run.Run("sh", "-c", mvnCmd); err != nil {
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

func supportedJRubyPrefixes() string {
	prefixes := make([]string, 0, len(jrubyToRubyVersion))
	for p := range jrubyToRubyVersion {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return versionPrefixLess(prefixes[i], prefixes[j])
	})
	return strings.Join(prefixes, ", ")
}

// versionPrefixLess orders "major.minor" strings numerically (9.4 < 10.0).
func versionPrefixLess(a, b string) bool {
	am, an, _ := strings.Cut(a, ".")
	bm, bn, _ := strings.Cut(b, ".")
	ai, _ := strconv.Atoi(am)
	bi, _ := strconv.Atoi(bm)
	if ai != bi {
		return ai < bi
	}
	aj, _ := strconv.Atoi(an)
	bj, _ := strconv.Atoi(bn)
	return aj < bj
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
		return "", fmt.Errorf("unknown JRuby version %q — cannot determine Ruby compatibility version (supported prefixes: %s)", jrubyVersion, supportedJRubyPrefixes())
	}
	return rubyVer, nil
}
