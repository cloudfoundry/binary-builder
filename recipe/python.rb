require 'mini_portile'
require_relative 'base'

class PythonRecipe < BaseRecipe
  def computed_options
    [
      "--enable-shared",
      "--with-ensurepip=no",
      "--prefix=#{prefix_path}"
    ]
  end

  def archive_files
    [ "#{prefix_path}/*" ]
  end

  def tar
    unless File.exist?("#{prefix_path}/bin/python")
      File.symlink("#{prefix_path}/bin/python3", "#{prefix_path}/bin/python")
    end
    super
  end

  def url
    "https://www.python.org/ftp/python/#{version}/Python-#{version}.tgz"
  end

  def prefix_path
    '/app/.heroku/vendor'
  end
end

