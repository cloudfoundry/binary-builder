# frozen_string_literal: true

require 'mini_portile2'
require 'fileutils'
require_relative 'base'

class NodeRecipe < BaseRecipe
  def computed_options
    %w[--prefix=/ --openssl-use-def-ca-store]
  end

  def install
    execute('install', [make_cmd, 'install', "DESTDIR=#{dest_dir}", 'PORTABLE=1'])
  end

  def archive_files
    [dest_dir]
  end

  def setup_tar
    FileUtils.cp(
      "#{work_path}/LICENSE",
      dest_dir
    )
  end

  def url
    "https://nodejs.org/dist/v#{version}/node-v#{version}.tar.gz"
  end

  def dest_dir
    "/tmp/node-v#{version}-linux-x64"
  end

  def configure
    # Node building requires python https://github.com/nodejs/node/blob/main/BUILDING.md#unix-and-macos
    # But cflinuxfs4 image does not come with python
    system <<-EOF
      #!/bin/sh
      if [ -z $(command -v python3) ]; then
        apt update
        apt install -y python3 python3-pip
      fi
    EOF
    execute('configure', %w(./configure) + computed_options)
  end
end
