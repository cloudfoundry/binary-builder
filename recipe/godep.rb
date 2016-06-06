# encoding: utf-8
require_relative 'base'

class GoRecipe < MiniPortile
  def cook
    install
  end

  def install
    raise 'Require `apt-get` package manager' unless which('apt-get')
    FileUtils.mkdir_p tmp_path
    execute('install', [which('apt-get'), 'update'], cd: Dir.pwd)
    execute('install', [which('apt-get'), '-y', 'install', 'golang'], cd: Dir.pwd)
  end
end

class GodepMeal < BaseRecipe
  attr_reader :name, :version

  def cook
    go_recipe.cook

    download unless downloaded?
    extract

    FileUtils.rm_rf("#{tmp_path}/godep")
    FileUtils.mv(Dir.glob("#{tmp_path}/godep-*").first, "#{tmp_path}/godep")
    Dir.chdir("#{tmp_path}/godep") do
      system(
        {"GOPATH" => "#{tmp_path}/godep/Godeps/_workspace:/tmp"},
        "go get ./..."
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
