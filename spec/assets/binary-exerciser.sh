#!/usr/bin/env bash
set +e

tar_name=$1; shift

mkdir /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

tar xzf $current_dir/${tar_name}
eval $(printf '%q ' "$@")
