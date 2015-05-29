#!/usr/bin/env bash
set +e

tar_name=$1; shift

mkdir binary-exerciser
cd binary-exerciser

tar xzf /binary-builder/${tar_name}
eval $(printf '%q ' "$@")
