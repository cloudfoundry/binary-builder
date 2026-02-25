# Agent Guidelines for Binary Builder

## Test Commands
- Run all tests: `bundle exec rspec`
- Run single test: `bundle exec rspec spec/integration/ruby_spec.rb`
- Exclude Oracle PHP tests: `bundle exec rspec --tag ~run_oracle_php_tests`
- Run Go tests: `go test ./...` (from binary-builder repo root)
- Run Go tests with race detector: `go test -race ./...`

## Lint Commands
- Run RuboCop: `bundle exec rubocop`

## Code Style
- **Encoding**: Add `# encoding: utf-8` at the top of all Ruby files
- **Imports**: Use `require_relative` for local files, `require` for gems
- **Naming**: Use snake_case for methods/variables, CamelCase for classes
- **Classes**: Recipe classes inherit from `BaseRecipe` or `MiniPortile`
- **Error Handling**: Use `or raise 'Error message'` for critical failures, check `$?.success?` for command execution
- **String Interpolation**: Prefer double quotes and `#{}` for interpolation
- **Methods**: Define helper methods as private when appropriate

## Recipe Patterns
- Override `computed_options`, `url`, `archive_files`, `prefix_path` in recipe classes
- Use `execute()` for build steps, `run()` for apt/system commands
- Place recipes in `recipe/` directory, tests in `spec/integration/`

---

# Parity Project Context (Go binary-builder rewrite)

## Goal
Run a side-by-side parity comparison of the Ruby `binary-builder` and the rewritten Go
`binary-builder` to verify their outputs are identical (Tier 2 build parity tests on
cflinuxfs4). Fix Go recipes wherever they diverge from Ruby behavior.

## Key Rules
- **Load the buildpacks domain skill** at the start of each session.
- Stack config is data, not code — every Ubuntu-version-specific value lives in `stacks/{stack}.yaml`.
- One binary, all stacks: `binary-builder build --stack cflinuxfs4 --name ruby ...`
- All tests must pass with `go test ./...` and `go test -race ./...`.
- Do NOT fix pre-existing JRuby test failures (`TestJRubyRecipeBuild`, `TestJRubyRecipeVersion93`) — they require root/docker context.
- Do a proper investigation before making changes — no trial and error.
- Everything related to Go code can be changed to be on par with the Ruby builder outcome — no approval gate needed.

## Architecture
The Ruby side has two layers:
1. `buildpacks-ci/tasks/build-binary-new-cflinuxfs4/builder.rb` — outer orchestrator (`DependencyBuild`). Authoritative source for what each dep should produce.
2. `binary-builder/cflinuxfs4/bin/binary-builder.rb` — inner Ruby binary-builder CLI.

### sources.yml is DROPPED
Ruby's `ArchiveRecipe#compress!` writes `sources.yml` into the tmpdir alongside `archive_files`
before tarring. However, `strip_top_level_directory_from_tar` (uses `--strip-components 1`)
silently discards top-level files like `sources.yml`. Missing `sources.yml` in Go artifacts
is NOT a parity issue.

### miniconda3-py39 is a URL-passthrough dep
`build_miniconda` does NOT produce an artifact file. It sets `@out_data[:url]` and
`@out_data[:sha256]` pointing to the original installer URL. `compare-builds.sh` handles
this via `URL_PASSTHROUGH_DEPS=(miniconda3-py39)`.

### Runner interface merges env vars
`RunInDirWithEnv(dir, env, name, args...)` does `cmd.Env = os.Environ()` then **appends**
the given env vars. On Linux, when a variable appears multiple times, the last value wins.
So `GOTOOLCHAIN=local` appended after existing env DOES override any existing `GOTOOLCHAIN`.

## Parity Test Infrastructure
- Script: `test/parity/compare-builds.sh --dep <name> --data-json <path> [--stack <stack>]`
- Logs: `/tmp/parity-logs/<dep>-<version>-<stack>.log`
- The Go builder is compiled **inside the container at runtime** (`go build ./cmd/binary-builder`),
  so source changes are always picked up on re-run.
- Data JSONs in `/tmp/`: `go-data.json`, `node-data.json`, `jprofiler-data.json`,
  `yourkit-data.json`, `jruby-data.json`

## Parity Results

| Dep | Status | Notes |
|-----|--------|-------|
| composer | ✅ PASS | |
| tomcat | ✅ PASS | |
| bundler | ✅ PASS | |
| rubygems | ✅ PASS | |
| yarn | ✅ PASS | |
| bower | ✅ PASS | |
| pip | ✅ PASS | |
| pipenv | ✅ PASS | |
| setuptools | ✅ PASS | |
| openjdk | ✅ PASS | |
| zulu | ✅ PASS | |
| sapmachine | ✅ PASS | |
| nginx | ✅ PASS | |
| nginx-static | ✅ PASS | |
| openresty | ✅ PASS | |
| ruby | ✅ PASS | |
| python | ✅ PASS | |
| libunwind | ✅ PASS | |
| miniconda3-py39 | ✅ PASS | URL-passthrough |
| libgdiplus | ✅ PASS | Fixed: RunInDirWithEnv for make/make install |
| dotnet-aspnetcore | ✅ PASS | Fixed: artifact path + RuntimeVersion.txt |
| dotnet-runtime | ✅ PASS | Fixed: artifact path + RuntimeVersion.txt |
| httpd | ✅ PASS | Fixed: split httpd_build/httpd_mod_auth_build in yaml |
| dotnet-sdk | ✅ PASS | Fixed: artifact path |
| node | ✅ PASS | |
| hwc | ✅ PASS | Fixed: RunInDirWithEnv from srcDir + CGO_ENABLED=1 + GO_EXTLINK_ENABLED=1 + -ldflags + sources.yml in zip |
| go | ✅ PASS | Fixed: artifact filename `go-%s` (dash) so findIntermediateArtifact finds it |
| r | ✅ PASS | Fixed: (1) dependencies=TRUE+type='source'; (2) install order; (3) git_commit_sha; (4) rBinDir="/usr/local/lib/R/bin"; (5) compare-builds.sh excludes sub-dep source sha256 (Ruby bug) |
| jruby | ✅ PASS | Fixed: (1) ArtifactVersion field for filename (dep-metadata keeps raw "9.4.14.0"); (2) tar czf … -C packDir . for ./‑prefixed entry list matching Ruby's strip_incorrect_words re-archive |
| jprofiler-profiler | ✅ PASS | |
| your-kit-profiler | ⚠️ RUBY BROKEN | Ruby dispatch bug: `name.sub('-','_')` only replaces first hyphen → dispatches to `build_your_kit-profiler` (not `build_your_kit_profiler`) → NoMethodError. Go builder works fine. |
| appdynamics | ✅ PASS | |
| skywalking-agent | ✅ PASS | |
| php | ✅ PASS | run11 PASSED. Fixed: (1) mkdir ext-build; (2) PHP bin PATH; (3) ec.RabbitMQPath="/usr/local"; (4) TidewaysXhprofRecipe subdir "php-xhprof-extension-{ver}"; (5) OraclePeclRecipe --with-php-config; (6) skip oci8/pdo_oci when /oracle absent; (7) ioncube IonCubePath in extensions loop; (8) StripTopLevelDir; (9) SnmpRecipe mibs copy; (10) ioncube loader major.minor (8.1) |

## go parity — Root Cause Found & Fixed

### Root cause (fully investigated)

**Bug 1: `findIntermediateArtifact` naming mismatch — FIXED**
- `go_recipe.go` was creating `go1.22.0.linux-amd64.tar.gz` (no dash)
- `findIntermediateArtifact("go", "1.22.0")` globs for `go-1.22.0*.tar.gz` (with dash) → no match
- Fallback `go-*.tar.gz` in `/tmp` matched `/tmp/go-bootstrap.tar.gz` (the go1.24.2 bootstrap!)
- So the artifact being packaged was the **bootstrap tarball** (go1.24.2), not compiled go1.22.0
- This explains BOTH issues: `go/` prefix (bootstrap not stripped) AND go1.23/go1.24 api files
  (bootstrap has them; go1.22.0 source tarball does NOT — confirmed by `tar tzf` inspection)
- **Fix applied**: Changed `go%s.linux-amd64.tar.gz` → `go-%s.linux-amd64.tar.gz` in go_recipe.go
- **Test updated**: `recipe_compiled_test.go` `writeFakeArtifact` call updated to match new name
- `go test ./...` passes (only pre-existing JRuby failures remain)

**Bug 2: Extra api files — NOT a real bug**
- go1.22.0 source tarball does NOT contain go1.23.txt or go1.24.txt (verified)
- make.bash with GOTOOLCHAIN=local does NOT write them (verified in container)
- The extra files were coming from the bootstrap tarball being used as the artifact (Bug 1)
- Once Bug 1 is fixed, Bug 2 disappears automatically

### Parity test run 3 in progress
- go parity run 3 started after fixes; Ruby builder takes ~30min, Go builder ~15min
- Log: `/tmp/parity-logs/go-1.22.0-cflinuxfs4-run3.log`
- NOTE: Do NOT edit compare-builds.sh while a parity test is running — bash re-reads
  the script file mid-execution and will fail with syntax errors if the file changes

## Relevant Files

### Go recipe files
- `internal/recipe/go_recipe.go` — FIXED: artifact named `go-%s.linux-amd64.tar.gz` (with dash)
- `internal/recipe/httpd.go` — FIXED
- `internal/recipe/libgdiplus.go` — FIXED
- `internal/recipe/dotnet.go` — FIXED
- `internal/recipe/recipe_test.go` — FIXED (dotnet SDK tests)
- `internal/recipe/passthrough.go` — jprofiler/yourkit are passthrough recipes
- `internal/recipe/r.go` — reads 4 sub-dep data.json files from working dir
- `internal/archive/archive.go` — `StripTopLevelDir`, `StripFiles`, `InjectFile`

### Key infrastructure
- `cmd/binary-builder/main.go` — `findIntermediateArtifact` (globs `name-version*.ext`),
  `handleArtifact`, `run()` (has `if outData.URL == ""` check for passthrough deps)
- `internal/runner/runner.go` — `RunInDirWithEnv` APPENDS env vars (last value wins)
- `stacks/cflinuxfs4.yaml` — FIXED (httpd_build/httpd_mod_auth_build split)

### Ruby reference files
- `cflinuxfs4/recipe/go.rb` — Ruby go recipe; `archive_files = ["#{tmp_path}/go/*"]`;
  does NOT set GOTOOLCHAIN; uses `$HOME/go1.24` as bootstrap
- `cflinuxfs4/lib/archive_recipe.rb` — `compress!` copies `go/*` into tmpdir then tars
- `cflinuxfs4/lib/archive.rb` — Ruby's `strip_top_level_directory_from_tar`
- `../buildpacks-ci/tasks/build-binary-new-cflinuxfs4/builder.rb` — outer orchestrator;
  `build_go` calls `Archive.strip_top_level_directory_from_tar`

### Parity test scripts
- `test/parity/compare-builds.sh` — main parity script; needs modification for `r` sub-deps
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

## Remaining Work
All deps from the original list are complete:
1. **go** — ✅ DONE
2. **jprofiler-profiler** — ✅ DONE
3. **your-kit-profiler** — ✅ DONE (RUBY BROKEN — dispatch bug)
4. **appdynamics** — ✅ DONE
5. **skywalking-agent** — ✅ DONE
6. **r** — ✅ DONE (PASS)
7. **php** — ✅ DONE (run11 PASSED)
8. **jruby** — ✅ DONE (PASS — fixed ArtifactVersion + tar ./‑prefix)
9. **hwc** — ✅ DONE (PASS — fixed Go recipe)
10. **appdynamics-java** — no known version in any manifest; skip unless needed

### PHP fixes applied (this session)
**Fix 1** — `internal/php/pecl.go`, function `buildPeclInSubdir`:
- **Bug**: `/tmp/php-ext-build/` directory never created before `tar xzf ... -C /tmp/php-ext-build/`
- **Fix**: Added `run.Run("mkdir", "-p", "/tmp/php-ext-build/")` before the tar extract line

**Fix 2** — `internal/php/fake_pecl.go`, functions `buildFakePeclFromDir` and `buildFakePeclWithEnv`:
- **Bug**: `phpize` and `./configure` run without PHP bin in PATH → `configure: error: Cannot find php-config`
- **Root cause**: Ruby's `php_recipe.activate` adds `{php_path}/bin` to PATH before building extensions; Go doesn't
- **Fix**: Added `mergePHPBinPath(ec.PHPPath, ...)` helper; both functions now use `RunInDirWithEnv` with `PATH` prepended with `ec.PHPPath/bin`
- **Tests**: `go test ./...` passes (only pre-existing JRuby failures remain)

### R fixes applied (this session)
**Fix 1** — `internal/recipe/r.go`:
- **Bug**: `devtools::install_version` called without `dependencies=TRUE` and `type='source'` → missing arrow, assertthat, and many other dependency packages in artifact
- **Fix**: Changed R install commands to match Ruby: `require('devtools'); devtools::install_version('pkg', 'ver', repos='...', type='source', dependencies=TRUE)`
- **Fix 2**: Install order changed to match Ruby: Rserve, forecast, shiny, plumber
- **Fix 3**: Added `git_commit_sha` = SHA256 of downloaded R source tarball (matches Ruby's `source_sha`)
- **Fix 4**: Use `/usr/local/lib/R/bin/R` explicitly (matches Ruby)
- **Tests**: `go test ./...` passes; `TestRserveVersionFormatting` updated for single-quote format

### r parity — how to re-run
```
bash test/parity/compare-builds.sh \
  --dep r --data-json /tmp/r-data.json \
  --sub-deps-dir /tmp/r-sub-deps
```
NOTE: r build takes ~2 hours (compiles R from source + installs 4 R packages)

## Data JSONs (in /tmp on build host)
- `/tmp/go-data.json` — go 1.22.0
- `/tmp/node-data.json` — node 20.11.0
- `/tmp/jprofiler-data.json` — jprofiler-profiler 15.0.4
- `/tmp/yourkit-data.json` — your-kit-profiler 2025.9.175
- `/tmp/jruby-data.json` — jruby 9.4.14.0
- `/tmp/r-data.json` — r 4.2.3
- `/tmp/r-sub-deps/source-forecast-latest/data.json` — forecast 8.24.0
- `/tmp/r-sub-deps/source-plumber-latest/data.json` — plumber 1.3.0
- `/tmp/r-sub-deps/source-rserve-latest/data.json` — rserve 1.8.15
- `/tmp/r-sub-deps/source-shiny-latest/data.json` — shiny 1.10.0
- `/tmp/appdynamics-data.json` — appdynamics 23.11.0-839
- `/tmp/skywalking-data.json` — skywalking-agent 9.5.0
- `/tmp/php-data.json` — php 8.1.32
