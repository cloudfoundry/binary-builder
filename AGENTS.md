# Agent Guidelines for Binary Builder

## Overview

Go tool that compiles CF buildpack dependencies (ruby, node, python, httpd, …) for a
specific CF stack (cflinuxfs4, cflinuxfs5). Entry point: `cmd/binary-builder/main.go`.

---

## Build & Test Commands

### Unit tests (Tier 1 — no Docker, no network)
```bash
go test ./...                  # all packages
go test -race ./...            # with race detector (CI requirement)
```

### Run a single test or test file
```bash
# By test name (regex) within a package:
go test ./internal/recipe/ -run TestRubyRecipeBuild
go test ./internal/runner/  -run TestFakeRunner
go test ./internal/stack/   -run TestLoad

# Run an entire package verbosely:
go test -v ./internal/recipe/

# Run with race detector for a single package:
go test -race ./internal/recipe/ -run TestHTTPD
```

### Parity tests (Tier 2 — requires Docker + network)
```bash
make parity-test DEP=httpd [STACK=cflinuxfs4]
make parity-test-all [STACK=cflinuxfs4]
```

### Exerciser test (Tier 3 — requires Docker)
```bash
make exerciser-test ARTIFACT=/tmp/ruby_3.3.6_...tgz STACK=cflinuxfs4
```

### Build the binary
```bash
go build ./cmd/binary-builder
```

---

## Architecture

```
cmd/binary-builder/main.go      ← CLI entry point
internal/
  recipe/      ← one file per dep (ruby.go, node.go, httpd.go, …)
  php/         ← PHP extension recipes (pecl.go, fake_pecl.go)
  runner/      ← Runner interface + RealRunner + FakeRunner
  stack/       ← Stack struct + YAML loader
  fetch/       ← Fetcher interface + HTTPFetcher
  archive/     ← StripTopLevelDir, StripFiles, InjectFile
  source/      ← source.Input (data.json parser)
  output/      ← OutData, BuildOutput, DepMetadataOutput
  artifact/    ← Artifact naming, SHA256, S3 URL helpers
  apt/         ← apt.New(runner).Install(ctx, pkgs...)
  portile/     ← configure/make/install wrapper
stacks/        ← cflinuxfs4.yaml, cflinuxfs5.yaml  ← ALL stack-specific data lives here
```

### Key design rules
- **Stack config is data, not code.** Every Ubuntu-version-specific value
  (apt packages, compiler paths, bootstrap URLs) lives in `stacks/{stack}.yaml`.
  Recipes read from `*stack.Stack`; no stack names are hardcoded in Go source.
- **Runner interface** — all `exec.Cmd` usage goes through `runner.Runner`.
  `RealRunner` executes; `FakeRunner` records calls for tests.
- **Fetcher interface** — all HTTP calls go through `fetch.Fetcher`.
  `HTTPFetcher` does the real work; `FakeFetcher` (in `recipe_test.go`) is used in tests.
- **`RunInDirWithEnv` appends env vars** — appended vars win over inherited env on Linux.
  So `GOTOOLCHAIN=local` appended DOES override any existing `GOTOOLCHAIN`.
- **miniconda3-py39 is URL-passthrough**: `Build()` sets `outData.URL`/`outData.SHA256`
  directly instead of writing a file. `main.go` checks `if outData.URL == ""` before
  calling `handleArtifact`.

---

## Code Style

### Language & toolchain
- **Go only.** The Ruby binary-builder has been fully removed.
- Module: `github.com/cloudfoundry/binary-builder` — use this import path.
- Minimum Go version: see `go.mod`.

### Naming conventions
| Kind | Convention | Example |
|------|-----------|---------|
| Exported type / func | PascalCase | `RubyRecipe`, `NewRegistry` |
| Unexported func / var | camelCase | `buildRegistry`, `mustCwd` |
| Interface | noun (no `I` prefix) | `Runner`, `Fetcher`, `Recipe` |
| Test helper | camelCase | `newFakeFetcher`, `useTempWorkDir` |
| Constants | PascalCase (exported) or camelCase | — |

### Import grouping
Three groups, separated by blank lines:
```go
import (
    // 1. stdlib
    "context"
    "fmt"
    "os"

    // 2. third-party
    "gopkg.in/yaml.v3"
    "github.com/stretchr/testify/assert"

    // 3. internal
    "github.com/cloudfoundry/binary-builder/internal/runner"
    "github.com/cloudfoundry/binary-builder/internal/stack"
)
```

### Error handling
- Return errors explicitly; never panic in production code paths.
- Wrap with context using `fmt.Errorf("component: action: %w", err)`.
- Pattern: `return fmt.Errorf("ruby: apt install ruby_build: %w", err)`
- Error strings are lowercase (Go convention).
- On fatal CLI errors: `fmt.Fprintf(os.Stderr, "binary-builder: %v\n", err); os.Exit(1)`.

### Comments
- Package-level doc comment on every package: `// Package foo does X.`
- Exported types/funcs always have doc comments.
- Inline comments explain *why*, not *what*.
- Use `// nolint:errcheck` only when the error is genuinely ignorable (e.g., closing
  a writer in a test after all data is flushed).

### Structs and interfaces
- Define interfaces where the consumer lives, not where the implementation lives.
- Struct fields use PascalCase; YAML tags use snake_case: `InstallDir string \`yaml:"install_dir"\``.
- Zero-value structs should be usable where practical.

---

## Recipe Patterns

Every recipe implements `recipe.Recipe`:
```go
type Recipe interface {
    Name() string
    Build(ctx context.Context, s *stack.Stack, src *source.Input, r runner.Runner, outData *output.OutData) error
    Artifact() ArtifactMeta  // OS, Arch, Stack ("" = use build stack)
}
```

### Shared recipe abstractions

Before writing a new recipe from scratch, check whether one of these abstractions fits:

| Abstraction | Location | Use when |
|-------------|----------|----------|
| `autoconf.Recipe` | `internal/autoconf/` | configure / make / make install cycle (libunwind, libgdiplus, openresty, nginx) |
| `RepackRecipe` | `internal/recipe/repack.go` | Download an archive and optionally strip its top-level dir (bower, yarn, setuptools, rubygems) |
| `BundleRecipe` | `internal/recipe/bundle.go` | `pip3 download` multiple packages into a tarball (pip, pipenv) |
| `GoToolRecipe` | `internal/recipe/dep.go` | Download + build a Go tool with `go get`/`go build` (dep, glide, godep) |
| `PassthroughRecipe` | `internal/recipe/passthrough.go` | No build step — just record the upstream URL and SHA256 |

#### Using `autoconf.Recipe`

`autoconf.Recipe` lives in `internal/autoconf/` (separate package to avoid import cycles).
It is **not** a `recipe.Recipe` itself — wrap it in a thin struct in `internal/recipe/`:

```go
type MyRecipe struct{ Fetcher fetch.Fetcher }

func (r *MyRecipe) Name() string         { return "mylib" }
func (r *MyRecipe) Artifact() ArtifactMeta { return ArtifactMeta{OS: "linux", Arch: "x64"} }

func (r *MyRecipe) Build(ctx context.Context, s *stack.Stack, src *source.Input,
    run runner.Runner, out *output.OutData) error {
    return (&autoconf.Recipe{
        DepName: "mylib",
        Fetcher: r.Fetcher,
        Hooks: autoconf.Hooks{
            AptPackages:   func(s *stack.Stack) []string { return s.AptPackages["mylib_build"] },
            ConfigureArgs: func(_, prefix string) []string { return []string{"--prefix=" + prefix, "--enable-shared"} },
            PackDirs:      func() []string { return []string{"include", "lib"} },
        },
    }).Build(ctx, s, src, run, out)
}
```

Available hooks (all optional — nil = default behaviour):

| Hook | Default | Typical override |
|------|---------|-----------------|
| `AptPackages` | `s.AptPackages["{name}_build"]` | Custom package list |
| `BeforeDownload` | no-op | GPG verification (nginx) |
| `SourceProvider` | fetch tarball, extract to `/tmp/{name}-{version}` | `git clone` (libgdiplus), read from `source/` (libunwind) |
| `AfterExtract` | no-op | `autoreconf -i`, `autogen.sh` |
| `ConfigureArgs` | `["--prefix={prefix}"]` | Full custom args |
| `ConfigureEnv` | nil | `CFLAGS`, `CXXFLAGS` |
| `MakeArgs` | nil | `["-j2"]` |
| `InstallEnv` | nil (falls back to `ConfigureEnv`) | `DESTDIR` (nginx) |
| `AfterInstall` | no-op | Remove runtime dirs, symlinks |
| `PackDirs` | `["."]` | `["include", "lib"]`, `["lib"]` |
| `AfterPack` | no-op | `archive.StripTopLevelDir` (nginx) |

### Adding a new recipe
1. Create `internal/recipe/{name}.go` with a struct implementing `Recipe`.
2. Register it in `buildRegistry()` in `cmd/binary-builder/main.go`.
3. Add a test in `internal/recipe/recipe_test.go` or a new `{name}_test.go` file.
4. If the dep is architecture-neutral, set `Arch: "noarch"` in `ArtifactMeta`.
5. For URL-passthrough deps (no build step), use `PassthroughRecipe` or
   set `outData.URL`/`outData.SHA256` directly.
6. For autoconf-based deps, use `autoconf.Recipe` with hooks (see above).
7. For download-and-strip deps, use `RepackRecipe`.

### Stack-specific behaviour
Recipes **must not** contain `if s.Name == "cflinuxfs4"` guards. Instead:
- Add the relevant value to both `stacks/cflinuxfs4.yaml` and `stacks/cflinuxfs5.yaml`.
- Read it from `s.AptPackages["key"]`, `s.Python.UseForceYes`, etc.

---

## Testing Conventions

- Test package: always use `package recipe_test` (external test package) for `internal/recipe/`.
  Other packages follow the same `_test` suffix convention.
- Assertion library: `github.com/stretchr/testify/assert` (non-fatal) and
  `github.com/stretchr/testify/require` (fatal / setup).
- Use `require.NoError` for setup steps; `assert.*` for behaviour assertions.
- **`FakeRunner`** in `internal/runner/runner.go` — inject instead of `RealRunner` in tests.
  Inspect `fakeRunner.Calls` to verify command sequence, args, env, and dir.
- **`FakeFetcher`** defined in `recipe_test.go` — inject instead of `HTTPFetcher`.
  Inspect `f.DownloadedURLs` and set `f.ErrMap` / `f.BodyMap` to control behaviour.
- **`useTempWorkDir(t)`** — helper in `recipe_helpers_test.go`. Call it in tests that
  need a clean CWD (recipes write artifacts relative to CWD). NOT safe for `t.Parallel()`.
- **`writeFakeArtifact(t, name)`** — creates a minimal valid `.tgz` in CWD so archive
  helpers don't fail when processing a fake build output.
- **Table-driven tests** are preferred for multiple similar cases (see
  `TestCompiledRecipeArtifactMetaSanity`).
- Test names follow `Test{Type}{Behaviour}` pattern: `TestRubyRecipeBuild`,
  `TestNodeRecipeStripsVPrefix`.

---

## Parity Test Infrastructure

- Script: `test/parity/compare-builds.sh --dep <name> --data-json <path> [--stack <stack>]`
- Logs: `/tmp/parity-logs/<dep>-<version>-<stack>.log`
- The Go builder is compiled **inside the container at runtime** (`go build ./cmd/binary-builder`),
  so source changes are always picked up on re-run without a separate image rebuild.

### Sample data.json files (on the build host at `/tmp/`)
| File | Dep | Version |
|------|-----|---------|
| `/tmp/go-data.json` | go | 1.22.0 |
| `/tmp/node-data.json` | node | 20.11.0 |
| `/tmp/php-data.json` | php | 8.1.32 |
| `/tmp/r-data.json` | r | 4.2.3 |
| `/tmp/jruby-data.json` | jruby | 9.4.14.0 |
| `/tmp/appdynamics-data.json` | appdynamics | 23.11.0-839 |
| `/tmp/skywalking-data.json` | skywalking-agent | 9.5.0 |

R sub-dep data.json files live under `/tmp/r-sub-deps/source-{pkg}-latest/data.json`
(forecast, plumber, rserve, shiny).

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `cmd/binary-builder/main.go` | CLI, `findIntermediateArtifact`, `handleArtifact`, `buildRegistry` |
| `internal/runner/runner.go` | `Runner` interface, `RealRunner`, `FakeRunner` + `Call` type |
| `internal/fetch/fetch.go` | `Fetcher` interface, `HTTPFetcher` |
| `internal/stack/stack.go` | `Stack` struct, `Load(stacksDir, name)` |
| `internal/recipe/recipe.go` | `Recipe` interface, `Registry` |
| `internal/archive/archive.go` | `StripTopLevelDir`, `StripFiles`, `InjectFile` |
| `internal/portile/` | configure/make/install abstraction |
| `internal/apt/` | apt-get install wrapper |
| `stacks/cflinuxfs4.yaml` | All cflinuxfs4-specific values |
| `stacks/cflinuxfs5.yaml` | All cflinuxfs5-specific values |
| `test/parity/compare-builds.sh` | Parity test harness |
| `Makefile` | `unit-test`, `unit-test-race`, `parity-test`, `exerciser-test` |

## cflinuxfs4/ports/ note
The `cflinuxfs4/ports/` directory contains root-owned build artifacts from previous Ruby
parity test runs. These cannot be removed without `sudo` and are NOT tracked by git.
They do not affect builds or tests.
