# encoding: utf-8
require_relative 'base'

class RabbitMQRecipe < BaseRecipe
  def url
    "https://github.com/alanxz/rabbitmq-c/releases/download/v#{version}/rabbitmq-c-#{version}.tar.gz"
  end
end

class PeclRecipe < BaseRecipe
  def url
    "http://pecl.php.net/get/#{name}-#{version}.tgz"
  end

  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config"
    ]
  end

  def configure
    return if configured?

    md5_file = File.join(tmp_path, 'configure.md5')
    digest   = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, 'w') { |f| f.write digest }

    execute('configure', 'phpize')
    execute('configure', %w(sh configure) + computed_options)
  end
end

class LibmemcachedRecipe < BaseRecipe
  def url
    "https://launchpad.net/libmemcached/1.0/#{version}/+download/libmemcached-#{version}.tar.gz"
  end
end

class LuaRecipe < BaseRecipe
  def url
    "http://www.lua.org/ftp/lua-#{version}.tar.gz"
  end

  def configure
  end

  def compile
    execute('compile', ['bash', '-c', "#{make_cmd} linux MYCFLAGS=-fPIC"])
  end

  def install
    return if installed?

    execute('install', ['bash', '-c', "#{make_cmd} install INSTALL_TOP=#{path}"])
  end
end

class IonCubeRecipe < BaseRecipe
  # NOTE: not a versioned URL, will always be the lastest support version
  def url
    'http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_x86-64.tar.gz'
  end

  def configure; end

  def compile; end

  def install; end

  def path
    work_path
  end
end

class HiredisRecipe < BaseRecipe
  def url
    "https://github.com/redis/hiredis/archive/v#{version}.tar.gz"
  end

  def configure
  end

  def install
    return if installed?

    execute('install', ['bash', '-c', "LIBRARY_PATH=lib PREFIX='#{path}' #{make_cmd} install"])
  end
end

class PHPIRedisRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      '--enable-phpiredis',
      "--with-hiredis-dir=#{@hiredis_path}"
    ]
  end

  def url
    "https://github.com/nrk/phpiredis/archive/#{version}.tar.gz"
  end
end

class AmqpPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      "--with-librabbitmq-dir=#{@rabbitmq_path}"
    ]
  end
end

class OraclePeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-oci8=shared,instantclient,/oracle"
    ]
  end

  def oracle_sdk?
    File.directory?('/oracle')
  end

  def setup_tar
    puts "Copying oracle libs from /oracle/ to #{@php_path}/lib"
    system <<-eof
      cp -a /oracle/libclntshcore.so.12.1 #{@php_path}/lib
      cp -a /oracle/libclntsh.so #{@php_path}/lib
      cp -a /oracle/libclntsh.so.12.1 #{@php_path}/lib
      cp -a /oracle/libipc1.so #{@php_path}/lib
      cp -a /oracle/libmql1.so #{@php_path}/lib
      cp -a /oracle/libnnz12.so #{@php_path}/lib
      cp -a /oracle/libociicus.so #{@php_path}/lib
      cp -a /oracle/libons.so #{@php_path}/lib
    eof
  end
end

class LuaPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      "--with-lua=#{@lua_path}"
    ]
  end
end

class PHPProtobufPeclRecipe < PeclRecipe
  def url
    "https://github.com/allegro/php-protobuf/archive/#{version}.tar.gz"
  end
end

class PhalconRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      '--enable-phalcon'
    ]
  end

  def work_path
    "#{super}/build/64bits"
  end

  def url
    "https://github.com/phalcon/cphalcon/archive/phalcon-v#{version}.tar.gz"
  end
end

class MemcachedPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      "--with-libmemcached-dir=#{@libmemcached_path}",
      '--enable-memcached-sasl',
      '--enable-memcached-msgpack',
      '--enable-memcached-igbinary',
      '--enable-memcached-json'
    ]
  end
end

class SuhosinPeclRecipe < PeclRecipe
  def url
    "http://download.suhosin.org/suhosin-#{version}.tar.gz"
  end
end

class TwigPeclRecipe < PeclRecipe
  def url
    "https://github.com/twigphp/Twig/archive/v#{version}.tar.gz"
  end

  def work_path
    "#{super}/ext/twig"
  end
end

class XcachePeclRecipe < PeclRecipe
  def url
    "http://xcache.lighttpd.net/pub/Releases/#{version}/xcache-#{version}.tar.gz"
  end
end

class XhprofPeclRecipe < PeclRecipe
  def url
    "https://github.com/phacility/xhprof/archive/#{version}.tar.gz"
  end

  def work_path
    "#{super}/extension"
  end
end

class SnmpRecipe
  def initialize(php_path)
    @php_path = php_path
  end

  def cook
    system <<-eof
      cd #{@php_path}
      mkdir -p mibs
      cp "/usr/lib/x86_64-linux-gnu/libnetsnmp.so.30" lib/
      # copy mibs that are packaged freely
      cp /usr/share/snmp/mibs/* mibs
      # copy mibs downloader & smistrip, will download un-free mibs
      cp /usr/bin/download-mibs bin
      cp /usr/bin/smistrip bin
      sed -i "s|^CONFDIR=/etc/snmp-mibs-downloader|CONFDIR=\$HOME/php/mibs/conf|" bin/download-mibs
      sed -i "s|^SMISTRIP=/usr/bin/smistrip|SMISTRIP=\$HOME/php/bin/smistrip|" bin/download-mibs
      # copy mibs download config
      cp -R /etc/snmp-mibs-downloader mibs/conf
      sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/iana.conf
      sed -i "s|^DEST=iana|DEST=|" mibs/conf/iana.conf
      sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/ianarfc.conf
      sed -i "s|^DEST=iana|DEST=|" mibs/conf/ianarfc.conf
      sed -i "s|^DIR=/usr/share/doc|DIR=\$HOME/php/mibs/originals|" mibs/conf/rfc.conf
      sed -i "s|^DEST=ietf|DEST=|" mibs/conf/rfc.conf
      sed -i "s|^BASEDIR=/var/lib/mibs|BASEDIR=\$HOME/php/mibs|" mibs/conf/snmp-mibs-downloader.conf
      # copy data files
      mkdir mibs/originals
      cp -R /usr/share/doc/mibiana mibs/originals
      cp -R /usr/share/doc/mibrfcs mibs/originals
    eof
  end
end

# PHP 5 and PHP 7 Common recipes

def amqppecl_recipe
  AmqpPeclRecipe.new('amqp', '1.7.0', md5: '5a701987a5c9d1f1b70b359e14d5162e',
                                      php_path: php_recipe.path,
                                      rabbitmq_path: rabbitmq_recipe.path)
end

def lua_recipe
  LuaRecipe.new('lua', '5.3.2', md5: '33278c2ab5ee3c1a875be8d55c1ca2a1')
end

def rabbitmq_recipe
  RabbitMQRecipe.new('rabbitmq', '0.7.1', md5: '6216c8876299a5efc4ff5ff84dc636d8')
end
