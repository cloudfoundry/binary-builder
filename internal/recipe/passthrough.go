package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/fileutil"
	"github.com/cloudfoundry/binary-builder/internal/output"
	"github.com/cloudfoundry/binary-builder/internal/runner"
	"github.com/cloudfoundry/binary-builder/internal/source"
	"github.com/cloudfoundry/binary-builder/internal/stack"
)

// SourceFilenameFunc returns the expected source filename for a given version.
type SourceFilenameFunc func(version string) string

// PassthroughRecipe handles dependencies that are downloaded and passed through
// without compilation. Covers: tomcat, composer, appdynamics, appdynamics-java,
// skywalking-agent, openjdk, zulu, sapmachine, jprofiler-profiler, your-kit-profiler.
type PassthroughRecipe struct {
	DepName            string
	SourceFilenameFunc SourceFilenameFunc
	Meta               ArtifactMeta
	Fetcher            fetch.Fetcher
}

func (p *PassthroughRecipe) Name() string { return p.DepName }

func (p *PassthroughRecipe) Artifact() ArtifactMeta { return p.Meta }

func (p *PassthroughRecipe) Build(ctx context.Context, _ *stack.Stack, src *source.Input, r runner.Runner, _ *output.OutData) error {
	filename := p.SourceFilenameFunc(src.Version)
	localPath := filepath.Join("source", filename)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("creating source directory: %w", err)
	}

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		if err := p.Fetcher.Download(ctx, src.URL, localPath, src.PrimaryChecksum()); err != nil {
			return fmt.Errorf("downloading %s: %w", p.DepName, err)
		}
	}

	// Move the downloaded file to the working directory with a version-named
	// intermediate filename (e.g. composer-2.7.1.phar, tomcat-9.0.85.tar.gz)
	// so that findIntermediateArtifact in main.go can locate it via glob
	// patterns like "tomcat-9.0.85*.tar.gz".
	ext := archiveExt(filename)
	intermediateName := fmt.Sprintf("%s-%s%s", p.DepName, src.Version, ext)
	if err := fileutil.MoveFile(localPath, intermediateName); err != nil {
		return fmt.Errorf("staging artifact %s: %w", p.DepName, err)
	}

	return nil
}

// NewPassthroughRecipes creates all passthrough and repack-only recipe instances.
// This includes JVM passthrough deps (openjdk, zulu, …) as well as PyPI sdist
// deps (setuptools, flit-core, …) — anything that needs no compilation step.
// Add a new entry here whenever buildpacks-ci adds a dep with source_type: pypi
// or a similar "download-only" source type.
func NewPassthroughRecipes(f fetch.Fetcher) []Recipe {
	return []Recipe{
		&PassthroughRecipe{
			DepName:            "tomcat",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("apache-tomcat-%s.tar.gz", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName:            "composer",
			SourceFilenameFunc: func(_ string) string { return "composer.phar" },
			Meta:               ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName:            "appdynamics",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("appdynamics-php-agent-linux_x64-%s.tar.bz2", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName:            "appdynamics-java",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("appdynamics-java-agent-%s.zip", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName: "skywalking-agent",
			SourceFilenameFunc: func(v string) string {
				return fmt.Sprintf("apache-skywalking-java-agent-%s.tgz", v)
			},
			Meta:    ArtifactMeta{OS: "linux", Arch: "noarch", Stack: "any-stack"},
			Fetcher: f,
		},
		&PassthroughRecipe{
			DepName:            "openjdk",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("bellsoft-jre%s-linux-amd64.tar.gz", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName:            "zulu",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("zulu%s-jre-linux_x64.tar.gz", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""},
			Fetcher:            f,
		},
		&PassthroughRecipe{
			DepName: "sapmachine",
			SourceFilenameFunc: func(v string) string {
				return fmt.Sprintf("sapmachine-jre-%s_linux-x64_bin.tar.gz", v)
			},
			Meta:    ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""},
			Fetcher: f,
		},
		&PassthroughRecipe{
			DepName: "jprofiler-profiler",
			SourceFilenameFunc: func(v string) string {
				return fmt.Sprintf("jprofiler_linux_%s.tar.gz", underscoreVersion(v))
			},
			Meta:    ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""},
			Fetcher: f,
		},
		&PassthroughRecipe{
			DepName:            "your-kit-profiler",
			SourceFilenameFunc: func(v string) string { return fmt.Sprintf("YourKit-JavaProfiler-%s.zip", v) },
			Meta:               ArtifactMeta{OS: "linux", Arch: "x64", Stack: ""},
			Fetcher:            f,
		},
		// PyPI sdist deps — download and strip top-level dir, no compilation.
		&PyPISourceRecipe{DepName: "setuptools", Fetcher: f},
		&PyPISourceRecipe{DepName: "flit-core", Fetcher: f},
		&PyPISourceRecipe{DepName: "poetry-core", Fetcher: f},
	}
}

// underscoreVersion replaces dots with underscores: "13.0.14" → "13_0_14".
func underscoreVersion(v string) string {
	result := make([]byte, len(v))
	for i := range v {
		if v[i] == '.' {
			result[i] = '_'
		} else {
			result[i] = v[i]
		}
	}
	return string(result)
}

// archiveExt returns the full file extension, handling compound extensions
// like ".tar.gz", ".tar.bz2", ".tar.xz" that filepath.Ext misses.
func archiveExt(filename string) string {
	for _, compound := range []string{".tar.gz", ".tar.bz2", ".tar.xz"} {
		if len(filename) > len(compound) && filename[len(filename)-len(compound):] == compound {
			return compound
		}
	}
	return filepath.Ext(filename)
}
