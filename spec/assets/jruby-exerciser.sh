#!/usr/bin/env bash
set +e

mkdir -p /tmp/binary-exerciser
current_dir=`pwd`
cd /tmp/binary-exerciser

tar xzf $current_dir/jruby-9.2.8.0-ruby-2.5-linux-x64.tgz
JAVA_HOME=/opt/java
PATH=$PATH:$JAVA_HOME/bin
./bin/jruby -e 'puts "#{RUBY_PLATFORM} #{RUBY_VERSION}"'
