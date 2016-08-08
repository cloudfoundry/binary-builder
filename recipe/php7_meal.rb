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
      cp -a /usr/lib/libc-client.so* #{path}/lib/
      cp -a /usr/lib/libmcrypt.so* #{path}/lib
      cp -a /usr/lib/libaspell.so* #{path}/lib
      cp -a /usr/lib/libpspell.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libmemcached.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libcassandra.so* #{path}/lib
      cp -a /usr/lib/x86_64-linux-gnu/libuv.so* #{path}/lib

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

    php_recipe.cook
    php_recipe.activate

    # native dependencies
    rabbitmq_recipe.cook
    lua_recipe.cook
    snmp_recipe.cook

    # php extensions
    standard_pecl('imagick', '3.4.2', '3f80e35c2434636cdb5df01b221b3ffa')
    standard_pecl('mailparse', '3.0.1', '5ae0643a11159414c7e790c73a9e25ec')
    standard_pecl('mongodb', '1.1.8', '94f0f8d551f5ca440420aa3db0054dd8')
    standard_pecl('msgpack', '2.0.1', '4d1db4592ffa4101601aefc794191de5')
    standard_pecl('solr', '2.4.0', '2c9accf66681a3daaaf371bc07e44902')
    standard_pecl('xdebug', '2.4.0', 'f49fc01332468f8b753fb37115505fb5')
    standard_pecl('yaf', '3.0.3', '74504e8f0ed7c346804c5a5043a5682b')
    standard_pecl('cassandra', '1.2.1', 'dca2cda61a1ff6a6cecb94f88a75c757')
    amqppecl_recipe.cook
    luapecl_recipe.cook
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
      lua_recipe.send(:files_hashs) +
      luapecl_recipe.send(:files_hashs) +
      amqppecl_recipe.send(:files_hashs) +
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
    }.merge(DetermineChecksum.new(@options).to_h))
  end

  def luapecl_recipe
    @luapecl_recipe ||= LuaPeclRecipe.new('lua', '2.0.1', md5: '56924db266f3748a0432328e764b7782',
                                                          php_path: php_recipe.path,
                                                          lua_path: lua_recipe.path)
  end
end
