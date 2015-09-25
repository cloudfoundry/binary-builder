#!/usr/bin/env bash
set +e

php_version=$1; shift
tar_name=$1; shift
current_dir=`pwd`
mkdir -p /tmp/binary-exerciser
cd /tmp/binary-exerciser

tar xzf $current_dir/$tar_name -C .
eval $(printf '%q ' "$@")
