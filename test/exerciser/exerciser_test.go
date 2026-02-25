//go:build integration

// Package exerciser_test verifies that built artifacts are functional.
//
// Each test extracts a tarball inside a Docker container running the target stack
// and asserts that the binary self-reports the expected version string (or that
// expected files are present for library/tool artifacts).
//
// Usage:
//
//	ARTIFACT=/tmp/ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz \
//	STACK=cflinuxfs4 \
//	go test -tags integration ./test/exerciser/ -run TestRubyBinary -v
//
// The ARTIFACT and STACK environment variables must be set. If either is absent
// the test is skipped (not failed), so the harness is safe to import in CI
// pipelines that gate on the integration build tag.
package exerciser_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runInContainer extracts ARTIFACT inside a Docker container running IMAGE and
// executes cmdArgs. It returns the combined stdout+stderr output.
func runInContainer(t *testing.T, tarball, stack string, cmdArgs ...string) string {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	runScript := filepath.Join(filepath.Dir(thisFile), "run.sh")

	args := append([]string{tarball, stack}, cmdArgs...)
	cmd := exec.Command(runScript, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("exerciser command failed: %v\noutput:\n%s", err, out)
	}
	return string(out)
}

// artifact returns the ARTIFACT env var or skips the test.
func artifact(t *testing.T) string {
	t.Helper()
	a := os.Getenv("ARTIFACT")
	if a == "" {
		t.Skip("ARTIFACT env var not set")
	}
	return a
}

// stackEnv returns the STACK env var or skips the test.
func stackEnv(t *testing.T) string {
	t.Helper()
	s := os.Getenv("STACK")
	if s == "" {
		t.Skip("STACK env var not set")
	}
	return s
}

// assertContains fails the test if output does not contain want.
func assertContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Errorf("expected output to contain %q\ngot:\n%s", want, output)
	}
}

// ── Compiled deps ──────────────────────────────────────────────────────────

func TestRubyBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/ruby", "-e", "puts RUBY_VERSION")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestPythonBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/python3", "--version")
	assertContains(t, out, "Python "+os.Getenv("VERSION"))
}

func TestNodeBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"node-v*/bin/node -e 'console.log(process.version)'")
	assertContains(t, out, "v"+os.Getenv("VERSION"))
}

func TestGoBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./go/bin/go", "version")
	assertContains(t, out, "go"+os.Getenv("VERSION"))
}

func TestNginxBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"env LD_LIBRARY_PATH=./lib ./nginx/sbin/nginx -v 2>&1")
	assertContains(t, out, "nginx/"+os.Getenv("VERSION"))
}

func TestNginxStaticBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./nginx/sbin/nginx", "-v")
	assertContains(t, out, "nginx/"+os.Getenv("VERSION"))
}

func TestOpenrestyBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./nginx/sbin/nginx", "-v")
	assertContains(t, out, "openresty/"+os.Getenv("VERSION"))
}

func TestHTTPDBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"env LD_LIBRARY_PATH=./lib ./httpd/bin/httpd -v 2>&1")
	assertContains(t, out, "Apache/"+os.Getenv("VERSION"))
}

func TestJRubyBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/jruby", "--version")
	// JRuby version is the prefix of the full artifact version (e.g. "9.4.5.0")
	version := os.Getenv("VERSION")
	if idx := strings.Index(version, "-ruby-"); idx != -1 {
		version = version[:idx]
	}
	assertContains(t, out, version)
}

func TestBundlerBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/bundle", "--version")
	assertContains(t, out, "Bundler version "+os.Getenv("VERSION"))
}

func TestRBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/R", "--version")
	assertContains(t, out, "R version "+os.Getenv("VERSION"))
}

func TestLibunwindFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls lib/libunwind.so*")
	if strings.TrimSpace(out) == "" {
		t.Error("expected lib/libunwind.so* to exist but found nothing")
	}
}

func TestLibgdiplusFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls lib/libgdiplus.so*")
	if strings.TrimSpace(out) == "" {
		t.Error("expected lib/libgdiplus.so* to exist but found nothing")
	}
}

func TestDepBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./dep", "version")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestGlideBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./glide", "--version")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestGodepBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./godep", "version")
	assertContains(t, out, "v"+os.Getenv("VERSION"))
}

func TestHWCBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "file", "hwc.exe")
	assertContains(t, out, "PE32+")
}

// ── Repack / simple deps ───────────────────────────────────────────────────

func TestPipBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/pip", "--version")
	assertContains(t, out, "pip "+os.Getenv("VERSION"))
}

func TestPipenvBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/pipenv", "--version")
	assertContains(t, out, "pipenv, version "+os.Getenv("VERSION"))
}

func TestSetuptoolsFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls setuptools-*.dist-info/")
	if strings.TrimSpace(out) == "" {
		t.Error("expected setuptools-*.dist-info/ directory to exist but found nothing")
	}
}

func TestYarnBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./bin/yarn", "--version")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestRubygemsFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls rubygems-*/")
	if strings.TrimSpace(out) == "" {
		t.Error("expected rubygems-*/ directory to exist but found nothing")
	}
}

func TestDotnetSDKBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./dotnet", "--version")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestDotnetRuntimeBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "./dotnet", "--version")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestDotnetAspnetcoreFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"ls shared/Microsoft.AspNetCore.App/")
	if strings.TrimSpace(out) == "" {
		t.Error("expected shared/Microsoft.AspNetCore.App/ to exist but found nothing")
	}
}

// ── Passthrough deps ───────────────────────────────────────────────────────

func TestComposerBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"php composer.phar --version 2>&1 || echo 'php not available, checking file exists'")
	// Composer is a phar — check it's present if php is not available
	if !strings.Contains(out, "Composer version "+os.Getenv("VERSION")) {
		out2 := runInContainer(t, a, s, "bash", "-c", "ls composer.phar")
		if strings.TrimSpace(out2) == "" {
			t.Error("composer.phar not found in artifact")
		}
	}
}

func TestTomcatFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls bin/catalina.sh")
	if strings.TrimSpace(out) == "" {
		t.Error("expected bin/catalina.sh to exist but found nothing")
	}
}

func TestOpenJDKBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "./bin/java -version 2>&1")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestZuluBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "./bin/java -version 2>&1")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestSAPMachineBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "./bin/java -version 2>&1")
	assertContains(t, out, os.Getenv("VERSION"))
}

func TestJProfilerFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls bin/jprofiler")
	if strings.TrimSpace(out) == "" {
		t.Error("expected bin/jprofiler to exist but found nothing")
	}
}

func TestYourKitFiles(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c", "ls lib/yjp.jar")
	if strings.TrimSpace(out) == "" {
		t.Error("expected lib/yjp.jar to exist but found nothing")
	}
}

// ── PHP — extended assertions ──────────────────────────────────────────────

func TestPHPBinary(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"export LD_LIBRARY_PATH=$PWD/php/lib && ./php/bin/php --version 2>&1")
	assertContains(t, out, "PHP "+os.Getenv("VERSION"))
}

func TestPHPNativeModules(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"export LD_LIBRARY_PATH=$PWD/php/lib && ./php/bin/php -m 2>&1")
	for _, mod := range []string{"date", "json", "pcre", "Reflection", "SPL", "standard"} {
		if !strings.Contains(out, mod) {
			t.Errorf("expected PHP native module %q to be present\noutput:\n%s", mod, out)
		}
	}
}

func TestPHPKeyExtensions(t *testing.T) {
	a, s := artifact(t), stackEnv(t)
	out := runInContainer(t, a, s, "bash", "-c",
		"export LD_LIBRARY_PATH=$PWD/php/lib && ./php/bin/php -m 2>&1")
	for _, ext := range []string{"curl", "gd", "mbstring", "mysqli", "pdo_mysql"} {
		if !strings.Contains(out, ext) {
			t.Errorf("expected PHP extension %q to be present\noutput:\n%s", ext, out)
		}
	}
}
