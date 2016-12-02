# encoding: utf-8
require 'mini_portile'
require_relative 'base'

class RubyRecipe < BaseRecipe
  def computed_options
    [
      '--enable-load-relative',
      '--disable-install-doc',
      'debugflags=-g',
      "--prefix=#{prefix_path}",
      "--without-gmp"
    ]
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
end
