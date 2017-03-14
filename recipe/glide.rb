# encoding: utf-8
require_relative 'base'

class GlideRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    install_go_compiler

    FileUtils.rm_rf("#{tmp_path}/glide")
    FileUtils.mv(Dir.glob("#{tmp_path}/glide-*").first, "#{tmp_path}/glide")
    Dir.chdir("#{tmp_path}/glide") do
      system(
        {"GOPATH" => "/tmp",
         "PATH" => "#{ENV["PATH"]}:/usr/local/go/bin"},
        "/usr/local/go/bin/go build"
      ) or raise "Could not install glide"
    end

    FileUtils.mv("#{tmp_path}/glide/glide", "/tmp/glide")
    FileUtils.mv("#{tmp_path}/glide/LICENSE", "/tmp/LICENSE")
  end

  def archive_files
    ['/tmp/glide', '/tmp/LICENSE']
  end

  def archive_path_name
    'bin'
  end

  def url
    "https://github.com/Masterminds/glide/archive/#{version}.tar.gz"
  end

  def go_recipe
    @go_recipe ||= GoRecipe.new(@name, @version)
  end

  def tmp_path
    '/tmp/src/github.com/Masterminds'
  end
end
