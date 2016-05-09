#!/usr/bin/env bash
set +e

tar_name=$1; shift

current_dir=`pwd`
tmpdir=$(mktemp -d /tmp/binary-builder.XXXXXXXX)
cd $tmpdir

tar xzf $current_dir/${tar_name} --touch

export GEM_HOME=$tmpdir
export GEM_PATH=$tmpdir

eval $(printf '%q ' "$@")
