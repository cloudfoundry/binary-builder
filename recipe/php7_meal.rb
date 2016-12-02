# encoding: utf-8
require_relative 'php_common'

class Php7Recipe < BaseRecipe
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
      '--with-mysqli=shared',
      '--enable-pdo=shared',
      '--with-pdo-sqlite=shared,/usr',
      '--with-pdo-mysql=shared,mysqlnd',
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
      '--with-readline'
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
    "php7-#{version}-linux-x64-#{Time.now.utc.to_i}.tgz"
  end

  def setup_tar
    system <<-eof
      cp -a /usr/local/lib/x86_64-linux-gnu/librabbitmq.so* #{path}/lib/
      cp -a #{@hiredis_path}/lib/libhiredis.so* #{path}/lib/
      cp #{@ioncube_path}/ioncube/ioncube_loader_lin_#{major_version}.so #{zts_path}/ioncube.so
      cp -a /usr/lib/libc-client.so* #{path}/lib/
      cp -a /usr/lib/libmcrypt.so* #{path}/lib
      cp -a /usr/lib/libaspell.so* #{path}/lib
      cp -a /usr/lib/libpspell.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libmemcached.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libcassandra.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libuv.so* #{path}/lib
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

class Php7Meal
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
        libaspell-dev \
        libc-client2007e-dev \
        libcurl4-openssl-dev \
        libexpat1-dev \
        libgdbm-dev \
        libgmp-dev \
        libjpeg-dev \
        libldap2-dev \
        libmcrypt-dev \
        libmemcached-dev \
        libpng12-dev \
        libpspell-dev \
        libreadline-dev \
        libsasl2-dev \
        libsnmp-dev \
        libsqlite3-dev \
        libssl-dev \
        libxml2-dev \
        libzip-dev \
        libzookeeper-mt-dev \
        snmp-mibs-downloader
      sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
    eof

    install_cassandra_dependencies

    ioncube_recipe.cook

    php_recipe.cook
    php_recipe.activate

    # native dependencies
    hiredis_recipe.cook
    phpiredis_recipe.cook
    rabbitmq_recipe.cook
    lua_recipe.cook
    snmp_recipe.cook
    librdkafka_recipe.cook

    # php extensions
    standard_pecl('apcu', '5.1.7', '7803b58fab6ecfe847ef5b9be6825dea')
    standard_pecl('cassandra', '1.2.2', '2226a4d66f8e0a4de85656f10472afc5')
    standard_pecl('imagick', '3.4.3RC1', '32042fc3043f013047927de21ff15a47')
    standard_pecl('mailparse', '3.0.1', '5ae0643a11159414c7e790c73a9e25ec')
    standard_pecl('mongodb', '1.1.9', '0644ad0451e5913cbac22e3456ba239b')
    standard_pecl('msgpack', '2.0.1', '4d1db4592ffa4101601aefc794191de5')
    standard_pecl('rdkafka', '2.0.0', '87bce41f61818fd7bc442f71d4c28cde')
    standard_pecl('redis', '3.0.0', '1b90e954afc1f9993cc0552d0f1d1daa')
    standard_pecl('solr', '2.4.0', '2c9accf66681a3daaaf371bc07e44902')
    standard_pecl('xdebug', '2.4.1', '03f52af10108450942c9c0ac3b72637f')
    standard_pecl('yaf', '3.0.4', '1420d91ca5deb31147b25bd08124e400')
    amqppecl_recipe.cook
    luapecl_recipe.cook
    phalcon_recipe.cook

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
    amqppecl_recipe.send(:files_hashs) +
      hiredis_recipe.send(:files_hashs) +
      librdkafka_recipe.send(:files_hashs) +
      lua_recipe.send(:files_hashs) +
      luapecl_recipe.send(:files_hashs) +
      phalcon_recipe.send(:files_hashs) +
      phpiredis_recipe.send(:files_hashs) +
      rabbitmq_recipe.send(:files_hashs) +
      (OraclePeclRecipe.oracle_sdk? ? oracle_recipe.send(:files_hashs) : []) +
      (OraclePeclRecipe.oracle_sdk? ? oracle_pdo_recipe.send(:files_hashs) : []) +
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

  def php_recipe
    @php_recipe ||= Php7Recipe.new(@name, @version, {
      hiredis_path: hiredis_recipe.path,
      ioncube_path: ioncube_recipe.path
    }.merge(DetermineChecksum.new(@options).to_h))
  end

  def ioncube_recipe
    @ioncube ||= IonCubeRecipe.new('ioncube', '6.0.6', md5: '7d2b42033a0570e99080beb6a7db1478')
  end

  def luapecl_recipe
    @luapecl_recipe ||= LuaPeclRecipe.new('lua', '2.0.2', md5: 'beb0c9b1c6ed2457d614607c8a1537af',
                                                          php_path: php_recipe.path,
                                                          lua_path: lua_recipe.path)
  end

  def oracle_recipe
    @oracle_recipe ||= OraclePeclRecipe.new('oci8', '2.1.1', md5: '01bb3429ce3206dcc3d3198e65dadfbc',
							     php_path: php_recipe.path)
  end

  def oracle_pdo_recipe
    @oracle_pdo_recipe ||= OraclePdoRecipe.new('pdo_oci', version,
                                               php_source: "#{php_recipe.send(:tmp_path)}/php-#{version}",
                                               php_path: php_recipe.path)
  end

  def phalcon_recipe
    @phalcon_recipe ||= PhalconRecipe.new('phalcon', '3.0.1', md5: '4a67015af27eb4fbb4e32c23d2610815',
                                                              php_path: php_recipe.path)
    @phalcon_recipe.set_php_version('php7')
    @phalcon_recipe
  end
end
