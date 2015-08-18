#!/usr/bin/env bash
set +e

mkdir -p /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

tar xzf $current_dir/jruby-9.0.0.0_ruby-2.2.0-linux-x64.tgz
./bin/jruby -e 'puts "#{RUBY_PLATFORM} #{RUBY_VERSION}"'
