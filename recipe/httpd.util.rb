require 'mini_portile'
require_relative './httpd.rb'

class HttpdUtilRecipe < MiniPortile

  def configure_options
    [
      "--with-apr=#{@staging_dir}/libapr-#{@apr_version}",
      "--with-iconv=#{@staging_dir}/libapr-iconv-#{@apr_iconv_version}",
      "--with-crypto",
      "--with-openssl",
      "--with-mysql",
      "--with-pgsql",
      "--with-gdbm",
      "--with-ldap"
    ]
  end

  def port_path
    "#{@staging_dir}/libapr-util-#{version}"
  end

  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end


  def initialize name, version, staging_dir, apr_version, apr_iconv_version
    super name, version, {}
    @staging_dir = staging_dir
    @apr_version = apr_version
    @apr_iconv_version = apr_iconv_version
    @files = [{ url: "http://apache.mirrors.tds.net/apr/apr-util-#{version}.tar.gz" }]
  end
end

