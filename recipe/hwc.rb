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

    FileUtils.rm_rf("#{tmp_path}/hwc")
    FileUtils.mv(Dir.glob("#{tmp_path}/hwc-*").first, "#{tmp_path}/hwc")
    Dir.chdir("#{tmp_path}/hwc") do
      system(
        {"GOPATH" => "/tmp",
         "PATH" => "#{ENV["PATH"]}:/usr/local/go/bin",
         "GOOS" => "windows"},
        "/usr/local/go/bin/go build"
      ) or raise "Could not build hwc"
    end

    FileUtils.mv("#{tmp_path}/hwc/hwc.exe", "/tmp/hwc.exe")
  end

  def archive_files
    ['/tmp/hwc.exe']
  end

  def url
    "https://github.com/cloudfoundry-incubator/hwc/archive/#{version}.tar.gz"
  end

  def tmp_path
    '/tmp/src/code.cloudfoundry.org'
  end

  def archive_filename
    "#{name}-#{version}-windows-amd64.zip"
  end
end
