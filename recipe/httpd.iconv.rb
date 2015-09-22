require 'mini_portile'
require_relative './httpd.rb'

class HttpdIconvRecipe < MiniPortile

  def configure_options
    [
      "--with-apr=#{@staging_dir}/libapr-#{@apr_version}/bin/apr-1-config"
    ]
  end

  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end

  def port_path
    "#{@staging_dir}/libapr-iconv-#{version}"
  end

  def initialize name, version, staging_dir, apr_version
    super name, version, {}
    @staging_dir = staging_dir
    @apr_version = apr_version
    @files = [{ url: "http://apache.mirrors.tds.net/apr/apr-iconv-#{version}.tar.gz" }]
  end
end

