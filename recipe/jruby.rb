require 'mini_portile'
require_relative 'base'

class JRubyRecipe < BaseRecipe
  def tar
    system "ls -A #{port_path}/bin #{port_path}/lib | xargs tar czf ruby-#{version}-linux-x64.tgz -C #{port_path}"
  end

  def url
    "https://s3.amazonaws.com/jruby.org/downloads/#{jruby_version}/jruby-src-#{jruby_version}.tar.gz"
  end

  def cook
    download unless downloaded?
    extract
    compile
    tar
  end

  def compile
    execute('compile', ['mvn', "-Djruby.default.ruby.version=#{ruby_version}"])
  end

  def ruby_version
    @ruby_version ||= version.match(/.*_ruby-(\d+\.\d).*/)[1]
  end

  def jruby_version
    @jruby_version ||= version.match(/(.*)_ruby-\d+\.\d.*/)[1]
  end
end

