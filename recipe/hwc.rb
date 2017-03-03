# encoding: utf-8
require_relative 'base'

class HwcRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go 1.8 binary to /usr/local/go/bin
    Dir.chdir("/usr/local") do
      go_download = "https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz"
      go_tar = "go.tar.gz"
      system("curl -L #{go_download} -o #{go_tar}")
      system("tar xf #{go_tar}")
    end

    FileUtils.rm_rf("#{tmp_path}/hwc")
    FileUtils.mv(Dir.glob("#{tmp_path}/hwc-*/hwc").first, "#{tmp_path}/hwc")
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
    '/tmp/src/github.com/cloudfoundry-incubator'
  end

  def archive_filename
    "#{name}-#{version}-windows-amd64.zip"
  end
end
