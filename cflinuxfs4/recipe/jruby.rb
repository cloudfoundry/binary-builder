# frozen_string_literal: true

require 'mini_portile2'
require_relative 'base'

class JRubyRecipe < BaseRecipe
  def archive_files
    %W[#{work_path}/bin #{work_path}/lib]
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
    @ruby_version ||= version.match(/.*-ruby-(\d+\.\d+)/)[1]
  end

  def jruby_version
    @jruby_version ||= version.match(/(.*)-ruby-\d+\.\d+/)[1]
  end
end
