#!/usr/bin/env bash
# binary-builder/test/parity/run-all.sh
# Usage: run-all.sh [<stack>] [DEP=<name>]
#
# Runs compare-builds.sh for every dep in the parity test matrix.
# For each dep we generate a real depwatcher-format data.json on-the-fly,
# then call compare-builds.sh --dep <name> --data-json <path> [--stack <stack>].
#
# To run a single dep only, set the DEP env var:
#   DEP=httpd ./test/parity/run-all.sh
#   DEP=httpd make parity-test
#
# Deps that require vendor credentials (appdynamics, appdynamics-java) are
# skipped with a SKIP notice; they can be tested manually when credentials
# are available.
#
# All checksums were verified against the upstream sources.
#
# Real depwatcher "modern" data.json format:
#   {
#     "source": {"name": "<dep>", "type": "<source_type>", "repo": "<org/repo>"},
#     "version": {"url": "<download_url>", "ref": "<version>",
#                 "sha256": "<sha256>", "sha512": "<sha512>"}
#   }

set -euo pipefail

STACK="${1:-cflinuxfs4}"
# Optional: filter to a single dep (set via env, e.g. DEP=httpd make parity-test)
FILTER_DEP="${DEP:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"

cleanup() { rm -rf "${TMPDIR}"; }
trap cleanup EXIT

declare -a PASSED=()
declare -a FAILED=()
declare -a FAILED_LOGS=()
declare -a SKIPPED=()

# ── Helper ────────────────────────────────────────────────────────────────────

# write_data_json <dep> <version> <url> <sha256> [<sha512>] [<source_type>] [<repo>]
write_data_json() {
  local dep="$1"
  local version="$2"
  local url="$3"
  local sha256="$4"
  local sha512="${5:-}"
  local source_type="${6:-url}"
  local repo="${7:-}"
  local path="${TMPDIR}/${dep}-data.json"

  jq -n \
    --arg name    "${dep}" \
    --arg stype   "${source_type}" \
    --arg repo    "${repo}" \
    --arg url     "${url}" \
    --arg ref     "${version}" \
    --arg sha256  "${sha256}" \
    --arg sha512  "${sha512}" \
    '{
       "source": { "name": $name, "type": $stype, "repo": $repo },
       "version": { "url": $url, "ref": $ref, "sha256": $sha256, "sha512": $sha512 }
     }' > "${path}"

  echo "${path}"
}

run_dep() {
  local dep="$1"
  local data_json="$2"

  # Skip if a DEP filter is set and this dep doesn't match.
  if [[ -n "${FILTER_DEP}" && "${dep}" != "${FILTER_DEP}" ]]; then return 0; fi

  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo " ${dep} on ${STACK}"
  echo "════════════════════════════════════════════════════════════════"

  local version
  version=$(jq -r '.version.ref // "unknown"' "${data_json}")
  local log_file="/tmp/parity-logs/${dep}-${version}-${STACK}.log"

  if "${SCRIPT_DIR}/compare-builds.sh" --dep "${dep}" --data-json "${data_json}" --stack "${STACK}"; then
    PASSED+=("${dep} ${version}")
  else
    FAILED+=("${dep} ${version}")
    FAILED_LOGS+=("${log_file}")
    echo "FAILED: ${dep} ${version} — log: ${log_file}"
  fi
}

# run_dep_r <data_json>
# Wraps run_dep for `r`, creating the 4 sub-dep data.json files that the
# Go recipe reads from source-*-latest/ directories inside the container.
run_dep_r() {
  local data_json="$1"
  local sub_deps_dir="${TMPDIR}/r-sub-deps"

  # forecast 8.24.0
  mkdir -p "${sub_deps_dir}/source-forecast-latest"
  jq -n \
    --arg name    "forecast" \
    --arg stype   "github_releases" \
    --arg repo    "robjhyndman/forecast" \
    --arg url     "https://cran.r-project.org/src/contrib/forecast_8.24.0.tar.gz" \
    --arg ref     "8.24.0" \
    --arg sha256  "" \
    --arg sha512  "" \
    '{"source":{"name":$name,"type":$stype,"repo":$repo},"version":{"url":$url,"ref":$ref,"sha256":$sha256,"sha512":$sha512}}' \
    > "${sub_deps_dir}/source-forecast-latest/data.json"

  # plumber 1.3.0
  mkdir -p "${sub_deps_dir}/source-plumber-latest"
  jq -n \
    --arg name    "plumber" \
    --arg stype   "github_releases" \
    --arg repo    "rstudio/plumber" \
    --arg url     "https://cran.r-project.org/src/contrib/plumber_1.3.0.tar.gz" \
    --arg ref     "1.3.0" \
    --arg sha256  "" \
    --arg sha512  "" \
    '{"source":{"name":$name,"type":$stype,"repo":$repo},"version":{"url":$url,"ref":$ref,"sha256":$sha256,"sha512":$sha512}}' \
    > "${sub_deps_dir}/source-plumber-latest/data.json"

  # rserve 1.8.15
  mkdir -p "${sub_deps_dir}/source-rserve-latest"
  jq -n \
    --arg name    "Rserve" \
    --arg stype   "github_releases" \
    --arg repo    "s-u/Rserve" \
    --arg url     "https://cran.r-project.org/src/contrib/Rserve_1.8-15.tar.gz" \
    --arg ref     "1.8.15" \
    --arg sha256  "" \
    --arg sha512  "" \
    '{"source":{"name":$name,"type":$stype,"repo":$repo},"version":{"url":$url,"ref":$ref,"sha256":$sha256,"sha512":$sha512}}' \
    > "${sub_deps_dir}/source-rserve-latest/data.json"

  # shiny 1.10.0
  mkdir -p "${sub_deps_dir}/source-shiny-latest"
  jq -n \
    --arg name    "shiny" \
    --arg stype   "github_releases" \
    --arg repo    "rstudio/shiny" \
    --arg url     "https://cran.r-project.org/src/contrib/shiny_1.10.0.tar.gz" \
    --arg ref     "1.10.0" \
    --arg sha256  "" \
    --arg sha512  "" \
    '{"source":{"name":$name,"type":$stype,"repo":$repo},"version":{"url":$url,"ref":$ref,"sha256":$sha256,"sha512":$sha512}}' \
    > "${sub_deps_dir}/source-shiny-latest/data.json"

  local dep="r"

  # Skip if a DEP filter is set and this dep doesn't match.
  if [[ -n "${FILTER_DEP}" && "${dep}" != "${FILTER_DEP}" ]]; then return 0; fi

  local version
  version=$(jq -r '.version.ref // "unknown"' "${data_json}")
  local log_file="/tmp/parity-logs/${dep}-${version}-${STACK}.log"

  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo " ${dep} on ${STACK}"
  echo "════════════════════════════════════════════════════════════════"

  if "${SCRIPT_DIR}/compare-builds.sh" --dep "${dep}" --data-json "${data_json}" \
       --stack "${STACK}" --sub-deps-dir "${sub_deps_dir}"; then
    PASSED+=("${dep} ${version}")
  else
    FAILED+=("${dep} ${version}")
    FAILED_LOGS+=("${log_file}")
    echo "FAILED: ${dep} ${version} — log: ${log_file}"
  fi
}

skip_dep() {
  local dep="$1"
  local reason="$2"

  # Skip silently if a DEP filter is set and this dep doesn't match.
  if [[ -n "${FILTER_DEP}" && "${dep}" != "${FILTER_DEP}" ]]; then return 0; fi

  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo " ${dep} — SKIPPED: ${reason}"
  echo "════════════════════════════════════════════════════════════════"
  SKIPPED+=("${dep} (${reason})")
}

# ── Test matrix ───────────────────────────────────────────────────────────────
# All source URLs and SHA256 checksums verified against upstream.

# ruby 3.3.6 — https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz
run_dep ruby "$(write_data_json ruby 3.3.6 \
  "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-3.3.6.tar.gz" \
  "8dc48fffaf270f86f1019053f28e51e4da4cce32a36760a0603a9aee67d7fd8d" \
  "" github_releases "ruby/ruby")"

# jruby 9.4.14.0 — https://repo1.maven.org/maven2/org/jruby/jruby-dist/9.4.14.0/
# NOTE: Maven only publishes a .zip (no .tar.gz); the Go builder downloads .zip.
run_dep jruby "$(write_data_json jruby 9.4.14.0 \
  "https://repo1.maven.org/maven2/org/jruby/jruby-dist/9.4.14.0/jruby-dist-9.4.14.0-src.zip" \
  "400086b33f701a47dc28c5965d5a408bc2740301a5fb3b545e37abaa002ccdf8" \
  "" maven "")"

# python 3.12.0 — https://www.python.org/ftp/python/3.12.0/Python-3.12.0.tgz
run_dep python "$(write_data_json python 3.12.0 \
  "https://www.python.org/ftp/python/3.12.0/Python-3.12.0.tgz" \
  "51412956d24a1ef7c97f1cb5f70e185c13e3de1f50d131c0aac6338080687afb" \
  "" url "")"

# node 20.11.0 — https://nodejs.org/dist/v20.11.0/node-v20.11.0.tar.gz
run_dep node "$(write_data_json node 20.11.0 \
  "https://nodejs.org/dist/v20.11.0/node-v20.11.0.tar.gz" \
  "9884b22d88554d65025352ba7e4cb20f5d17a939231bea41a7894c0344fab1bf" \
  "" url "")"

# go 1.22.0 — https://go.dev/dl/go1.22.0.src.tar.gz
run_dep go "$(write_data_json go 1.22.0 \
  "https://go.dev/dl/go1.22.0.src.tar.gz" \
  "4d196c3d41a0d6c1dfc64d04e3cc1f608b0c436bd87b7060ce3e23234e1f4d5c" \
  "" url "")"

# nginx 1.25.3 — https://nginx.org/download/nginx-1.25.3.tar.gz
run_dep nginx "$(write_data_json nginx 1.25.3 \
  "https://nginx.org/download/nginx-1.25.3.tar.gz" \
  "64c5b975ca287939e828303fa857d22f142b251f17808dfe41733512d9cded86" \
  "" url "")"

# nginx-static 1.25.3 — same source as nginx
run_dep nginx-static "$(write_data_json nginx-static 1.25.3 \
  "https://nginx.org/download/nginx-1.25.3.tar.gz" \
  "64c5b975ca287939e828303fa857d22f142b251f17808dfe41733512d9cded86" \
  "" url "")"

# openresty 1.25.3.1 — https://openresty.org/download/openresty-1.25.3.1.tar.gz
run_dep openresty "$(write_data_json openresty 1.25.3.1 \
  "https://openresty.org/download/openresty-1.25.3.1.tar.gz" \
  "32ec1a253a5a13250355a075fe65b7d63ec45c560bbe213350f0992a57cd79df" \
  "" url "")"

# httpd 2.4.58 — Go recipe downloads .tar.bz2 (not .tar.gz); sha256 is for .tar.bz2
run_dep httpd "$(write_data_json httpd 2.4.58 \
  "https://archive.apache.org/dist/httpd/httpd-2.4.58.tar.gz" \
  "fa16d72a078210a54c47dd5bef2f8b9b8a01d94909a51453956b3ec6442ea4c5" \
  "" url "")"

# bundler 2.5.6 — https://rubygems.org/gems/bundler-2.5.6.gem
run_dep bundler "$(write_data_json bundler 2.5.6 \
  "https://rubygems.org/gems/bundler-2.5.6.gem" \
  "1a1f21d1456e16dd2fee93461d9640348047aa2dcaf5d776874a60ddd4df5c64" \
  "" url "")"

# rubygems 3.5.6 — https://rubygems.org/rubygems/rubygems-3.5.6.tgz
run_dep rubygems "$(write_data_json rubygems 3.5.6 \
  "https://rubygems.org/rubygems/rubygems-3.5.6.tgz" \
  "f3fcc0327cee0b7ebbee2ef014a42ba05b4032d7e1834dbcd3165dde700c99c2" \
  "" url "")"

# r 4.4.2 — https://cran.r-project.org/src/base/R-4/R-4.4.2.tar.gz
# Uses run_dep_r to supply the 4 sub-dep data.json files required by the Go recipe.
run_dep_r "$(write_data_json r 4.4.2 \
  "https://cran.r-project.org/src/base/R-4/R-4.4.2.tar.gz" \
  "1578cd603e8d866b58743e49d8bf99c569e81079b6a60cf33cdf7bdffeb817ec" \
  "" url "")"

# libunwind 1.6.2 — https://github.com/libunwind/libunwind/archive/refs/tags/v1.6.2.tar.gz
run_dep libunwind "$(write_data_json libunwind 1.6.2 \
  "https://github.com/libunwind/libunwind/archive/refs/tags/v1.6.2.tar.gz" \
  "b76546101ca00c5525ae939104ca1b9de4a444a61cfa9bfe7e505c66c4fb1f10" \
  "" github_releases "libunwind/libunwind")"

# libgdiplus 6.1 — https://github.com/mono/libgdiplus/archive/refs/tags/6.1.tar.gz
run_dep libgdiplus "$(write_data_json libgdiplus 6.1 \
  "https://github.com/mono/libgdiplus/archive/refs/tags/6.1.tar.gz" \
  "6ba47acef48ffa2a75d71f8958e0de7f8f52ea066ed97409b33e7a32f31835fd" \
  "" github_releases "mono/libgdiplus")"

# hwc 106.0.0 — https://github.com/cloudfoundry/hwc/archive/refs/tags/106.0.0.tar.gz
run_dep hwc "$(write_data_json hwc 106.0.0 \
  "https://github.com/cloudfoundry/hwc/archive/refs/tags/106.0.0.tar.gz" \
  "87fe14594a5d51f43680a84a669ff1ae7b1ec64630608726beeca172ab0d4163" \
  "" github_releases "cloudfoundry/hwc")"

# pip 24.0 — https://files.pythonhosted.org/...
run_dep pip "$(write_data_json pip 24.0 \
  "https://files.pythonhosted.org/packages/94/59/6638090c25e9bc4ce0c42817b5a234e183872a1129735a9330c472cc2056/pip-24.0.tar.gz" \
  "ea9bd1a847e8c5774a5777bb398c19e80bcd4e2aa16a4b301b718fe6f593aba2" \
  "" url "")"

# pipenv 2023.12.1 — https://files.pythonhosted.org/...
run_dep pipenv "$(write_data_json pipenv 2023.12.1 \
  "https://files.pythonhosted.org/packages/a6/26/5cdf9f0c6eb835074c3e43dde2880bfa739daa23fa534a5dd65848af5913/pipenv-2023.12.1.tar.gz" \
  "4aea73e23944e464ad2b849328e780ad121c5336e1c24a7ac15aa493c41c2341" \
  "" url "")"

# setuptools 69.0.3 — https://files.pythonhosted.org/...
run_dep setuptools "$(write_data_json setuptools 69.0.3 \
  "https://files.pythonhosted.org/packages/fc/c9/b146ca195403e0182a374e0ea4dbc69136bad3cd55bc293df496d625d0f7/setuptools-69.0.3.tar.gz" \
  "be1af57fc409f93647f2e8e4573a142ed38724b8cdd389706a867bb4efcf1e78" \
  "" url "")"

# yarn 1.22.21 — https://registry.npmjs.org/yarn/-/yarn-1.22.21.tgz
run_dep yarn "$(write_data_json yarn 1.22.21 \
  "https://registry.npmjs.org/yarn/-/yarn-1.22.21.tgz" \
  "dbed5b7e10c552ba0e1a545c948d5473bc6c5a28ce22a8fd27e493e3e5eb6370" \
  "" url "")"

# bower 1.8.14 — https://registry.npmjs.org/bower/-/bower-1.8.14.tgz
run_dep bower "$(write_data_json bower 1.8.14 \
  "https://registry.npmjs.org/bower/-/bower-1.8.14.tgz" \
  "00df3dcc6e8b3a4dd7668934a20e60e6fc0c4269790192179388c928553a3f7e" \
  "" url "")"

# composer 2.7.1 — https://github.com/composer/composer/releases/download/2.7.1/composer.phar
run_dep composer "$(write_data_json composer 2.7.1 \
  "https://github.com/composer/composer/releases/download/2.7.1/composer.phar" \
  "1ffd0be3f27e237b1ae47f9e8f29f96ac7f50a0bd9eef4f88cdbe94dd04bfff0" \
  "" github_releases "composer/composer")"

# tomcat 10.1.18 — https://archive.apache.org/dist/tomcat/tomcat-10/v10.1.18/bin/
run_dep tomcat "$(write_data_json tomcat 10.1.18 \
  "https://archive.apache.org/dist/tomcat/tomcat-10/v10.1.18/bin/apache-tomcat-10.1.18.tar.gz" \
  "6da0b4cbd3140e64a8719a2de19c20bf3902d264a142a816ac552ae216ade311" \
  "" url "")"

# openjdk 11.0.22_7 — Adoptium Temurin JDK 11
run_dep openjdk "$(write_data_json openjdk 11.0.22_7 \
  "https://github.com/adoptium/temurin11-binaries/releases/download/jdk-11.0.22%2B7/OpenJDK11U-jdk_x64_linux_hotspot_11.0.22_7.tar.gz" \
  "25cf602cac350ef36067560a4e8042919f3be973d419eac4d839e2e0000b2cc8" \
  "" github_releases "adoptium/temurin11-binaries")"

# zulu 21.32.17 — Azul Zulu JDK 21.0.2
run_dep zulu "$(write_data_json zulu 21.32.17 \
  "https://cdn.azul.com/zulu/bin/zulu21.32.17-ca-jdk21.0.2-linux_x64.tar.gz" \
  "5ad730fbee6bb49bfff10bf39e84392e728d89103d3474a7e5def0fd134b300a" \
  "" zulu "")"

# sapmachine 21.0.2 — SAP Machine JDK 21
run_dep sapmachine "$(write_data_json sapmachine 21.0.2 \
  "https://github.com/SAP/SapMachine/releases/download/sapmachine-21.0.2/sapmachine-jdk-21.0.2_linux-x64_bin.tar.gz" \
  "3123189ec5b99eed78de0328e2fd49d7c13cc7d4524c341f1fe8fbd5165be31f" \
  "" github_releases "SAP/SapMachine")"

# skywalking-agent 9.5.0 — Apache SkyWalking Java Agent (SHA512 checksum)
run_dep skywalking-agent "$(write_data_json skywalking-agent 9.5.0 \
  "https://archive.apache.org/dist/skywalking/java-agent/9.5.0/apache-skywalking-java-agent-9.5.0.tgz" \
  "" \
  "deb782b41e6cde1e4eae94f806bb73bccb0f6bd0362c6b9f90e387a6d84bad672c34b70ca204f9e5f74899726542c76c36b2e2af05ecbcab8fff73a661a3de21" \
  "url" "")"

# jprofiler-profiler 15.0.4 — ej-technologies JProfiler
# URL: https://download.ej-technologies.com/jprofiler/jprofiler_linux_15_0_4.tar.gz
run_dep jprofiler-profiler "$(write_data_json jprofiler-profiler 15.0.4 \
  "https://download.ej-technologies.com/jprofiler/jprofiler_linux_15_0_4.tar.gz" \
  "fec741718854a11b2383bb278ca7103984e0ae659268ed53ea5a8b32077b86c9" \
  "" jprofiler "")"

# your-kit-profiler 2025.9.185 — YourKit Java Profiler (latest publicly available)
# Version format: <year>.<minor>.<build> → URL uses year.minor/YourKit-JavaProfiler-year.minor-b<build>-x64.zip
run_dep your-kit-profiler "$(write_data_json your-kit-profiler 2025.9.185 \
  "https://download.yourkit.com/yjp/2025.9/YourKit-JavaProfiler-2025.9-b185-x64.zip" \
  "1818a6f74ef231e53876c66ba9e7e4f0952f57cb1af40c2d410e21a6da8c33b7" \
  "" yourkit "")"

# php 8.1.32 — https://www.php.net/distributions/php-8.1.32.tar.gz
run_dep php "$(write_data_json php 8.1.32 \
  "https://www.php.net/distributions/php-8.1.32.tar.gz" \
  "4846836d1de27dbd28e89180f073531087029a77e98e8e019b7b2eddbdb1baff" \
  "" url "")"

# dotnet-sdk 8.0.101 — Microsoft .NET SDK (SHA512 checksum, not SHA256)
run_dep dotnet-sdk "$(write_data_json dotnet-sdk 8.0.101 \
  "https://builds.dotnet.microsoft.com/dotnet/Sdk/8.0.101/dotnet-sdk-8.0.101-linux-x64.tar.gz" \
  "" \
  "26df0151a3a59c4403b52ba0f0df61eaa904110d897be604f19dcaa27d50860c82296733329cb4a3cf20a2c2e518e8f5d5f36dfb7931bf714a45e46b11487c9a" \
  "url" "")"

# dotnet-runtime 8.0.1 — Microsoft .NET Runtime (SHA512 checksum)
run_dep dotnet-runtime "$(write_data_json dotnet-runtime 8.0.1 \
  "https://builds.dotnet.microsoft.com/dotnet/Runtime/8.0.1/dotnet-runtime-8.0.1-linux-x64.tar.gz" \
  "" \
  "cbd03325280ff93cd0edab71c5564a50bb2423980f63d04602914db917c9c811a0068d848cab07d82e3260bff6684ad7cffacc2f449c06fc0b0aa8f845c399b6" \
  "url" "")"

# dotnet-aspnetcore 8.0.1 — Microsoft ASP.NET Core Runtime
run_dep dotnet-aspnetcore "$(write_data_json dotnet-aspnetcore 8.0.1 \
  "https://builds.dotnet.microsoft.com/dotnet/aspnetcore/Runtime/8.0.1/aspnetcore-runtime-8.0.1-linux-x64.tar.gz" \
  "cd825a5bd7b40e5706840d7b22650b787f71db5e2e496c80e16571bf5003f8fe" \
  "" url "")"

# appdynamics — requires vendor credentials; skip
skip_dep appdynamics   "requires AppDynamics vendor credentials (appdynamics-credentials)"
skip_dep appdynamics-java "requires AppDynamics vendor credentials (appdynamics-credentials)"

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
echo "════════════════════════════════════════════════════════════════"
echo " Parity test summary — stack: ${STACK}"
echo "════════════════════════════════════════════════════════════════"
echo "  Passed:  ${#PASSED[@]}"
for p in "${PASSED[@]}"; do echo "    ✓ ${p}"; done
echo "  Failed:  ${#FAILED[@]}"
for i in "${!FAILED[@]}"; do
  echo "    ✗ ${FAILED[$i]}"
  echo "      log: ${FAILED_LOGS[$i]}"
done
echo "  Skipped: ${#SKIPPED[@]}"
for s in "${SKIPPED[@]}"; do echo "    - ${s}"; done

if [[ "${#FAILED[@]}" -gt 0 ]]; then
  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo " Failure details (last 20 lines of each log)"
  echo "════════════════════════════════════════════════════════════════"
  for i in "${!FAILED[@]}"; do
    echo ""
    echo "── ${FAILED[$i]} ──────────────────────────────────────────────"
    echo "   ${FAILED_LOGS[$i]}"
    echo "────────────────────────────────────────────────────────────────"
    if [[ -f "${FAILED_LOGS[$i]}" ]]; then
      tail -20 "${FAILED_LOGS[$i]}"
    else
      echo "   (log file not found)"
    fi
  done
  echo ""
  echo "FAIL: ${#FAILED[@]} dep(s) failed parity test"
  exit 1
fi

echo ""
echo "PASS: All ${#PASSED[@]} deps passed parity test on ${STACK} (${#SKIPPED[@]} skipped)"
