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

PHP requires an extensions YAML file:

```
binary-builder build \
  --stack cflinuxfs4 \
  --name php \
  --version 8.1.32 \
  --sha256 <checksum> \
  --php-extensions-file ./php_extensions/php81-extensions.yml
```

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

# Parity test for a single dep (requires Docker + network)
make parity-test DEP=ruby VERSION=3.3.6 SHA256=<checksum>

# Parity test for all deps
make parity-test-all
```

## Architecture

- `cmd/binary-builder/` — CLI entry point
- `internal/recipe/` — per-dependency build recipes
- `internal/php/` — PHP extension build logic
- `internal/archive/` — tarball / zip manipulation helpers
- `internal/runner/` — subprocess execution helpers
- `stacks/` — per-stack YAML configuration (versions, URLs, paths)
- `php_extensions/` — PHP extension lists per PHP minor version
- `test/parity/` — Tier 2 parity test scripts (compare Ruby vs Go builder outputs)

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).
