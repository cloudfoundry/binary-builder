# Agent Guidelines for Binary Builder

## Test Commands
- Run all Go tests: `go test ./...`
- Run Go tests with race detector: `go test -race ./...`
- Run single Go test file: `go test ./internal/recipe/ -run TestRuby`

## Code Style
- **Language**: Go only. The Ruby binary-builder has been removed.
- **Naming**: Use camelCase for unexported, PascalCase for exported identifiers
- **Error Handling**: Return errors explicitly; use `fmt.Errorf("...: %w", err)` for wrapping
- **Testing**: Use `github.com/stretchr/testify` assertions
- **Stack config is data, not code**: every Ubuntu-version-specific value lives in `stacks/{stack}.yaml`

## Recipe Patterns
- One binary, all stacks: `binary-builder build --stack cflinuxfs4 --name ruby ...`
- Recipes live in `internal/recipe/`; PHP extension recipes in `internal/php/`
- All tests must pass with `go test ./...` and `go test -race ./...`
- Do NOT fix pre-existing JRuby test failures (`TestJRubyRecipeBuild`, `TestJRubyRecipeVersion93`) — they require root/docker context

---

# Go binary-builder — Architecture & Context

## Key Rules
- **Load the buildpacks domain skill** at the start of each session.
- Do a proper investigation before making changes — no trial and error.

## Architecture

### Entry point
- `cmd/binary-builder/main.go` — `findIntermediateArtifact` (globs `name-version*.ext`),
  `handleArtifact`, `run()` (has `if outData.URL == ""` check for passthrough deps)

### Runner interface merges env vars
`RunInDirWithEnv(dir, env, name, args...)` does `cmd.Env = os.Environ()` then **appends**
the given env vars. On Linux, when a variable appears multiple times, the last value wins.
So `GOTOOLCHAIN=local` appended after existing env DOES override any existing `GOTOOLCHAIN`.

### miniconda3-py39 is a URL-passthrough dep
`build_miniconda` does NOT produce an artifact file. It sets `outData.URL` and `outData.SHA256`
pointing to the original installer URL. `compare-builds.sh` handles this via
`URL_PASSTHROUGH_DEPS=(miniconda3-py39)`.

## Parity Test Infrastructure
- Script: `test/parity/compare-builds.sh --dep <name> --data-json <path> [--stack <stack>]`
- Logs: `/tmp/parity-logs/<dep>-<version>-<stack>.log`
- The Go builder is compiled **inside the container at runtime** (`go build ./cmd/binary-builder`),
  so source changes are always picked up on re-run.

## Relevant Go Files

### Recipes
- `internal/recipe/go_recipe.go` — artifact named `go-%s.linux-amd64.tar.gz` (with dash)
- `internal/recipe/httpd.go`
- `internal/recipe/libgdiplus.go`
- `internal/recipe/dotnet.go`
- `internal/recipe/passthrough.go` — jprofiler/yourkit are passthrough recipes
- `internal/recipe/r.go` — reads 4 sub-dep data.json files from working dir

### PHP
- `internal/php/pecl.go` — PECL extension build; creates `/tmp/php-ext-build/` before extraction
- `internal/php/fake_pecl.go` — uses `mergePHPBinPath` to prepend `ec.PHPPath/bin` to PATH

### Infrastructure
- `cmd/binary-builder/main.go` — CLI, `findIntermediateArtifact`, `handleArtifact`
- `internal/runner/runner.go` — `RunInDirWithEnv` APPENDS env vars (last value wins)
- `internal/archive/archive.go` — `StripTopLevelDir`, `StripFiles`, `InjectFile`
- `stacks/cflinuxfs4.yaml` — stack config (httpd_build/httpd_mod_auth_build split)

### Parity test scripts
- `test/parity/compare-builds.sh` — main parity script
- `test/parity/run-all.sh`

## Data JSONs (in /tmp on build host)
- `/tmp/go-data.json` — go 1.22.0
- `/tmp/node-data.json` — node 20.11.0
- `/tmp/jprofiler-data.json` — jprofiler-profiler 15.0.4
  (URL: `https://download.ej-technologies.com/jprofiler/jprofiler_linux_15_0_4.tar.gz`,
   sha256: `fec741718854a11b2383bb278ca7103984e0ae659268ed53ea5a8b32077b86c9`)
- `/tmp/yourkit-data.json` — your-kit-profiler 2025.9.175
  (URL: `https://download.yourkit.com/yjp/2025.9/YourKit-JavaProfiler-2025.9-b175-x64.zip`,
   sha256: `3c1e7600e76067cfc446666101db515a9a247d69333b7cba5dfb05cf40e5e1d9`)
- `/tmp/jruby-data.json` — jruby 9.4.14.0
- `/tmp/r-data.json` — r 4.2.3
- `/tmp/r-sub-deps/source-forecast-latest/data.json` — forecast 8.24.0
- `/tmp/r-sub-deps/source-plumber-latest/data.json` — plumber 1.3.0
- `/tmp/r-sub-deps/source-rserve-latest/data.json` — rserve 1.8.15
- `/tmp/r-sub-deps/source-shiny-latest/data.json` — shiny 1.10.0
- `/tmp/appdynamics-data.json` — appdynamics 23.11.0-839
- `/tmp/skywalking-data.json` — skywalking-agent 9.5.0
- `/tmp/php-data.json` — php 8.1.32

## cflinuxfs4/ports/ note
The `cflinuxfs4/ports/` directory contains root-owned build artifacts from previous Ruby
parity test runs. These cannot be removed without `sudo` in this environment. They are NOT
tracked by git and do not affect builds or tests. Add to `.gitignore` if needed.
