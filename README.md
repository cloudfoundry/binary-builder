# binary-builder

A Go tool for building binaries used by Cloud Foundry buildpacks.

## Supported binaries

| Dependency | Stacks |
|---|---|
| Ruby | cflinuxfs4 |
| JRuby | cflinuxfs4 |
| Python | cflinuxfs4 |
| Node.js | cflinuxfs4 |
| Go | cflinuxfs4 |
| PHP | cflinuxfs4 |
| Nginx / nginx-static / OpenResty | cflinuxfs4 |
| Apache HTTPD | cflinuxfs4 |
| Bundler | cflinuxfs4 |
| RubyGems | cflinuxfs4 |
| Yarn / Bower / Composer | cflinuxfs4 |
| Pip / Pipenv / Setuptools | cflinuxfs4 |
| OpenJDK / Zulu / SAPMachine | cflinuxfs4 |
| .NET SDK / Runtime / ASP.NET Core | cflinuxfs4 |
| HWC | cflinuxfs4 |
| R | cflinuxfs4 |
| libgdiplus / libunwind | cflinuxfs4 |
| miniconda3-py39 | cflinuxfs4 |
| AppDynamics / SkyWalking / JProfiler / YourKit | cflinuxfs4 |
| Tomcat | cflinuxfs4 |

## Usage

```
binary-builder build \
  --stack cflinuxfs4 \
  --name ruby \
  --version 3.3.6 \
  --sha256 <checksum>
```

The tool reads stack-specific configuration from `stacks/<stack>.yaml` and writes the
artifact (a `.tgz` or `.zip`) to the current working directory.

### PHP

PHP is built the same way as any other dependency — no extra flags needed.
Extension and native module definitions are embedded directly in the binary
(see `internal/php/assets/`):

```
binary-builder build \
  --stack cflinuxfs4 \
  --name php \
  --version 8.1.32 \
  --sha256 <checksum>
```

To add support for a new PHP minor version, create
`internal/php/assets/php<major><minor>-extensions-patch.yml` with any
additions or exclusions relative to the major-version base file. No code
changes are required — the file is discovered automatically at build time.

## Building

```bash
go build ./cmd/binary-builder
```

## Testing

```bash
# Unit tests (no Docker or network required)
make unit-test

# Unit tests with race detector
make unit-test-race

# Parity test for a single dep from the matrix (requires Docker + network)
# VERSION is not an argument — each dep runs at the version pinned in run-all.sh.
make parity-test DEP=ruby
make parity-test DEP=php STACK=cflinuxfs4

# To test a specific version not in the matrix, call compare-builds.sh directly
# with a custom data.json:
test/parity/compare-builds.sh --dep php --data-json /tmp/php-8.3.0-data.json --stack cflinuxfs4

# Parity test for all deps
make parity-test-all
```

## Architecture

- `cmd/binary-builder/` — CLI entry point
- `internal/recipe/` — per-dependency build recipes
- `internal/php/` — PHP extension build logic and embedded extension data (`assets/`)
- `internal/archive/` — tarball / zip manipulation helpers
- `internal/runner/` — subprocess execution helpers
- `stacks/` — per-stack YAML configuration (versions, URLs, paths)
- `test/parity/` — Parity test scripts (compare Ruby vs Go builder outputs)

## Parity Tests

The parity tests verify that the Go builder produces identical output to the
original Ruby builder for every supported dependency. This is the primary
confidence check that the Go rewrite is correct.

### Scripts

| Script | Purpose |
|---|---|
| `test/parity/run-all.sh` | Runs every dep in the test matrix sequentially; prints a pass/fail summary and tails failure logs |
| `test/parity/compare-builds.sh` | Runs both builders for a single dep and diffs their output |

### How it works

For each dependency, `compare-builds.sh` does the following:

**1. Source pre-download**

Some deps (`libunwind`, `dotnet-*`, `jprofiler-profiler`, `your-kit-profiler`)
are built from a source tarball that must already be present in a `source/`
directory at build time — neither builder downloads them inline. The script
downloads the tarball on the host first, then mounts it into both containers
as a read-only volume at `/tmp/host-source/`.

All other deps download their own source inside the container during the build.

**2. Run the Ruby builder**

Runs `buildpacks-ci/tasks/build-binary-new-cflinuxfs4/build.rb` inside a
`cloudfoundry/<stack>` Docker container with this layout:

```
/task/
  source/data.json          ← the depwatcher input
  source/<tarball>          ← pre-downloaded source (if applicable)
  source-*-latest/          ← R sub-dep data.json dirs (r dep only)
  binary-builder/           ← symlink to this repo
  buildpacks-ci/            ← symlink to ../buildpacks-ci
  artifacts/                ← artifact output (*.tgz / *.zip)
  dep-metadata/             ← dep-metadata JSON output
  builds-artifacts/
    binary-builds-new/<dep>/  ← builds JSON output
```

`SKIP_COMMIT=true` prevents git commits. Ruby 3.3.6 is compiled from source
inside the container if not already present.

**3. Run the Go builder**

Compiles `binary-builder` from source inside the same `cloudfoundry/<stack>`
container (using `mise` to install the required Go version), then runs:

```
binary-builder build \
  --stack <stack> \
  --source-file /tmp/data.json \
  --stacks-dir /binary-builder/stacks \
  --artifacts-dir /out/artifact \
  --builds-dir /out/builds \
  --dep-metadata-dir /out/dep-metadata \
  --skip-commit
```

The source tarball (if any) and R sub-dep dirs are copied into the working
directory before the build runs.

**4. Compare outputs**

If the Ruby builder failed, the comparison is skipped entirely — the test exits
0 with `RUBY BROKEN`. Otherwise all three output types are compared:

| Output | How it is compared | Hard failure? |
|---|---|---|
| **Artifact filename** | Both filenames are normalised by replacing the 8-char content SHA (`_<sha8>.`) with `_.` then compared | Yes |
| **Artifact contents** | Files inside the `.tgz` or `.zip` are listed and sorted, then diffed | Yes |
| **Builds JSON** | Fields `version`, `source.url`, `source.sha256`, `source.sha512`, `source.md5`, `url`, `sha256`, and `sub_dependencies[*].version` are compared individually | Yes |
| **Dep-metadata JSON structural fields** | All fields except `sha256` and `url` (the artifact hash) and `sub_dependencies[*].source.sha256` are compared with `jq -S` (sorted keys) | Yes |
| **Dep-metadata JSON artifact hash** | Top-level `sha256` and `url` fields are diffed | Warn only — non-reproducible builds (e.g. `bundler`) legitimately differ |
| **Sub-dep source sha256** | `sub_dependencies[*].source.sha256` | Warn only — Ruby builder has a known bug where it records the sha256 of an HTTP redirect response body rather than the actual tarball |

### Exit outcomes

| Result | Meaning |
|---|---|
| `PASS` | Both builders produced identical output on all hard-failure checks |
| `RUBY BROKEN` | Ruby builder failed; Go builder output not compared; exits 0 |
| `FAIL` | One or more hard-failure mismatches; exits 1 |

### Input format

Both builders receive the same depwatcher `data.json`:

```json
{
  "source": { "name": "ruby", "type": "github_releases", "repo": "ruby/ruby" },
  "version": { "url": "https://...", "ref": "3.3.6", "sha256": "...", "sha512": "" }
}
```

For SHA512-only deps (e.g. `dotnet-*`, `skywalking-agent`), `sha256` is `""`
and `sha512` carries the real checksum. Both fields are always present in the
builder output — the `sha256` field is never omitted even when empty.

### Running

```bash
# All deps (requires Docker + network)
test/parity/run-all.sh [<stack>]

# Single dep
test/parity/compare-builds.sh --dep ruby --data-json /tmp/ruby-data.json --stack cflinuxfs4

# R dep (needs sub-dep data.json dirs)
test/parity/compare-builds.sh --dep r --data-json /tmp/r-data.json \
  --sub-deps-dir /tmp/r-sub-deps
```

All output is written to both the terminal and
`/tmp/parity-logs/<dep>-<version>-<stack>.log`. To watch a running build:

```bash
tail -f /tmp/parity-logs/ruby-3.3.6-cflinuxfs4.log
```

`run-all.sh` prints a summary at the end and tails the last 20 lines of each
failure log automatically.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).
