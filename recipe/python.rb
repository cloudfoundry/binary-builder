# encoding: utf-8
require 'mini_portile'
require_relative 'base'

class PythonRecipe < BaseRecipe
  def computed_options
    [
      '--enable-shared',
      '--with-ensurepip=no',
      '--with-dbmliborder=bdb:gdbm',
      "--prefix=#{prefix_path}",
      '--enable-unicode=ucs4'
    ]
  end

  def cook
    install_apt('libdb-dev libgdbm-dev')

    super
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

  private

  def install_apt(packages)
    STDOUT.print "Running 'install dependencies' for #{@name} #{@version}... "
    STDOUT.flush
    apt_output = `sudo apt-get update && sudo apt-get -y install #{packages}`
    if $?.success?
      STDOUT.puts "OK"
    else
      STDOUT.puts "ERROR, output was:"
      STDOUT.puts apt_output
      raise "Failed to complete install dependencies task"
    end
  end
end
