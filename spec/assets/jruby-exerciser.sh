#!/usr/bin/env bash
set +e

mkdir -p /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

tar xzf $current_dir/jruby-9.1.6.0_ruby-2.3.1-linux-x64.tgz
JAVA_HOME=/opt/java
PATH=$PATH:$JAVA_HOME/bin
./bin/jruby -e 'puts "#{RUBY_PLATFORM} #{RUBY_VERSION}"'
