require 'mini_portile'
require_relative 'base'

class RubyRecipe < BaseRecipe
  def computed_options
    [
      '--enable-load-relative',
      '--disable-install-doc',
      'debugflags=-g',
      "prefix=#{prefix_path}"
    ]
  end

  def prefix_path
    "/app/vendor/ruby-#{version}"
  end

  def minor_version
    version.match(/(\d+\.\d+)\./)[1]
  end

  def tar
    system "ls -A #{prefix_path} | xargs tar czf ruby-#{version}-linux-x64.tgz -C #{prefix_path}"
  end

  def url
    "https://cache.ruby-lang.org/pub/ruby/#{minor_version}/ruby-#{version}.tar.gz"
  end
end

