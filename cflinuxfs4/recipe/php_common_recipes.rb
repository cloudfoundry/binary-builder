# frozen_string_literal: true

require_relative 'base'
require_relative '../lib/geoip_downloader'
require 'uri'

class BasePHPModuleRecipe < BaseRecipe
  def initialize(name, version, options = {})
    super name, version, options

    @files = [{
                url: url,
                local_path: local_path
              }.merge(DetermineChecksum.new(options).to_h)]
  end

  def local_path
    File.join(archives_path, File.basename(url))
  end

  # override this method to allow local_path to be specified
  # this prevents recipes with the same versions downloading colliding files (such as `v1.0.0.tar.gz`)
  def files_hashs
    @files.map do |file|
      hash = case file
             when String
               { url: file }
             when Hash
               file.dup
             else
               raise ArgumentError, 'files must be an Array of Stings or Hashs'
             end

      hash[:local_path] = local_path
      hash
    end
  end
end

class PeclRecipe < BasePHPModuleRecipe
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
    digest = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, 'w') { |f| f.write digest }

    execute('phpize', 'phpize')
    execute('configure', %w[sh configure] + computed_options)
  end
end

class AmqpPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config"
    ]
  end
end

class PkgConfigLibRecipe < BasePHPModuleRecipe
  def cook
    exists = system("PKG_CONFIG_PATH=$PKG_CONFIG_PATH:#{pkg_path} pkg-config #{pkgcfg_name} --exists")
    super() unless exists
  end

  def pkg_path
    "#{File.expand_path(port_path)}/lib/pkgconfig/"
  end
end

class MaxMindRecipe < PeclRecipe
  def work_path
    File.join(tmp_path, "maxminddb-#{version}", 'ext')
  end
end

class HiredisRecipe < PkgConfigLibRecipe
  def url
    "https://github.com/redis/hiredis/archive/v#{version}.tar.gz"
  end

  def local_path
    "hiredis-#{version}.tar.gz"
  end

  def configure; end

  def install
    return if installed?

    execute('install', ['bash', '-c', "LIBRARY_PATH=lib PREFIX='#{path}' #{make_cmd} install"])
  end

  def pkgcfg_name
    'hiredis'
  end
end

class ImagickRecipe < BasePHPModuleRecipe
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
    digest = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, 'w') { |f| f.write digest }

    # Setup libmagickwand-dev libmagickcore-dev
    execute('install', %w[apt-get update])
    execute('install', %w[apt-get install -y libmagickwand-dev])

    # Configure
    execute('phpize', 'phpize')
    execute('configure', %w[sh configure] + computed_options)
  end
end

class LibSodiumRecipe < PkgConfigLibRecipe
  def url
    "https://github.com/jedisct1/libsodium/archive/#{version}-RELEASE.tar.gz"
  end

  def pkgcfg_name
    'libsodium'
  end
end

class IonCubeRecipe < BaseRecipe
  def url
    "http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_x86-64_#{version}.tar.gz"
  end

  def configure; end

  def compile; end

  def install; end

  def self.build_ioncube?(_php_version)
    true
  end

  def path
    tmp_path
  end
end

class LibRdKafkaRecipe < PkgConfigLibRecipe
  def url
    "https://github.com/edenhill/librdkafka/archive/v#{version}.tar.gz"
  end

  def pkgcfg_name
    'rdkafka'
  end

  def local_path
    "librdkafka-#{version}.tar.gz"
  end

  def work_path
    File.join(tmp_path, "librdkafka-#{version}")
  end

  def configure_prefix
    '--prefix=/usr'
  end

  def configure
    return if configured?

    md5_file = File.join(tmp_path, 'configure.md5')
    digest = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, 'w') { |f| f.write digest }

    execute('configure', %w[bash ./configure] + computed_options)
  end
end

class CassandraCppDriverRecipe < PkgConfigLibRecipe
  def url
    "https://github.com/datastax/cpp-driver/archive/#{version}.tar.gz"
  end

  def pkgcfg_name
    'cassandra'
  end

  def local_path
    "cassandra-cpp-driver-#{version}.tar.gz"
  end

  def configure; end

  def compile
    execute('compile', ['bash', '-c', 'mkdir -p build && cd build && cmake .. && make'])
  end

  def install
    execute('install', ['bash', '-c', 'cd build && make install'])
  end
end

class LuaPeclRecipe < PeclRecipe
  def configure_options
    %W[--with-php-config=#{@php_path}/bin/php-config --with-lua=#{@lua_path}]
  end
end

class LuaRecipe < BaseRecipe
  def url
    "http://www.lua.org/ftp/lua-#{version}.tar.gz"
  end

  def configure; end

  def compile
    execute('compile', ['bash', '-c', "#{make_cmd} linux MYCFLAGS=-fPIC"])
  end

  def install
    return if installed?

    execute('install', ['bash', '-c', "#{make_cmd} install INSTALL_TOP=#{path}"])
  end
end

class MemcachedPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      '--with-libmemcached-dir',
      '--enable-memcached-sasl',
      '--enable-memcached-msgpack',
      '--enable-memcached-igbinary',
      '--enable-memcached-json'
    ]
  end
end

class FakePeclRecipe < PeclRecipe
  def url
    "file://#{@php_source}/ext/#{name}-#{version}.tar.gz"
  end

  def download
    # this copys an extension folder out of the PHP source director (i.e. `ext/<name>`)
    # it pretends to download it by making a zip of the extension files
    # that way the rest of the PeclRecipe works normally
    files_hashs.each do |file|
      path = URI(file[:url]).path.rpartition('-')[0] # only need path before the `-`, see url above
      system <<-EOF
        tar czf "#{file[:local_path]}" -C "#{File.dirname(path)}" "#{File.basename(path)}"
      EOF
    end
  end
end

class Gd72and73FakePeclRecipe < FakePeclRecipe
  def configure_options
    baseOpts = %w[--with-jpeg-dir --with-png-dir --with-xpm-dir --with-freetype-dir --with-webp-dir --with-zlib-dir]

    if version.start_with?('7.2')
      baseOpts.push('--enable-gd-jis-conv')
    else
      baseOpts.push('--with-gd-jis-conv')
    end
  end
end

class Gd74FakePeclRecipe < FakePeclRecipe
  # how to build gd.so in PHP 7.4 changed dramatically
  #  In 7.4+, you can just use libgd from Ubuntu
  def configure_options
    [
      '--with-external-gd'
    ]
  end
end

class OdbcRecipe < FakePeclRecipe
  def configure_options
    [
      '--with-unixODBC=shared,/usr'
    ]
  end

  def patch
    system <<-EOF
      cd #{work_path}
      echo 'AC_DEFUN([PHP_ALWAYS_SHARED],[])dnl' > temp.m4
      echo >> temp.m4
      cat config.m4 >> temp.m4
      mv temp.m4 config.m4
    EOF
  end

  def setup_tar
    system <<-EOF
      cp -a /usr/lib/x86_64-linux-gnu/libodbc.so* #{@php_path}/lib/
      cp -a /usr/lib/x86_64-linux-gnu/libodbcinst.so* #{@php_path}/lib/
    EOF
  end
end

class SodiumRecipe < FakePeclRecipe
  def configure_options
    ENV['LDFLAGS'] = "-L#{@libsodium_path}/lib"
    ENV['PKG_CONFIG_PATH'] = "#{@libsodium_path}/lib/pkgconfig"
    sodium_flag = "--with-sodium=#{@libsodium_path}"
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      sodium_flag
    ]
  end

  def setup_tar
    system <<-EOF
      cp -a #{@libsodium_path}/lib/libsodium.so* #{@php_path}/lib/
    EOF
  end
end

class PdoOdbcRecipe < FakePeclRecipe
  def configure_options
    [
      '--with-pdo-odbc=unixODBC,/usr'
    ]
  end

  def setup_tar
    system <<-EOF
      cp -a /usr/lib/x86_64-linux-gnu/libodbc.so* #{@php_path}/lib/
      cp -a /usr/lib/x86_64-linux-gnu/libodbcinst.so* #{@php_path}/lib/
    EOF
  end
end

class OraclePdoRecipe < FakePeclRecipe
  def configure_options
    [
      "--with-pdo-oci=shared,instantclient,/oracle,#{OraclePdoRecipe.oracle_version}"
    ]
  end

  def self.oracle_version
    Dir['/oracle/*'].select { |i| i.match(/libclntsh\.so\./) }.map { |i| i.sub(/.*libclntsh\.so\./, '') }.first
  end

  def setup_tar
    system <<-EOF
      cp -an /oracle/libclntshcore.so.12.1 #{@php_path}/lib
      cp -an /oracle/libclntsh.so #{@php_path}/lib
      cp -an /oracle/libclntsh.so.12.1 #{@php_path}/lib
      cp -an /oracle/libipc1.so #{@php_path}/lib
      cp -an /oracle/libmql1.so #{@php_path}/lib
      cp -an /oracle/libnnz12.so #{@php_path}/lib
      cp -an /oracle/libociicus.so #{@php_path}/lib
      cp -an /oracle/libons.so #{@php_path}/lib
    EOF
  end
end

class OraclePeclRecipe < PeclRecipe
  def configure_options
    [
      '--with-oci8=shared,instantclient,/oracle'
    ]
  end

  def self.oracle_sdk?
    File.directory?('/oracle')
  end

  def setup_tar
    system <<-EOF
      cp -an /oracle/libclntshcore.so.12.1 #{@php_path}/lib
      cp -an /oracle/libclntsh.so #{@php_path}/lib
      cp -an /oracle/libclntsh.so.12.1 #{@php_path}/lib
      cp -an /oracle/libipc1.so #{@php_path}/lib
      cp -an /oracle/libmql1.so #{@php_path}/lib
      cp -an /oracle/libnnz12.so #{@php_path}/lib
      cp -an /oracle/libociicus.so #{@php_path}/lib
      cp -an /oracle/libons.so #{@php_path}/lib
    EOF
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
    "https://github.com/nrk/phpiredis/archive/v#{version}.tar.gz"
  end

  def local_path
    "phpiredis-#{version}.tar.gz"
  end
end

class RedisPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      '--enable-redis-igbinary',
      '--enable-redis-lzf',
      '--with-liblzf=no'
    ]
  end
end

# TODO: Remove after PHP 7 is out of support
class PHPProtobufPeclRecipe < PeclRecipe
  def url
    "https://github.com/allegro/php-protobuf/archive/v#{version}.tar.gz"
  end

  def local_path
    "php-protobuf-#{version}.tar.gz"
  end
end

class TidewaysXhprofRecipe < PeclRecipe
  def url
    "https://github.com/tideways/php-xhprof-extension/archive/v#{version}.tar.gz"
  end

  def local_path
    "tideways-xhprof-#{version}.tar.gz"
  end
end

class EnchantFakePeclRecipe < FakePeclRecipe
  def patch
    super
    system <<-EOF
      cd #{work_path}
      sed -i 's|#include "../spl/spl_exceptions.h"|#include <spl/spl_exceptions.h>|' enchant.c
    EOF
  end
end

class RabbitMQRecipe < PkgConfigLibRecipe
  def url
    "https://github.com/alanxz/rabbitmq-c/archive/v#{version}.tar.gz"
  end

  def pkgcfg_name
    'librabbitmq'
  end

  def local_path
    "rabbitmq-#{version}.tar.gz"
  end

  def work_path
    File.join(tmp_path, "rabbitmq-c-#{@version}")
  end

  def configure; end

  def compile
    execute('compile', ['bash', '-c', 'cmake .'])
    execute('compile', ['bash', '-c', 'cmake --build .'])
    execute('compile', ['bash', '-c', 'cmake -DCMAKE_INSTALL_PREFIX=/usr/local .'])
    execute('compile', ['bash', '-c', 'cmake --build . --target install'])
  end
end

class SnmpRecipe
  attr_reader :name, :version

  def initialize(name, version, options)
    @name = name
    @version = version
    @options = options
  end

  def files_hashs
    []
  end

  def cook
    system <<-EOF
      cd #{@php_path}
      mkdir -p mibs
      cp -a /usr/lib/x86_64-linux-gnu/libnetsnmp.so* lib/
      # copy mibs that are packaged freely
      cp -r /usr/share/snmp/mibs/* mibs
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
      # TODO: these are gone or have moved, commenting out for now
      # mkdir mibs/originals
      # cp -R /usr/share/doc/mibiana mibs/originals
      # cp -R /usr/share/doc/mibrfcs mibs/originals
    EOF
  end
end

class SuhosinPeclRecipe < PeclRecipe
  def url
    "https://github.com/sektioneins/suhosin/archive/#{version}.tar.gz"
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
