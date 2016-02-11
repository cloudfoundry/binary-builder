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
      cp #{@rabbitmq_path}/lib/librabbitmq.so.1 #{path}/lib/
      cp #{@hiredis_path}/lib/libhiredis.so.0.10 #{path}/lib/
      cp #{@ioncube_path}/ioncube_loader_lin_#{major_version}.so #{zts_path}/ioncube.so
      cp /usr/lib/libc-client.so.2007e #{path}/lib/
      cp /usr/lib/libmcrypt.so.4 #{path}/lib
      cp /usr/lib/libaspell.so.15 #{path}/lib
      cp /usr/lib/libpspell.so.15 #{path}/lib
      cp /usr/lib/x86_64-linux-gnu/libmemcached.so.10 #{path}/lib

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
        libmemcached-dev \
        libpng12-dev \
        libpspell-dev \
        libreadline-dev \
        libsasl2-dev \
        libsnmp-dev \
        libsqlite3-dev \
        libssl-dev \
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

    standard_pecl('intl', '3.0.0', 'a6029b9e7b1d0fcdb6e8bfad49e59ae9')
    standard_pecl('igbinary', '1.2.1', '04a2474ff5eb99c7d0007bf9f4e8a6ec')
    standard_pecl('imagick', '3.1.2', 'f2fd71b026debe056e0ec8d76c2ffe94')
    standard_pecl('mailparse', '2.1.6', '0f84e1da1d074a4915a9bcfe2319ce84')
    standard_pecl('memcache', '2.2.7', '171e3f51a9afe18b76348ddf1c952141')
    standard_pecl('mongo', '1.6.5', '058b5d76c95e1b12267cf1b449118acc')
    standard_pecl('msgpack', '0.5.5', 'adc8d9ea5088bdb83e7cc7c2f535d858')
    standard_pecl('protocolbuffers', '0.2.6', 'a304ca632b0d7c5710d5590ac06248a9')
    standard_pecl('redis', '2.2.7', 'c55839465b2c435fd091ac50923f2d96')
    standard_pecl('sundown', '0.3.11', 'c1397e9d3312226ec6c84e8e34c717a6')
    standard_pecl('xdebug', '2.3.1', '117d8e54d84b1cb7e07a646377007bd5')
    standard_pecl('yaf', '2.3.3', '942dc4109ad965fa7f09fddfc784f335')

    rabbitmq_recipe.cook
    amqppecl_recipe.cook
    lua_recipe.cook
    luapecl_recipe.cook
    hiredis_recipe.cook
    phpiredis_recipe.cook
    phpprotobufpecl_recipe.cook
    phalconpecl_recipe.cook
    suhosinpecl_recipe.cook
    twigpecl_recipe.cook
    xcachepecl_recipe.cook
    xhprofpecl_recipe.cook
    memcachedpecl_recipe.cook
    snmp_recipe.cook
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

  def memcachedpecl_recipe
    @memcachedpecl_recipe ||= MemcachedPeclRecipe.new('memcached', '2.2.0', php_path: php_recipe.path)
  end

  def php_recipe
    @php_recipe ||= Php5Recipe.new(@name, @version, {
      rabbitmq_path: rabbitmq_recipe.path,
      hiredis_path: hiredis_recipe.path,
      ioncube_path: ioncube_recipe.path
    }.merge(DetermineChecksum.new(@options).to_h))
  end

  def rabbitmq_recipe
    @rabbitmq_recipe ||= RabbitMQRecipe.new('rabbitmq', '0.5.2', md5: 'aa8d4d0b949f508c0da25a9c20bd7da7')
  end

  def lua_recipe
    @lua_recipe ||= LuaRecipe.new('lua', '5.2.4', md5: '913fdb32207046b273fdb17aad70be13')
  end

  def luapecl_recipe
    @luapecl_recipe ||= LuaPeclRecipe.new('lua', '1.1.0', md5: '58bd532957473f2ac87f1032c4aa12b5',
                                                          php_path: php_recipe.path,
                                                          lua_path: lua_recipe.path)
  end

  def amqppecl_recipe
    @amqppecl_recipe ||= AmqpPeclRecipe.new('amqp', '1.4.0',                                               md5: 'e7fefbd5c87eaad40c29e2ad5de7bd30',
                                                                                                           php_path: php_recipe.path,
                                                                                                           rabbitmq_path: rabbitmq_recipe.path)
  end

  def hiredis_recipe
    @hiredis_recipe ||= HiredisRecipe.new('hiredis', '0.11.0',                                             md5: 'e2ac29509823ccc96990b6fe765b5d46')
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
    @ioncube ||= IonCubeRecipe.new('ioncube', '5.0.22', md5: 'c165f53a817c0ada6d87003529e4a55d')
  end

  def phalconpecl_recipe
    @phalconpecl_recipe ||= PhalconPeclRecipe.new('phalcon', '1.3.4', md5: '36ec688a6fb710ce4b1e34c00bf24748',
                                                                      php_path: php_recipe.path)
  end

  def suhosinpecl_recipe
    @suhosinpecl_recipe ||= SuhosinPeclRecipe.new('suhosin', '0.9.37.1', md5: '8d1c37e62ff712638b5d3847d94bfab3',
                                                                         php_path: php_recipe.path)
  end

  def twigpecl_recipe
    @twigpecl_recipe ||= TwigPeclRecipe.new('twig', '1.18.0', md5: '294f9606acc7170decfad27575fa1d00',
                                                              php_path: php_recipe.path)
  end

  def xcachepecl_recipe
    @xcachepecl_recipe ||= XcachePeclRecipe.new('xcache', '3.2.0', md5: '8b0a6f27de630c4714ca261480f34cda',
                                                                   php_path: php_recipe.path)
  end

  def xhprofpecl_recipe
    @xhprofpecl_recipe ||= XhprofPeclRecipe.new('xhprof', '0bbf2a2ac3',                                                   md5: '1df4aebf1cb24e7cf369b3af357106fc',
                                                                                                                          php_path: php_recipe.path)
  end
end
