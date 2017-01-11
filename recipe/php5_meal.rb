# encoding: utf-8
require_relative 'php_common'

class Php5Recipe < BaseRecipe
  def configure_options
    [
      '--disable-static',
      '--enable-shared',
      '--enable-ftp=shared',
      '--enable-sockets=shared',
      '--enable-soap=shared',
      '--enable-fileinfo=shared',
      '--enable-bcmath',
      '--enable-calendar',
      '--enable-intl',
      '--with-kerberos',
      '--enable-zip=shared',
      '--with-bz2=shared',
      '--with-curl=shared',
      '--enable-dba=shared',
      '--with-cdb',
      '--with-gdbm',
      '--with-mcrypt=shared',
      '--with-mhash=shared',
      '--with-mysql=shared',
      '--with-mysqli=shared',
      '--enable-pdo=shared',
      '--with-pdo-sqlite=shared,/usr',
      '--with-pdo-mysql=shared,mysqlnd',
      '--with-mssql=shared',
      '--with-pdo-dblib=shared',
      '--with-gd=shared',
      '--with-jpeg-dir=/usr',
      '--with-freetype-dir=/usr',
      '--enable-gd-native-ttf',
      '--with-pdo-pgsql=shared',
      '--with-pgsql=shared',
      '--with-pspell=shared',
      '--with-gettext=shared',
      '--with-gmp=shared',
      '--with-imap=shared',
      '--with-imap-ssl=shared',
      '--with-ldap=shared',
      '--with-ldap-sasl',
      '--with-zlib=shared',
      '--with-xsl=shared',
      '--with-snmp=shared',
      '--enable-mbstring=shared',
      '--enable-mbregex',
      '--enable-exif=shared',
      '--with-openssl=shared',
      '--enable-fpm',
      '--enable-pcntl=shared',
      '--with-readline=shared'
    ]
  end

  def url
    "https://php.net/distributions/php-#{version}.tar.gz"
  end

  def archive_files
    ["#{port_path}/*"]
  end

  def archive_path_name
    'php'
  end

  def configure
    return if configured?

    md5_file = File.join(tmp_path, 'configure.md5')
    digest   = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, 'w') { |f| f.write digest }

    # LIBS=-lz enables using zlib when configuring
    execute('configure', ['bash', '-c', "LIBS=-lz ./configure #{computed_options.join ' '}"])
  end

  def major_version
    @major_version ||= version.match(/^(\d+\.\d+)/)[1]
  end

  def zts_path
    Dir["#{path}/lib/php/extensions/no-debug-non-zts-*"].first
  end

  def archive_filename
    "php-#{version}-linux-x64-#{Time.now.utc.to_i}.tgz"
  end

  def setup_tar
  system <<-eof
      cp -a #{@hiredis_path}/lib/libhiredis.so* #{path}/lib/
      cp #{@ioncube_path}/ioncube/ioncube_loader_lin_#{major_version}.so #{zts_path}/ioncube.so
      cp -a #{@libmemcached_path}/lib/libmemcached.so* #{path}/lib/
      cp -a /usr/lib/libc-client.so* #{path}/lib/
      cp -a /usr/lib/libmcrypt.so* #{path}/lib
      cp -a /usr/lib/libaspell.so* #{path}/lib
      cp -a /usr/lib/libpspell.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libgearman.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libcassandra.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libuv.so* #{path}/lib
      cp -a /usr/local/lib/x86_64-linux-gnu/librabbitmq.so* #{path}/lib/
      cp -a /usr/lib/x86_64-linux-gnu/libsybdb.so* #{path}/lib/
      cp -a /usr/lib/librdkafka.so* #{path}/lib

      # Remove unused files
      rm "#{path}/etc/php-fpm.conf.default"
      rm -rf "#{path}/include"
      rm -rf "#{path}/php"
      rm -rf "#{path}/lib/php/build"
      rm "#{path}/bin/php-cgi"
      find "#{path}/lib/php/extensions" -name "*.a" -type f -delete
    eof
  end
end

class Php5Meal
  attr_reader :name, :version

  def initialize(name, version, options)
    @name    = name
    @version = version
    @options = options
  end

  def cook
    system <<-eof
      sudo apt-get update
      sudo apt-get -y upgrade
      sudo apt-get -y install \
        automake \
        freetds-dev \
        libaspell-dev \
        libc-client2007e-dev \
        libcurl4-openssl-dev \
        libexpat1-dev \
        libgdbm-dev \
        libgearman-dev \
        libgmp-dev \
        libjpeg-dev \
        libldap2-dev \
        libmcrypt-dev \
        libpng12-dev \
        libpspell-dev \
        libreadline-dev \
        libsasl2-dev \
        libsnmp-dev \
        libsqlite3-dev \
        libssl-dev \
        libsybdb5 \
        libxml2-dev \
        libzip-dev \
        libzookeeper-mt-dev \
        snmp-mibs-downloader
      sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libsybdb.so /usr/lib/libsybdb.so
    eof

    install_cassandra_dependencies

    ioncube_recipe.cook

    php_recipe.cook
    php_recipe.activate

    # native libraries
    rabbitmq_recipe.cook
    lua_recipe.cook
    hiredis_recipe.cook
    phpiredis_recipe.cook
    snmp_recipe.cook
    libmemcached_recipe.cook
    librdkafka_recipe.cook

    # php extensions
    standard_pecl('apcu', '4.0.11', '13c0c0dd676e5a7905d54fa985d0ee62')
    standard_pecl('cassandra', '1.2.2', '2226a4d66f8e0a4de85656f10472afc5')
    standard_pecl('igbinary', '1.2.1', '04a2474ff5eb99c7d0007bf9f4e8a6ec')
    standard_pecl('imagick', '3.4.3RC1', '32042fc3043f013047927de21ff15a47')
    standard_pecl('gearman', '1.1.2', 'fb3bc8df2d017048726d5654459e8433')
    standard_pecl('rdkafka', '3.0.0', 'c798343029fd4a7c8fe3fae365d438df')
    standard_pecl('mailparse', '2.1.6', '0f84e1da1d074a4915a9bcfe2319ce84')
    standard_pecl('memcache', '2.2.7', '171e3f51a9afe18b76348ddf1c952141')
    standard_pecl('mongo', '1.6.14', '19cd8bd94494f924ce8314f304fd83b6')
    standard_pecl('mongodb', '1.1.9', '0644ad0451e5913cbac22e3456ba239b')
    standard_pecl('msgpack', '0.5.7', 'b87b5c5e0dab9f41c824201abfbf363d')
    standard_pecl('protocolbuffers', '0.2.6', 'a304ca632b0d7c5710d5590ac06248a9')
    standard_pecl('redis', '2.2.8', 'b6c998a6904cb89b06281e1cfb89bc4d')
    standard_pecl('solr', '2.4.0', '2c9accf66681a3daaaf371bc07e44902')
    standard_pecl('sundown', '0.3.11', 'c1397e9d3312226ec6c84e8e34c717a6')
    standard_pecl('xdebug', '2.5.0', '03f52af10108450942c9c0ac3b72637f')
    standard_pecl('yaf', '2.3.5', '77d5d9d6c8471737395350966986bc2e')
    amqppecl_recipe.cook
    luapecl_recipe.cook
    phpprotobufpecl_recipe.cook
    phalcon_recipe.cook
    suhosinpecl_recipe.cook
    twigpecl_recipe.cook
    xcachepecl_recipe.cook
    xhprofpecl_recipe.cook
    memcachedpecl_recipe.cook

    if OraclePeclRecipe.oracle_sdk?
      system 'ln -s /oracle/libclntsh.so.* /oracle/libclntsh.so'

      oracle_recipe.cook
      oracle_pdo_recipe.cook
    end
  end

  def url
    php_recipe.url
  end

  def archive_files
    php_recipe.archive_files
  end

  def archive_path_name
    php_recipe.archive_path_name
  end

  def archive_filename
    php_recipe.archive_filename
  end

  def setup_tar
    php_recipe.setup_tar
    if OraclePeclRecipe.oracle_sdk?
      oracle_recipe.setup_tar
      oracle_pdo_recipe.setup_tar
    end
  end

  private

  def files_hashs
      rabbitmq_recipe.send(:files_hashs) +
      amqppecl_recipe.send(:files_hashs) +
      lua_recipe.send(:files_hashs) +
      luapecl_recipe.send(:files_hashs) +
      hiredis_recipe.send(:files_hashs) +
      librdkafka_recipe.send(:files_hashs) +
      phpiredis_recipe.send(:files_hashs) +
      phpprotobufpecl_recipe.send(:files_hashs) +
      phalcon_recipe.send(:files_hashs) +
      suhosinpecl_recipe.send(:files_hashs) +
      twigpecl_recipe.send(:files_hashs) +
      xcachepecl_recipe.send(:files_hashs) +
      xhprofpecl_recipe.send(:files_hashs) +
      (OraclePeclRecipe.oracle_sdk? ? oracle_recipe.send(:files_hashs) : []) +
      (OraclePeclRecipe.oracle_sdk? ? oracle_pdo_recipe.send(:files_hashs) : []) +
      libmemcached_recipe.send(:files_hashs) +
      memcachedpecl_recipe.send(:files_hashs) +
      @pecl_recipes.collect { |r| r.send(:files_hashs) }.flatten
  end

  def standard_pecl(name, version, md5)
    @pecl_recipes ||= []
    recipe = PeclRecipe.new(name, version, md5: md5,
                                           php_path: php_recipe.path)
    recipe.cook
    @pecl_recipes << recipe
  end

  def snmp_recipe
    SnmpRecipe.new(php_recipe.path)
  end

  def libmemcached_recipe
    @libmemcached_recipe ||= LibmemcachedRecipe.new('libmemcached', '1.0.18')
  end

  def memcachedpecl_recipe
    @memcachedpecl_recipe ||= MemcachedPeclRecipe.new('memcached', '2.2.0', php_path: php_recipe.path, libmemcached_path: libmemcached_recipe.path)
  end

  def php_recipe
    @php_recipe ||= Php5Recipe.new(@name, @version, {
      rabbitmq_path: File.join(rabbitmq_recipe.path, "rabbitmq-c-#{rabbitmq_recipe.version}", 'librabbitmq'),
      hiredis_path: hiredis_recipe.path,
      libmemcached_path: libmemcached_recipe.path,
      ioncube_path: ioncube_recipe.path
    }.merge(DetermineChecksum.new(@options).to_h))
  end

  def luapecl_recipe
    @luapecl_recipe ||= LuaPeclRecipe.new('lua', '1.1.0', md5: '58bd532957473f2ac87f1032c4aa12b5',
                                                          php_path: php_recipe.path,
                                                          lua_path: lua_recipe.path)
  end


  def phpprotobufpecl_recipe
    @phpprotobufpecl_recipe ||= PHPProtobufPeclRecipe.new('phpprotobuf', '0.11.1', md5: 'adbf5214bfd44ce18962dd49f5640552',
                                                                                       php_path: php_recipe.path)
  end

  def ioncube_recipe
    @ioncube ||= IonCubeRecipe.new('ioncube', '6.0.6', md5: '7d2b42033a0570e99080beb6a7db1478')
  end

  def phalcon_recipe
    @phalcon_recipe ||= PhalconRecipe.new('phalcon', '3.0.2', md5: '43e2aa0360af1787db03f5cc6cd1b676',
                                                              php_path: php_recipe.path)
    @phalcon_recipe.set_php_version('php5')
    @phalcon_recipe
  end

  def suhosinpecl_recipe
    @suhosinpecl_recipe ||= SuhosinPeclRecipe.new('suhosin', '0.9.38', md5: '0c26402752b0aff69e4b891f062a52bf',
                                                                         php_path: php_recipe.path)
  end

  def twigpecl_recipe
    @twigpecl_recipe ||= TwigPeclRecipe.new('twig', '1.27.0', md5: '9f1f740e3fd0570b16a8b150fb0380de',
                                                              php_path: php_recipe.path)
  end

  def xcachepecl_recipe
    @xcachepecl_recipe ||= XcachePeclRecipe.new('xcache', '3.2.0', md5: '8b0a6f27de630c4714ca261480f34cda',
                                                                   php_path: php_recipe.path)
  end

  def xhprofpecl_recipe
    @xhprofpecl_recipe ||= XhprofPeclRecipe.new('xhprof', '0bbf2a2ac3', md5: '1df4aebf1cb24e7cf369b3af357106fc',
                                                                        php_path: php_recipe.path)
  end

  def oracle_recipe
    @oracle_recipe ||= OraclePeclRecipe.new('oci8', '2.0.11', md5: 'b953aec8600b1990fc1956bd5f580b0b',
                                                              php_path: php_recipe.path)
  end

  def oracle_pdo_recipe
    @oracle_pdo_recipe ||= OraclePdoRecipe.new('pdo_oci', version,
                                               php_source: "#{php_recipe.send(:tmp_path)}/php-#{version}",
                                               php_path: php_recipe.path)
  end
end
