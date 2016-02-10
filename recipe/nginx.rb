# encoding: utf-8
require_relative 'base'

class NginxRecipe < BaseRecipe
  def computed_options
    [
      '--prefix=/',
      '--error-log-path=stderr',
      '--with-http_ssl_module',
      '--with-http_realip_module',
      '--with-http_gunzip_module',
      '--with-http_gzip_static_module',
      '--with-http_auth_request_module',
      '--with-http_random_index_module',
      '--with-http_secure_link_module',
      '--with-http_stub_status_module',
      '--without-http_uwsgi_module',
      '--without-http_scgi_module',
      '--with-pcre',
      '--with-pcre-jit'
    ]
  end

  def install
    return if installed?
    execute('install', [make_cmd, 'install', "DESTDIR=#{path}"])
  end

  def archive_files
    ["#{path}/*"]
  end

  def archive_path_name
    'nginx'
  end

  def setup_tar
    `rm -Rf #{path}/html/ #{path}/conf/*`
  end

  def url
    "http://nginx.org/download/nginx-#{version}.tar.gz"
  end
end
