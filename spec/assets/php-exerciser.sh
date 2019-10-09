#!/usr/bin/env bash

set -e

tar_name=$1; shift
current_dir=$(pwd)
mkdir -p /tmp/binary-exerciser
cd /tmp/binary-exerciser

tar xzf "$current_dir/$tar_name"
export LD_LIBRARY_PATH="$PWD/php/lib"
eval "$(printf '%q ' "$@")"
