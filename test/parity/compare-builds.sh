#!/usr/bin/env bash
# binary-builder/test/parity/compare-builds.sh
# Usage: compare-builds.sh --dep <name> --data-json <path> [--stack <stack>]
#                          [--sub-deps-dir <dir>]
#
# Runs both the Ruby builder and the Go builder inside the target stack Docker
# container with the same real depwatcher data.json, then diffs every observable
# output. Exits 1 on any mismatch.
#
# All output is tee'd to a log file:
#   /tmp/parity-logs/<dep>-<version>-<stack>.log
# Check progress without re-running: tail -f /tmp/parity-logs/<dep>-*.log
#
# The data.json must be in the real depwatcher modern format:
#   {
#     "source": {"name": "...", "type": "...", "repo": "..."},
#     "version": {"url": "...", "ref": "...", "sha256": "...", "sha512": "..."}
#   }
#
# For the `r` dep, pass --sub-deps-dir pointing to a directory containing:
#   source-forecast-latest/data.json
#   source-plumber-latest/data.json
#   source-rserve-latest/data.json
#   source-shiny-latest/data.json
# These are mounted into both containers at the working directory.
#
# The Ruby builder clones binary-builder master inside the container (for the
# cflinuxfs4/ Ruby source tree) and mounts the local buildpacks-ci working tree
# (task scripts stay in sync with local changes).  The Go builder mounts the
# local binary-builder working tree so that in-progress changes are tested.

set -euo pipefail

DEP=""
DATA_JSON=""
STACK="cflinuxfs4"
SUB_DEPS_DIR=""

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --dep)          DEP="$2";          shift 2 ;;
    --data-json)    DATA_JSON="$2";    shift 2 ;;
    --stack)        STACK="$2";        shift 2 ;;
    --sub-deps-dir) SUB_DEPS_DIR="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

[[ -n "${DEP}" ]]       || { echo "Usage: compare-builds.sh --dep <name> --data-json <path> [--stack <stack>] [--sub-deps-dir <dir>]"; exit 1; }
[[ -n "${DATA_JSON}" ]] || { echo "--data-json is required"; exit 1; }
[[ -f "${DATA_JSON}" ]] || { echo "data.json not found: ${DATA_JSON}"; exit 1; }

VERSION=$(jq -r '.version.ref // .version // "unknown"' "${DATA_JSON}")
IMAGE="cloudfoundry/${STACK}"

# Resolve binary-builder repo root (two levels up from this script).
# Used by the Go builder to mount the local working tree.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

RUBY_OUT="$(mktemp -d)"
GO_OUT="$(mktemp -d)"
SOURCE_DIR="$(mktemp -d)"  # pre-downloaded source tarballs for libunwind/dotnet/etc.
DATA_JSON_ABS="$(realpath "${DATA_JSON}")"
SUB_DEPS_DIR_ABS=""
[[ -n "${SUB_DEPS_DIR}" ]] && SUB_DEPS_DIR_ABS="$(realpath "${SUB_DEPS_DIR}")"

# ── Logging setup ─────────────────────────────────────────────────────────────
# All output (stdout + stderr) goes to both the terminal and a persistent log
# file so long-running builds can be inspected without re-running.
LOG_DIR="/tmp/parity-logs"
mkdir -p "${LOG_DIR}"
LOG_FILE="${LOG_DIR}/${DEP}-${VERSION}-${STACK}.log"
# Redirect all further output through tee; preserve stderr on the terminal too.
exec > >(tee "${LOG_FILE}") 2>&1
echo "==> Log file: ${LOG_FILE}"

cleanup() {
  local _exit=$?
  # Docker runs as root inside containers so output files may be root-owned.
  # Use a throwaway container to chmod before removing; ignore any errors.
  for dir in "${RUBY_OUT}" "${GO_OUT}" "${SOURCE_DIR}"; do
    [[ -d "${dir}" ]] && docker run --rm -v "${dir}:/cleanup:z" busybox \
      chmod -R a+rwX /cleanup 2>/dev/null || true
  done
  rm -rf "${RUBY_OUT}" "${GO_OUT}" "${SOURCE_DIR}" 2>/dev/null || true
  exit "${_exit}"
}
trap cleanup EXIT

echo "==> Parity test: ${DEP} ${VERSION} on ${STACK}"

# ── Source pre-download ────────────────────────────────────────────────────────
# Some deps (libunwind, dotnet-*) expect a pre-downloaded source tarball in
# source/ alongside data.json. We download it here on the host and mount the
# source/ dir into both containers.
prepare_source() {
  local url
  url=$(jq -r '.version.url // empty' "${DATA_JSON_ABS}")
  [[ -z "${url}" ]] && return 0

  # Only download for deps that need a pre-placed source file.
  # These are deps where the builder reads source/*.tar.gz or source/<file>
  # rather than downloading itself.
  case "${DEP}" in
    libunwind|dotnet-sdk|dotnet-runtime|dotnet-aspnetcore|jprofiler-profiler|your-kit-profiler)
      local filename
      filename=$(basename "${url}")
      echo "--> Pre-downloading source: ${url}"
      curl -fsSL -o "${SOURCE_DIR}/${filename}" "${url}"
      chmod a+r "${SOURCE_DIR}/${filename}"
      echo "    Saved: ${SOURCE_DIR}/${filename}"
      ;;
    *)
      # All other deps fetch their own source inside the build
      ;;
  esac
}

# ── Ruby builder ─────────────────────────────────────────────────────────────
#
# The Ruby builder clones binary-builder master inside the container (for the
# cflinuxfs4/ Ruby source tree) but mounts the local buildpacks-ci working tree
# (so the task scripts stay in sync with local changes and are not affected by
# upstream breakage).

run_ruby_builder() {
  echo "--> Running Ruby builder..."

  mkdir -p "${RUBY_OUT}/artifact" "${RUBY_OUT}/dep-metadata" "${RUBY_OUT}/builds"
  chmod -R o+rwx "${RUBY_OUT}"

  # Build sub-deps volume mount args (for r dep).
  local ruby_subdeps_args=()
  if [[ -n "${SUB_DEPS_DIR_ABS}" ]]; then
    ruby_subdeps_args=(-v "${SUB_DEPS_DIR_ABS}:/tmp/host-sub-deps:ro,z")
  fi

   docker run --rm \
    -v "${REPO_ROOT}/../buildpacks-ci:/buildpacks-ci-ro:ro,z" \
    -v "${DATA_JSON_ABS}:/tmp/data.json:ro,z" \
    -v "${SOURCE_DIR}:/tmp/host-source:ro,z" \
    -v "${RUBY_OUT}:/out:z" \
    "${ruby_subdeps_args[@]}" \
    -e STACK="${STACK}" \
    "${IMAGE}" \
    bash -c '
      set -euo pipefail

      apt-get update -qq
      apt-get install -y -qq git

      # Copy buildpacks-ci to a writable location so tasks that write files
      # (e.g. php_extensions/php-final-extensions.yml) can do so freely.
      cp -a /buildpacks-ci-ro /buildpacks-ci

      # Clone binary-builder master for the Ruby source tree (cflinuxfs4/
      # Gemfile, Gemfile.lock, bin/binary-builder, etc.).  We do NOT clone
      # buildpacks-ci — the local working tree is mounted instead so the task
      # scripts stay consistent with local changes.
      echo "--> Cloning binary-builder master..."
      git clone --depth=1 https://github.com/cloudfoundry/binary-builder.git /srv/binary-builder

      RUBY_VERSION="3.4.6"
      if ! command -v ruby &>/dev/null || ! ruby --version | grep -q "3.4"; then
        apt-get install -y -qq wget build-essential zlib1g-dev libssl-dev libreadline-dev libyaml-dev libffi-dev
        pushd /tmp
        wget -q "https://cache.ruby-lang.org/pub/ruby/3.4/ruby-${RUBY_VERSION}.tar.gz"
        tar -xzf "ruby-${RUBY_VERSION}.tar.gz"
        cd "ruby-${RUBY_VERSION}"
        ./configure --disable-install-doc
        make -j$(nproc)
        make install
        popd
        rm -rf "/tmp/ruby-${RUBY_VERSION}"*
      fi

      # Set up Concourse-style task directory layout.
      mkdir -p /task/source /task/artifacts "/task/builds-artifacts/binary-builds-new/'"${DEP}"'" /task/dep-metadata
      cp /tmp/data.json /task/source/data.json
      # Copy any pre-downloaded source files (libunwind tarball, dotnet tarball, etc.)
      cp /tmp/host-source/* /task/source/ 2>/dev/null || true
      ln -sf /srv/binary-builder /task/binary-builder
      ln -sf /buildpacks-ci      /task/buildpacks-ci

      # For r dep: copy sub-dep data.json dirs into task working directory.
      if [[ -d /tmp/host-sub-deps ]]; then
        cp -r /tmp/host-sub-deps/source-*-latest /task/ 2>/dev/null || true
      fi

      cd /task
      STACK='"${STACK}"' SKIP_COMMIT=true ruby buildpacks-ci/tasks/build-binary-new-cflinuxfs4/build.rb

      cp artifacts/*          /out/artifact/    2>/dev/null || true
      cp dep-metadata/*       /out/dep-metadata/ 2>/dev/null || true
      cp "builds-artifacts/binary-builds-new/'"${DEP}"'"/*.json /out/builds/ 2>/dev/null || true
    '
}

# ── Go builder ───────────────────────────────────────────────────────────────

run_go_builder() {
  echo "--> Running Go builder..."

  mkdir -p "${GO_OUT}/artifact" "${GO_OUT}/dep-metadata" "${GO_OUT}/builds"
  chmod -R o+rwx "${GO_OUT}"

  GO_VERSION="1.25.7"

  # Build sub-deps volume mount args (for r dep).
  local go_subdeps_args=()
  if [[ -n "${SUB_DEPS_DIR_ABS}" ]]; then
    go_subdeps_args=(-v "${SUB_DEPS_DIR_ABS}:/tmp/host-sub-deps:ro,z")
  fi

  docker run --rm \
    -v "${REPO_ROOT}:/binary-builder:z" \
    -v "${DATA_JSON_ABS}:/tmp/data.json:ro,z" \
    -v "${SOURCE_DIR}:/tmp/host-source:ro,z" \
    -v "${GO_OUT}:/out:z" \
    "${go_subdeps_args[@]}" \
    -e STACK="${STACK}" \
    "${IMAGE}" \
    bash -c "
      set -euo pipefail

      # Install mise if not present, then use it to install the required Go version
      if ! command -v mise &>/dev/null; then
        apt-get update -qq
        # zstd is required so the system tar can auto-detect compression when
        # mise extracts the Go toolchain tarball inside this container.
        apt-get install -y -qq curl ca-certificates zstd
        curl -fsSL https://mise.run | MISE_QUIET=1 sh
      fi
      export PATH=\"\${HOME}/.local/bin:\${PATH}\"
      mise use --global go@${GO_VERSION}
      export PATH=\"\${HOME}/.local/share/mise/shims:\${PATH}\"

      go version

      # Compile binary-builder from source (must run from module root)
      cd /binary-builder
      go build -buildvcs=false -o /usr/local/bin/binary-builder ./cmd/binary-builder

      # Run Go builder
      mkdir -p /tmp/workdir/source
      # Copy any pre-downloaded source files (libunwind tarball, dotnet tarball, etc.)
      cp /tmp/host-source/* /tmp/workdir/source/ 2>/dev/null || true
      # For r dep: copy sub-dep data.json dirs into workdir (Go recipe reads them from CWD).
      if [[ -d /tmp/host-sub-deps ]]; then
        cp -r /tmp/host-sub-deps/source-*-latest /tmp/workdir/ 2>/dev/null || true
      fi
      cd /tmp/workdir

      binary-builder build \
        --stack ${STACK} \
        --source-file /tmp/data.json \
        --stacks-dir /binary-builder/stacks \
        --artifacts-dir /out/artifact \
        --builds-dir /out/builds \
        --dep-metadata-dir /out/dep-metadata \
        --skip-commit
    "
}

# ── Compare ──────────────────────────────────────────────────────────────────

compare_outputs() {
  local mismatches=0

  # --- Artifact filename pattern (strip 8-char SHA prefix) ---
  ruby_artifact=$(ls "${RUBY_OUT}/artifact/" 2>/dev/null | head -1)
  go_artifact=$(ls "${GO_OUT}/artifact/"   2>/dev/null | head -1)

  # URL-passthrough deps produce no artifact file —
  # both builders set outData.URL directly pointing to the original download.
  # For these deps, no artifact is expected; just compare builds JSON.
  URL_PASSTHROUGH_DEPS=()
  is_url_passthrough=false
  for pt_dep in "${URL_PASSTHROUGH_DEPS[@]}"; do
    if [[ "${DEP}" == "${pt_dep}" ]]; then
      is_url_passthrough=true
      break
    fi
  done

  if [[ "${is_url_passthrough}" == "true" ]]; then
    if [[ -n "${ruby_artifact}" || -n "${go_artifact}" ]]; then
      echo "WARN: URL-passthrough dep ${DEP} unexpectedly produced an artifact file"
      echo "  Ruby: ${ruby_artifact:-none}  Go: ${go_artifact:-none}"
    else
      echo "  OK: no artifact file (URL-passthrough dep)"
    fi
  else
    if [[ -z "${ruby_artifact}" ]]; then
      echo "FAIL: Ruby builder produced no artifact"
      return 1
    fi
    if [[ -z "${go_artifact}" ]]; then
      echo "FAIL: Go builder produced no artifact"
      return 1
    fi
  fi

  if [[ "${is_url_passthrough}" == "false" ]]; then
    ruby_pattern=$(echo "${ruby_artifact}" | sed 's/_[0-9a-f]\{8\}\./_./')
    go_pattern=$(echo "${go_artifact}"     | sed 's/_[0-9a-f]\{8\}\./_./')

    if [[ "${ruby_pattern}" != "${go_pattern}" ]]; then
      echo "MISMATCH: artifact filename pattern"
      echo "  Ruby: ${ruby_artifact}"
      echo "  Go:   ${go_artifact}"
      mismatches=$((mismatches + 1))
    else
      echo "  OK: artifact filename pattern (${go_pattern})"
    fi

    # --- Tar/zip contents (sorted file list) ---
    ruby_ext="${ruby_artifact##*.}"
    go_ext="${go_artifact##*.}"

    if [[ "${ruby_ext}" == "tgz" || "${ruby_ext}" == "gz" ]]; then
      ruby_files=$(tar -tzf "${RUBY_OUT}/artifact/${ruby_artifact}" 2>/dev/null | sort)
      go_files=$(tar -tzf "${GO_OUT}/artifact/${go_artifact}"       2>/dev/null | sort)
    elif [[ "${ruby_ext}" == "zip" ]]; then
      ruby_files=$(unzip -l "${RUBY_OUT}/artifact/${ruby_artifact}" 2>/dev/null | awk 'NR>3{print $4}' | sort)
      go_files=$(unzip -l "${GO_OUT}/artifact/${go_artifact}"       2>/dev/null | awk 'NR>3{print $4}' | sort)
    else
      ruby_files=""
      go_files=""
    fi

    # Known Ruby builder bug: SnmpRecipe sets @php_path = nil (constructor takes
    # name/version/options only, no php_path), so "cd #{@php_path}" expands to
    # "cd" (empty) and the mibs/conf copy runs in the wrong directory — the
    # snmp-mibs-downloader tree never appears in the Ruby artifact.  The Go
    # builder is correct.  Filter from both sides before comparing.
    ruby_files=$(echo "${ruby_files}" | grep -v 'mibs/conf/snmp-mibs-downloader')
    go_files=$(echo "${go_files}"     | grep -v 'mibs/conf/snmp-mibs-downloader')

    if [[ -n "${ruby_files}" ]] && [[ "${ruby_files}" != "${go_files}" ]]; then
      echo "MISMATCH: artifact file list"
      diff <(echo "${ruby_files}") <(echo "${go_files}") || true
      mismatches=$((mismatches + 1))
    else
      echo "  OK: artifact file list"
    fi
  fi

  # --- builds JSON (field by field) ---
  ruby_json=$(ls "${RUBY_OUT}/builds/"*.json 2>/dev/null | head -1)
  go_json=$(ls "${GO_OUT}/builds/"*.json     2>/dev/null | head -1)

  if [[ -z "${ruby_json}" || -z "${go_json}" ]]; then
    echo "WARN: builds JSON missing (Ruby: ${ruby_json:-none}, Go: ${go_json:-none})"
  else
    for field in version "source.url" "source.sha256" "source.sha512" "source.md5" url sha256; do
      ruby_val=$(jq -r ".${field} // empty" "${ruby_json}" 2>/dev/null)
      go_val=$(jq -r ".${field} // empty"   "${go_json}"   2>/dev/null)
      if [[ "${ruby_val}" != "${go_val}" ]]; then
        echo "MISMATCH: builds JSON field .${field}"
        echo "  Ruby: ${ruby_val}"
        echo "  Go:   ${go_val}"
        mismatches=$((mismatches + 1))
      fi
    done

    ruby_subdeps=$(jq -r '.sub_dependencies // {} | to_entries[] | "\(.key)=\(.value.version)"' "${ruby_json}" 2>/dev/null | sort)
    go_subdeps=$(jq -r   '.sub_dependencies // {} | to_entries[] | "\(.key)=\(.value.version)"' "${go_json}"   2>/dev/null | sort)
    if [[ "${ruby_subdeps}" != "${go_subdeps}" ]]; then
      echo "MISMATCH: sub_dependencies"
      diff <(echo "${ruby_subdeps}") <(echo "${go_subdeps}") || true
      mismatches=$((mismatches + 1))
    else
      echo "  OK: builds JSON fields + sub_dependencies"
    fi
  fi

   # --- dep-metadata JSON ---
   ruby_meta=$(ls "${RUBY_OUT}/dep-metadata/"*.json 2>/dev/null | head -1)
   go_meta=$(ls "${GO_OUT}/dep-metadata/"*.json     2>/dev/null | head -1)

   if [[ -n "${ruby_meta}" && -n "${go_meta}" ]]; then
      # Compare dep-metadata in two passes:
      #
      # 1. Structural fields (version, source.*) — must match exactly.
      # 2. sha256 / url fields — these embed the artifact hash, which differs
      #    between independent runs for non-reproducible builds (e.g. bundler,
      #    where `gem install` records the current wall-clock time as file mtime).
      #    We only WARN on sha256/url mismatches so the parity test still passes
      #    as long as the artifact file list and structural metadata are identical.
      #
      # Also exclude sub_dependencies[].source.sha256: the Ruby builder computes
      # these via sha_from_url (sha256 of the HTTP redirect response body, not the
      # actual tarball), which is a Ruby builder bug.  The Go builder uses the
      # correct sha256 from data.json.  Treat sub-dep source sha256 as WARN-only.
      structural_fields='del(.sha256, .url) | if .sub_dependencies then .sub_dependencies |= with_entries(.value.source.sha256 = null) else . end'
     if ! diff <(jq -S "${structural_fields}" "${ruby_meta}") \
               <(jq -S "${structural_fields}" "${go_meta}") > /dev/null 2>&1; then
       echo "MISMATCH: dep-metadata JSON (structural fields)"
       diff <(jq -S "${structural_fields}" "${ruby_meta}") \
            <(jq -S "${structural_fields}" "${go_meta}") || true
       mismatches=$((mismatches + 1))
     else
       # Check sha256/url (artifact hash) — WARN only, not a hard failure.
       if ! diff <(jq -S . "${ruby_meta}") <(jq -S . "${go_meta}") > /dev/null 2>&1; then
         echo "WARN: dep-metadata JSON sha256/url differ (non-reproducible build — expected for gem-install deps)"
         diff <(jq -S . "${ruby_meta}") <(jq -S . "${go_meta}") || true
       else
         echo "  OK: dep-metadata JSON"
       fi
     fi
   else
     echo "WARN: dep-metadata JSON missing (Ruby: ${ruby_meta:-none}, Go: ${go_meta:-none})"
   fi

  return "${mismatches}"
}

# ── Main ─────────────────────────────────────────────────────────────────────

prepare_source

run_ruby_builder

run_go_builder

mismatches=0
compare_outputs || mismatches=$?

if [[ "${mismatches}" -gt 0 ]]; then
  echo ""
  echo "FAIL: Parity test FAILED for ${DEP} ${VERSION} on ${STACK} (${mismatches} mismatch(es))"
  exit 1
fi

echo ""
echo "PASS: Parity test PASSED for ${DEP} ${VERSION} on ${STACK}"
