# frozen_string_literal: true

require 'English'
require 'mini_portile2'
require_relative 'base'

class RubyRecipe < BaseRecipe
  def computed_options
    [
      '--enable-load-relative',
      '--disable-install-doc',
      'debugflags=-g',
      "--prefix=#{prefix_path}",
      '--without-gmp'
    ]
  end

  def cook
    run('apt-get update') or raise 'Failed to apt-get update'
    run('apt-get -y install libffi-dev') or raise 'Failed to install libffi-dev'
    super
  end

  def prefix_path
    "/app/vendor/ruby-#{version}"
  end

  def minor_version
    version.match(/(\d+\.\d+)\./)[1]
  end

  def archive_files
    ["#{prefix_path}/*"]
  end

  def url
    "https://cache.ruby-lang.org/pub/ruby/#{minor_version}/ruby-#{version}.tar.gz"
  end

  private

  def run(command)
    output = `#{command}`
    if $CHILD_STATUS.success?
      true
    else
      $stdout.puts 'ERROR, output was:'
      $stdout.puts output
      false
    end
  end
end
