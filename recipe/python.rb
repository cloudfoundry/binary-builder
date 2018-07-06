# encoding: utf-8
require 'fileutils'
require 'mini_portile'
require_relative 'base'

class PythonRecipe < BaseRecipe
  def computed_options
    [
      '--enable-shared',
      '--with-ensurepip=no',
      '--with-dbmliborder=bdb:gdbm',
      '--with-tcltk-includes="-I/usr/include/tcl8.6"',
      '--with-tcltk-libs="-L/usr/lib/x86_64-linux-gnu -ltcl8.6 -L/usr/lib/x86_64-linux-gnu -ltk8.6"',
      "--prefix=#{prefix_path}",
      '--enable-unicode=ucs4'
    ]
  end

  def cook
    install_apt('libdb-dev libffi-dev libgdbm-dev tk8.6-dev')

    run('apt-get -y --force-yes -d install --reinstall libtcl8.6 libtk8.6 libxss1') or raise 'Failed to download libtcl8.6 libtk8.6 libxss1'
    FileUtils.mkdir_p prefix_path
    Dir.glob('/var/cache/apt/archives/lib{tcl8.6,tk8.6,xss1}_*.deb').each do |path|
      STDOUT.puts("dpkg -x #{path} #{prefix_path}")
      run("dpkg -x #{path} #{prefix_path}") or raise "Could not extract #{path}"
    end

    # Try to install newer openssl by using debian package
    run('wget http://security.debian.org/debian-security/pool/updates/main/o/openssl/openssl_1.1.0f-3+deb9u2_amd64.deb')
    Dir.glob('openssl_*.deb').each do |path|
      STDOUT.puts("dpkg -x #{path} #{prefix_path}")
      run("dpkg -x #{path} #{prefix_path}") or raise "Could not extract #{path}"
    end

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
    if run("sudo apt-get update && sudo apt-get -y install #{packages}")
      STDOUT.puts "OK"
    else
      raise "Failed to complete install dependencies task"
    end
  end

  def run(command)
    output = `#{command}`
    if $?.success?
      return true
    else
      STDOUT.puts "ERROR, output was:"
      STDOUT.puts output
      return false
    end
  end
end
