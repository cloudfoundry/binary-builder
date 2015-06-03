#!/usr/bin/env bash
set +e

php_version=$1; shift
tar_name=php-$php_version-linux-x64.tgz
mkdir /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

tar xzf $current_dir/${tar_name}
tar xzf php-$php_version/php-$php_version.tar.gz -C .
tar xzf php-$php_version/php-cli-$php_version.tar.gz -C .
eval $(printf '%q ' "$@")
