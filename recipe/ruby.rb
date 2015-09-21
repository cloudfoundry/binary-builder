require 'mini_portile'

class RubyRecipe < MiniPortile
  def configure_options
    [
      '--enable-load-relative',
      '--disable-install-doc',
      'debugflags=-g'
    ]
  end

  def port_path
    "/app/vendor/ruby-#{version}"
  end

  def minor_version
    version.match(/(\d+\.\d+)\./)[1]
  end

  def cook
    super
    system "ls -A #{port_path} | xargs tar czf ruby-#{version}-linux-x64.tgz -C #{port_path}"
  end

  def url
    "https://cache.ruby-lang.org/pub/ruby/#{minor_version}/ruby-#{version}.tar.gz"
  end
end

