# encoding: utf-8
require 'mini_portile'
require 'fileutils'
require_relative 'base'

class NodeRecipe < BaseRecipe
  def computed_options
    [
      '--prefix=/'
    ]
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
    execute('configure', %w(python configure) + computed_options)
  end
end
