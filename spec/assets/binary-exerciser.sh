#!/bin/sh
set +e

tar_name=$1
executable_path=$2
executable_test=$3

mkdir binary-exerciser
cd binary-exerciser

tar xzf /binary-builder/${tar_name}
${executable_path} -e "${executable_test}"
