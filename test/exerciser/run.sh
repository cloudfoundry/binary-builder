#!/usr/bin/env bash
# binary-builder/test/exerciser/run.sh
# Usage: run.sh <tarball-path> <stack> <command...>
#
# Extracts the tarball inside the target stack Docker container and runs
# the given command. Used to verify that a built artifact actually works.
#
# Examples:
#   ./run.sh /tmp/ruby_3.3.6_linux_x64_cflinuxfs4_e4311262.tgz cflinuxfs4 \
#     ./bin/ruby -e 'puts RUBY_VERSION'
#
#   ./run.sh /tmp/php_8.3.0_linux_x64_cflinuxfs4_abcd1234.tgz cflinuxfs4 \
#     bash -c 'LD_LIBRARY_PATH=$PWD/php/lib ./php/bin/php --version'

set -euo pipefail

TARBALL="${1:?tarball path required}"
STACK="${2:?stack required}"
shift 2

IMAGE="cloudfoundry/${STACK}"
TARBALL_ABS="$(realpath "${TARBALL}")"
TARBALL_NAME="$(basename "${TARBALL_ABS}")"

# Quote each remaining arg so the docker bash -c invocation is safe
CMD_ARGS=( "$@" )
CMD_QUOTED=""
for arg in "${CMD_ARGS[@]}"; do
  CMD_QUOTED="${CMD_QUOTED} $(printf '%q' "${arg}")"
done

docker run --rm \
  -v "${TARBALL_ABS}:/tmp/${TARBALL_NAME}" \
  "${IMAGE}" \
  bash -c "
    set -euo pipefail
    mkdir -p /tmp/exerciser
    cd /tmp/exerciser
    tar xzf /tmp/${TARBALL_NAME}
    ${CMD_QUOTED}
  "
