require 'mini_portile'
require_relative 'base'

class NodeRecipe < BaseRecipe
  def computed_options
    [
      '--shared-openssl',
      '--prefix=/'
    ]
  end

  def install
    execute('install', [make_cmd, "install", "DESTDIR=#{dest_dir}", 'PORTABLE=1'])
  end

  def tar
    system "cp #{work_path}/LICENSE #{dest_dir}"
    system "ls -A /tmp | xargs tar czf node-#{version}-linux-x64.tgz -C /tmp"
  end

  def url
    "https://nodejs.org/dist/v#{version}/node-v#{version}.tar.gz"
  end

  def dest_dir
    "/tmp/node-v#{version}-linux-x64"
  end

  def configure
    execute('configure', %w(python configure) + computed_options)
  end
end

