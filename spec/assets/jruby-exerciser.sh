#!/bin/sh
set +e

wget https://pivotal-buildpacks.s3.amazonaws.com/ruby/binaries/cflinuxfs2/openjdk1.8-latest.tar.gz
mkdir openjdk
tar xzf openjdk1.8-latest.tar.gz -C openjdk
export PATH=`pwd`/openjdk/bin:$PATH

mkdir binary-exerciser
cd binary-exerciser

tar xzf /binary-builder/jruby-ruby-2.2.0-jruby-9.0.0.0.pre1-linux-x64.tgz
./bin/jruby -e 'puts "#{RUBY_PLATFORM} #{RUBY_VERSION}"'
