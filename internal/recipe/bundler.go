package recipe

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// BundlerRecipe builds bundler by bootstrapping a pre-built Ruby, then running
// `gem install bundler -v {version} --no-document --env-shebang`, tarring the
// result, and replacing shebangs — matching the Ruby builder's behavior exactly.
type BundlerRecipe struct {
	Fetcher fetch.Fetcher
}

func (b *BundlerRecipe) Name() string { return "bundler" }
func (b *BundlerRecipe) Artifact() ArtifactMeta {
	return ArtifactMeta{OS: "linux", Arch: "noarch", Stack: ""}
}

func (b *BundlerRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input, run runner.Runner, _ *output.OutData) error {
	bootstrap := s.Bootstrap.Ruby
	rubyTarball := filepath.Join("/tmp", filepath.Base(bootstrap.URL))

	// Download the pre-built Ruby bootstrap binary with SHA256 verification.
	if err := b.Fetcher.Download(ctx, bootstrap.URL, rubyTarball, source.Checksum{Algorithm: "sha256", Value: bootstrap.SHA256}); err != nil {
		return fmt.Errorf("bundler: downloading ruby bootstrap: %w", err)
	}
	defer os.Remove(rubyTarball)

	// Extract Ruby bootstrap to its install dir.
	if err := run.Run("mkdir", "-p", bootstrap.InstallDir); err != nil {
		return err
	}
	if err := run.Run("tar", "xzf", rubyTarball, "-C", bootstrap.InstallDir, "--strip-components=1"); err != nil {
		return fmt.Errorf("bundler: extracting ruby bootstrap: %w", err)
	}

	// Use the full path to gem to avoid PATH resolution issues.
	gemBin := filepath.Join(bootstrap.InstallDir, "bin", "gem")

	// Create a tmpdir to act as GEM_HOME/GEM_PATH — mirrors the Ruby recipe's
	// `in_gem_env` block which sets these to a temp directory so the gem
	// installs there without touching system gems.
	gemHome, err := os.MkdirTemp("", "bundler-gemhome-*")
	if err != nil {
		return fmt.Errorf("bundler: creating gem tmpdir: %w", err)
	}
	defer os.RemoveAll(gemHome)

	// gem install bundler --version X --no-document --env-shebang
	// GEM_HOME and GEM_PATH point to gemHome so all files land there.
	if err := run.RunWithEnv(
		map[string]string{
			"GEM_HOME": gemHome,
			"GEM_PATH": gemHome,
			"RUBYOPT":  "",
		},
		gemBin, "install", "bundler", "--version", src.Version, "--no-document", "--env-shebang",
	); err != nil {
		return fmt.Errorf("bundler: gem install: %w", err)
	}

	// Replace shebangs in bin/ scripts: #!/path/to/ruby → #!/usr/bin/env ruby
	// Mirrors the Ruby recipe's replace_shebangs method.
	// The bin/ dir may not exist when running under a fake runner in tests.
	binDir := filepath.Join(gemHome, "bin")
	if _, statErr := os.Stat(binDir); os.IsNotExist(statErr) {
		// No bin/ dir — nothing to replace (e.g. fake runner in unit tests).
	} else if err := replaceShebangs(binDir); err != nil {
		return fmt.Errorf("bundler: replacing shebangs: %w", err)
	}

	// Remove the .gem cache file (Ruby recipe does `rm -f bundler-X.gem` and
	// `rm -rf cache/bundler-X.gem`).
	os.Remove(filepath.Join(gemHome, fmt.Sprintf("bundler-%s.gem", src.Version)))
	os.RemoveAll(filepath.Join(gemHome, "cache", fmt.Sprintf("bundler-%s.gem", src.Version)))

	// Tar gemHome contents into bundler-{version}.tgz in the CWD.
	// The Ruby recipe does `tar czvf {current_dir}/{archive_filename} .` from gemHome.
	archiveName := fmt.Sprintf("bundler-%s.tgz", src.Version)
	if err := run.RunInDir(gemHome, "tar", "czf", filepath.Join(mustCwd(), archiveName), "."); err != nil {
		return fmt.Errorf("bundler: creating archive: %w", err)
	}

	return nil
}

// replaceShebangs replaces Ruby interpreter shebangs in bin/ scripts with
// the portable `#!/usr/bin/env ruby` form.
func replaceShebangs(binDir string) error {
	return filepath.WalkDir(binDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) == 0 || !strings.HasPrefix(lines[0], "#!") {
			return nil
		}

		shebang := lines[0]
		// Only replace Ruby shebangs (lines containing "ruby").
		if !strings.Contains(shebang, "ruby") {
			return nil
		}

		newContent := strings.Replace(content, shebang, "#!/usr/bin/env ruby", 1)
		return os.WriteFile(path, []byte(newContent), 0755)
	})
}
