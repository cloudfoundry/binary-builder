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
      cp -a #{@rabbitmq_path}/lib/librabbitmq.so* #{path}/lib/
      cp -a #{@hiredis_path}/lib/libhiredis.so* #{path}/lib/
      cp #{@ioncube_path}/ioncube_loader_lin_#{major_version}.so #{zts_path}/ioncube.so
      cp -a #{@libmemcached_path}/lib/libmemcached.so* #{path}/lib/
      cp -a /usr/lib/libc-client.so* #{path}/lib/
      cp -a /usr/lib/libmcrypt.so* #{path}/lib
      cp -a /usr/lib/libaspell.so* #{path}/lib
      cp -a /usr/lib/libpspell.so* #{path}/lib

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
        libaspell-dev \
        libc-client2007e-dev \
        libcurl4-openssl-dev \
        libexpat1-dev \
        libgdbm-dev \
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
        libxml2-dev \
        libzip-dev \
        libzookeeper-mt-dev \
        snmp-mibs-downloader
      sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
    eof

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

    # php extensions
    standard_pecl('igbinary', '1.2.1', '04a2474ff5eb99c7d0007bf9f4e8a6ec')
    standard_pecl('imagick', '3.4.1', 'cc4f119a5f27b582f0f10e61451e266f')
    standard_pecl('mailparse', '2.1.6', '0f84e1da1d074a4915a9bcfe2319ce84')
    standard_pecl('memcache', '2.2.7', '171e3f51a9afe18b76348ddf1c952141')
    standard_pecl('mongo', '1.6.14', '19cd8bd94494f924ce8314f304fd83b6')
    standard_pecl('msgpack', '0.5.7', 'b87b5c5e0dab9f41c824201abfbf363d')
    standard_pecl('protocolbuffers', '0.2.6', 'a304ca632b0d7c5710d5590ac06248a9')
    standard_pecl('redis', '2.2.7', 'c55839465b2c435fd091ac50923f2d96')
    standard_pecl('sundown', '0.3.11', 'c1397e9d3312226ec6c84e8e34c717a6')
    standard_pecl('xdebug', '2.4.0', 'f49fc01332468f8b753fb37115505fb5')
    standard_pecl('yaf', '2.3.5', '77d5d9d6c8471737395350966986bc2e')
    standard_pecl('solr', '2.4.0', '2c9accf66681a3daaaf371bc07e44902')
    amqppecl_recipe.cook
    luapecl_recipe.cook
    phpprotobufpecl_recipe.cook
    phalconpecl_recipe.cook
    suhosinpecl_recipe.cook
    twigpecl_recipe.cook
    xcachepecl_recipe.cook
    xhprofpecl_recipe.cook
    memcachedpecl_recipe.cook
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
  end

  private

  def files_hashs
    rabbitmq_recipe.send(:files_hashs) +
      amqppecl_recipe.send(:files_hashs) +
      lua_recipe.send(:files_hashs) +
      luapecl_recipe.send(:files_hashs) +
      hiredis_recipe.send(:files_hashs) +
      phpiredis_recipe.send(:files_hashs) +
      phpprotobufpecl_recipe.send(:files_hashs) +
      phalconpecl_recipe.send(:files_hashs) +
      suhosinpecl_recipe.send(:files_hashs) +
      twigpecl_recipe.send(:files_hashs) +
      xcachepecl_recipe.send(:files_hashs) +
      xhprofpecl_recipe.send(:files_hashs) +
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
      rabbitmq_path: rabbitmq_recipe.path,
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

  def hiredis_recipe
    @hiredis_recipe ||= HiredisRecipe.new('hiredis', '0.13.3', md5: '43dca1445ec6d3b702821dba36000279')
  end

  def phpiredis_recipe
    @phpiredis_recipe ||= PHPIRedisRecipe.new('phpiredis', '704c08c7b', md5: '1ea635f3712aa1b80245eeed2d570a0e',
                                                                        php_path: php_recipe.path,
                                                                        hiredis_path: hiredis_recipe.path)
  end

  def phpprotobufpecl_recipe
    @phpprotobufpecl_recipe ||= PHPProtobufPeclRecipe.new('phpprotobuf', 'd792f5b8e0', md5: '32d0febec95218348b34b74ede028d18',
                                                                                       php_path: php_recipe.path)
  end

  def ioncube_recipe
    @ioncube ||= IonCubeRecipe.new('ioncube', '5.1.2', md5: 'dbff6dcfde17c34c9d38fe5adabf939b')
  end

  def phalconpecl_recipe
    @phalconpecl_recipe ||= PhalconPeclRecipe.new('phalcon', '2.0.11', md5: 'b644ac4915e95b6cec7dd4834fd9e127',
                                                                      php_path: php_recipe.path)
  end

  def suhosinpecl_recipe
    @suhosinpecl_recipe ||= SuhosinPeclRecipe.new('suhosin', '0.9.38', md5: '0c26402752b0aff69e4b891f062a52bf',
                                                                         php_path: php_recipe.path)
  end

  def twigpecl_recipe
    @twigpecl_recipe ||= TwigPeclRecipe.new('twig', '1.24.0', md5: 'ff6a06115a36975770e08d62992f5557',
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
end
