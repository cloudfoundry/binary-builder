require 'mini_portile'
require_relative './httpd.apr'
require_relative './httpd.iconv'
require_relative './httpd.util'


class HttpdRecipe < MiniPortile
  def configure_options
    [
      "--with-apr=#{@staging_dir}/libapr-#{@apr_version}" ,
      "--with-apr-util=#{@staging_dir}/libapr-util-#{@apr_util_version}" ,
      "--enable-mpms-shared=worker event" ,
      "--enable-mods-shared=reallyall" ,
      "--disable-isapi" ,
      "--disable-dav" ,
      "--disable-dialup"
    ]
  end

  def port_path
    @install_dir
  end

  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end

  def compile
    execute('compile', [make_cmd, "prefix=#{@install_dir}"])
  end

  def url
    "https://archive.apache.org/dist/httpd/httpd-#{version}.tar.bz2"
  end

  def initialize name, version
    @apr_version = '1.5.2'
    @staging_dir = "/tmp/staged/app"
    @apr_iconv_version = '1.2.1'
    @apr_util_version = '1.5.4'
    @install_dir = "/app/httpd"
    super
  end

  def cook
    httpd_apr_recipe = HttpdAprRecipe.new('apr', @apr_version, @staging_dir)
    httpd_apr_recipe.cook

    httpd_iconv_recipe = HttpdIconvRecipe.new('apr-iconv', @apr_iconv_version, @staging_dir, @apr_version)
    httpd_iconv_recipe.cook

    httpd_util_recipe = HttpdUtilRecipe.new('apr-util', @apr_util_version, @staging_dir, @apr_version, @apr_iconv_version)
    httpd_util_recipe.cook
    super
    system   <<-eof
      cd #{@install_dir}
      rm -rf build/ cgi-bin/ error/ icons/ include/ man/ manual/ htdocs/
      rm -rf conf/extra/* conf/httpd.conf conf/httpd.conf.bak conf/magic conf/original
      cd -
      mkdir -p #{@install_dir}/lib
      cp "#{@staging_dir}/libapr-#{@apr_version}/lib/libapr-1.so.0" #{@install_dir}/lib
      cp "#{@staging_dir}/libapr-util-#{@apr_util_version}/lib/libaprutil-1.so.0" #{@install_dir}/lib
      cp "#{@staging_dir}/libapr-iconv-#{@apr_iconv_version}/lib/libapriconv-1.so.0" #{@install_dir}/lib
      ls -A #{File.expand_path("..", @install_dir)} | xargs tar czf httpd-#{version}-linux-x64.tgz -C #{File.expand_path("..", @install_dir)}
    eof
  end


end

