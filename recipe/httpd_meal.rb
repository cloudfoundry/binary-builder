# encoding: utf-8
require_relative 'base'

class AprRecipe < BaseRecipe
  def url
    "https://apache.osuosl.org/apr/apr-#{version}.tar.gz"
  end
end

class AprIconvRecipe < BaseRecipe
  def configure_options
    [
      "--with-apr=#{@apr_path}/bin/apr-1-config"
    ]
  end

  def url
    "https://apache.osuosl.org/apr/apr-iconv-#{version}.tar.gz"
  end
end

class AprUtilRecipe < BaseRecipe
  def configure_options
    [
      "--with-apr=#{@apr_path}",
      "--with-iconv=#{@apr_iconv_path}",
      '--with-crypto',
      '--with-openssl',
      '--with-mysql',
      '--with-pgsql',
      '--with-gdbm',
      '--with-ldap'
    ]
  end

  def url
    "https://apache.osuosl.org/apr/apr-util-#{version}.tar.gz"
  end
end

class HTTPdRecipe < BaseRecipe
  def computed_options
    [
      '--prefix=/app/httpd',
      "--with-apr=#{@apr_path}",
      "--with-apr-util=#{@apr_util_path}",
      '--with-ssl=/usr/lib/x86_64-linux-gnu',
      '--enable-mpms-shared=worker event',
      '--enable-mods-shared=reallyall',
      '--disable-isapi',
      '--disable-dav',
      '--disable-dialup'
    ]
  end

  def install
    return if installed?
    execute('install', [make_cmd, 'install', "prefix=#{path}"])
  end

  def url
    "https://archive.apache.org/dist/httpd/httpd-#{version}.tar.bz2"
  end

  def archive_files
    ["#{path}/*"]
  end

  def archive_path_name
    'httpd'
  end

  def setup_tar
    system <<-eof
      cd #{path}

      rm -rf cgi-bin/ error/ icons/ include/ man/ manual/ htdocs/
      rm -rf conf/extra/* conf/httpd.conf conf/httpd.conf.bak conf/magic conf/original

      mkdir -p lib
      cp "#{@apr_path}/lib/libapr-1.so.0" ./lib
      cp "#{@apr_util_path}/lib/libaprutil-1.so.0" ./lib
      mkdir -p "./lib/apr-util-1"
      cp "#{@apr_util_path}/lib/apr-util-1/"*.so ./lib/apr-util-1/
      mkdir -p "./lib/iconv"
      cp "#{@apr_iconv_path}/lib/libapriconv-1.so.0" ./lib
      cp "#{@apr_iconv_path}/lib/iconv/"*.so ./lib/iconv/
      cp /usr/lib/x86_64-linux-gnu/libcjose.so* ./lib/
      cp /usr/lib/x86_64-linux-gnu/libhiredis.so* ./lib/
      cp /usr/lib/x86_64-linux-gnu/libjansson.so* ./lib/
    eof
  end
end

class ModAuthOpenidcRecipe < BaseRecipe
  def url
    "https://github.com/zmartzone/mod_auth_openidc/releases/download/v#{version}/mod_auth_openidc-#{version}.tar.gz"
  end

  def configure_options
    ENV['APR_LIBS'] = `#{@apr_path}/bin/apr-1-config --link-ld --libs`
    ENV['APR_CFLAGS'] = `#{@apr_path}/bin/apr-1-config --cflags --includes`
    [
      "--with-apxs2=#{@httpd_path}/bin/apxs"
    ]
  end
end

class HTTPdMeal
  attr_reader :name, :version

  def initialize(name, version, options = {})
    @name    = name
    @version = version
    @options = options
  end

  def cook
    run('apt update') or raise 'Failed to apt update'
    run('apt-get install -y libldap2-dev') or raise 'Failed to install libldap2-dev'

    apr_recipe.cook
    apr_iconv_recipe.cook
    apr_util_recipe.cook
    httpd_recipe.cook

    # this symlink is needed so that modules can call `apxs`
    #  putting it here because we only need to do it once
    system <<-eof
      cd /app
      if ! [ -L "/app/httpd" ]; then
        ln -s "#{httpd_recipe.path}" httpd
      fi
    eof

    run('apt-get install -y libjansson-dev libcjose-dev libhiredis-dev') or raise 'Failed to install additional dependencies'
    mod_auth_openidc_recipe.cook
  end

  def url
    httpd_recipe.url
  end

  def archive_files
    httpd_recipe.archive_files
  end

  def archive_path_name
    httpd_recipe.archive_path_name
  end

  def archive_filename
    httpd_recipe.archive_filename
  end

  def setup_tar
    httpd_recipe.setup_tar
  end

  private

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

  def files_hashs
    hashes = httpd_recipe.send(:files_hashs) +
      apr_recipe.send(:files_hashs)       +
      apr_iconv_recipe.send(:files_hashs) +
      apr_util_recipe.send(:files_hashs) +
      mod_auth_openidc_recipe.send(:files_hashs)

    hashes
  end

  def mod_auth_openidc_recipe
    @mod_auth_openidc ||= ModAuthOpenidcRecipe.new('mod-auth-openidc', '2.3.8',
      httpd_path: httpd_recipe.path,
      apr_path: apr_recipe.path,
      md5: 'd6abc2f68dabf5d2557400af2499f500')
  end

  def httpd_recipe
    @http_recipe ||= HTTPdRecipe.new(@name, @version, {
      apr_path: apr_recipe.path,
      apr_util_path: apr_util_recipe.path,
      apr_iconv_path: apr_iconv_recipe.path
    }.merge(DetermineChecksum.new(@options).to_h))
  end

  def apr_util_recipe
    @apr_util_recipe ||= AprUtilRecipe.new('apr-util', '1.6.1', apr_path: apr_recipe.path,
                                                                apr_iconv_path: apr_iconv_recipe.path,
                                                                md5: 'bd502b9a8670a8012c4d90c31a84955f')
  end

  def apr_iconv_recipe
    @apr_iconv_recipe ||= AprIconvRecipe.new('apr-iconv', '1.2.2', apr_path: apr_recipe.path,
                                                                   md5: '60ae6f95ee4fdd413cf7472fd9c776e3')
  end

  def apr_recipe
    @apr_recipe ||= AprRecipe.new('apr', '1.7.0', md5: '757239852b082b844e268a86f2806fd2')
  end
end
