# encoding: utf-8
require 'mini_portile'
require 'fileutils'
require_relative 'openssl_recipe'
require_relative 'base'

class NodeRecipe < BaseRecipe

  def initialize(name, version, options = {})
    super name, version, options
    # override openssl in container
    OpenSSLRecipe.new('openssl', 'OpenSSL_1_1_0g',
                      { sha256: '8e9516b8635bb9113c51a7b5b27f9027692a56b104e75b709e588c3ffd6a0422' }).cook
  end

  def computed_options
    puts '----- computed_options ---------'
    puts [ Gem::Version.new(version), '>=', Gem::Version.new('6.0.0') ]
    if Gem::Version.new(version) >= Gem::Version.new('6.0.0')
      ['--prefix=/', '--openssl-use-def-ca-store']
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
    "https://nodejs.org/dist/v#{version}/node-v#{version}.tar.gz"
  end

  def dest_dir
    "/tmp/node-v#{version}-linux-x64"
  end

  def configure
    execute('configure', %w(python configure) + computed_options)
  end
end
