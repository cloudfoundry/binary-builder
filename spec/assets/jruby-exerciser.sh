#!/bin/sh
set +e

mkdir /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

wget https://pivotal-buildpacks.s3.amazonaws.com/ruby/binaries/cflinuxfs2/openjdk1.8-latest.tar.gz
mkdir -p openjdk
tar xzf openjdk1.8-latest.tar.gz -C openjdk
export PATH=/tmp/binary-exerciser/openjdk/bin:$PATH

tar xzf $current_dir/jruby-9.0.0.0.pre1_ruby-2.2.0-linux-x64.tgz
./bin/jruby -e 'puts "#{RUBY_PLATFORM} #{RUBY_VERSION}"'
