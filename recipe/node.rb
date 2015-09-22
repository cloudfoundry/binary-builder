require 'mini_portile'

class NodeRecipe < MiniPortile
  def configure_options
    [
      '--shared-openssl',
      '--prefix=/'
    ]
  end

  def compile
    execute('compile', [make_cmd, "DESTDIR=/tmp/node-#{version}-linux-x64", 'PORTABLE=1'])
  end

  def cook
    super
    system "cp #{work_path}/LICENSE /tmp/node-#{version}-linux-x64"
    system "ls -A /tmp/node-#{version}-linux-x64 | xargs tar czf node-#{version}-linux-x64.tgz -C #{port_path}"
  end

  def url
    "https://nodejs.org/dist/v#{version}/node-v#{version}.tar.gz"
  end

  def configure
    execute('configure', %w(python configure) + computed_options)
  end
end

