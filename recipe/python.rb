# encoding: utf-8
require 'mini_portile'
require_relative 'base'

class PythonRecipe < BaseRecipe
  def computed_options
    [
      '--enable-shared',
      '--with-ensurepip=no',
      "--prefix=#{prefix_path}",
      '--enable-unicode=ucs4'
    ]
  end

  def archive_files
    ["#{prefix_path}/*"]
  end

  def setup_tar
    unless File.exist?("#{prefix_path}/bin/python")
      File.symlink('./python3', "#{prefix_path}/bin/python")
    end
  end

  def url
    "https://www.python.org/ftp/python/#{version}/Python-#{version}.tgz"
  end

  def prefix_path
    '/app/.heroku/vendor'
  end
end
