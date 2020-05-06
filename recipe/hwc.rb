# encoding: utf-8
require_relative 'base'
require_relative '../lib/install_go'
require 'yaml'
require 'digest'

class HwcRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    install_go_compiler

    system <<-eof
      sudo apt-get update
      sudo apt-get -y upgrade
      sudo apt-get -y install mingw-w64
    eof

    FileUtils.rm_rf("#{tmp_path}/hwc")
    FileUtils.mv(Dir.glob("#{tmp_path}/hwc-*").first, "#{tmp_path}/hwc")
    Dir.chdir("#{tmp_path}/hwc") do
      system(
        {"PATH" => "#{ENV["PATH"]}:/usr/local/go/bin"},
      "./scripts/build.sh"
      ) or raise "Could not build hwc"
    end

    FileUtils.mv("#{tmp_path}/hwc/hwc-rel/hwc.exe", "/tmp/hwc.exe")
    FileUtils.mv("#{tmp_path}/hwc/hwc-rel/hwc_x86.exe", "/tmp/hwc_x86.exe")
  end

  def archive_files
    ['/tmp/hwc.exe', '/tmp/hwc_x86.exe']
  end

  def url
    "https://github.com/cloudfoundry/hwc/archive/#{version}.tar.gz"
  end

  def tmp_path
    '/tmp/src/code.cloudfoundry.org'
  end

  def archive_filename
    "#{name}-#{version}-windows-x86-64.zip"
  end
end
