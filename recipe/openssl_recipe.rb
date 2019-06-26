require_relative 'base'

class OpenSSLRecipe < BaseRecipe
  def computed_options
    %w(--prefix=/usr --libdir=/lib/x86_64-linux-gnu --openssldir=/include/x86_64-linux-gnu/openssl)
  end

  def install
    return if installed?
    execute('install', [make_cmd, 'install', "DESTDIR=#{path}"])
  end

  def configure
    execute('configure', ['bash', '-c', "./config #{computed_options.join ' '}"])
  end

  def archive_files
    ["#{path}/*"]
  end

  def archive_path_name
    'openssl'
  end

  def setup_tar
    `rm -Rf #{path}/html/ #{path}/conf/*`
  end

  def url
    "https://github.com/openssl/openssl/archive/OpenSSL_1_1_0g.tar.gz"
  end
end