# encoding: utf-8
require 'mini_portile'
require_relative 'base'

class JRubyRecipe < BaseRecipe
  def archive_files
    [
      "#{work_path}/bin",
      "#{work_path}/lib"
    ]
  end

  def url
    "https://s3.amazonaws.com/jruby.org/downloads/#{jruby_version}/jruby-src-#{jruby_version}.tar.gz"
  end

  def cook
    download unless downloaded?
    extract
    compile
  end

  def compile
    execute('compile', ['mvn', '-P', '!truffle', "-Djruby.default.ruby.version=#{ruby_version}"])
  end

  def ruby_version
    @ruby_version ||= version.match(/.*_ruby-(\d+\.\d).*/)[1]
  end

  def jruby_version
    @jruby_version ||= version.match(/(.*)_ruby-\d+\.\d.*/)[1]
  end
end
