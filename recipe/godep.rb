# encoding: utf-8
require_relative 'base'

class GodepMeal < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    install_go_compiler

    FileUtils.rm_rf("#{tmp_path}/godep")
    FileUtils.mv(Dir.glob("#{tmp_path}/godep-*").first, "#{tmp_path}/godep")
    Dir.chdir("#{tmp_path}/godep") do
      system(
        {"GOPATH" => "#{tmp_path}/godep/Godeps/_workspace:/tmp"},
        "/usr/local/go/bin/go get ./..."
      ) or raise "Could not install godep"
    end
    FileUtils.mv("#{tmp_path}/godep/License", "/tmp/License")
  end

  def archive_files
    ['/tmp/bin/godep', '/tmp/License']
  end

  def archive_path_name
    'bin'
  end

  def url
    "https://github.com/tools/godep/archive/#{version}.tar.gz"
  end

  def go_recipe
    @go_recipe ||= GoRecipe.new(@name, @version)
  end

  def tmp_path
    '/tmp/src/github.com/tools'
  end
end
