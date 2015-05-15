#!/bin/sh
set +e

tar_name=$1
exec_path=$2
exec_flag=$3
exec_test=$4

mkdir binary-exerciser
cd binary-exerciser

tar xzf /binary-builder/${tar_name}
${exec_path} $exec_flag "${exec_test}"
