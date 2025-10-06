# encoding: utf-8
require 'mini_portile'
require 'fileutils'
require_relative 'base'

class NodeRecipe < BaseRecipe

  def computed_options
    if Gem::Version.new(version) >= Gem::Version.new('6.0.0')
      %w[--prefix=/ --openssl-use-def-ca-store]
    else
      ['--prefix=/']
    end
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
    "https://github.com/nodejs/node/archive/refs/tags/v#{version}.tar.gz"
  end

  def dest_dir
    "/tmp/node-v#{version}-linux-x64"
  end

  def configure
    execute('configure', %w(python3 configure) + computed_options)
  end
end
