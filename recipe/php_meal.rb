class ChecksumRecipe < BaseRecipe
  def initialize(*)
    super
    @files = [{
      url: self.url,
      md5: @md5
    }]
  end
end

class RabbitMQRecipe < ChecksumRecipe
  def url
    "https://github.com/alanxz/rabbitmq-c/releases/download/v#{version}/rabbitmq-c-#{version}.tar.gz"
  end
end

class PeclRecipe < ChecksumRecipe
  def url
    "http://pecl.php.net/get/#{name}-#{version}.tgz"
  end

  def configure
    return if configured?

    md5_file = File.join(tmp_path, 'configure.md5')
    digest   = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, "w") { |f| f.write digest }

    execute('configure', 'phpize')
    execute('configure', %w(sh configure) + computed_options)
  end
end

class LuaRecipe < ChecksumRecipe
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

    execute('install', ['bash', '-c', "#{make_cmd} install INSTALL_TOP=#{self.path}"])
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

class LuaPeclRecipe < PeclRecipe
  def configure_options
    [
      "--with-php-config=#{@php_path}/bin/php-config",
      "--with-lua=#{@lua_path}"
    ]
  end
end

class PhpRecipe < BaseRecipe
  def initialize(name, version, options={})
    super name, version

    @rabbitmq_path = options[:rabbitmq_path]
  end

  def configure_options
    [
      "--disable-static",
      "--enable-shared",
      "--enable-ftp=shared",
      "--enable-sockets=shared",
      "--enable-soap=shared",
      "--enable-fileinfo=shared",
      "--enable-bcmath",
      "--enable-calendar",
      "--with-kerberos",
      "--enable-zip=shared",
      "--with-bz2=shared",
      "--with-curl=shared",
      "--enable-dba=shared",
      "--with-cdb",
      "--with-gdbm",
      "--with-mcrypt=shared",
      "--with-mhash=shared",
      "--with-mysql=shared",
      "--with-mysqli=shared",
      "--enable-pdo=shared",
      "--with-pdo-sqlite=shared,/usr",
      "--with-pdo-mysql=shared,mysqlnd",
      "--with-gd=shared",
      "--with-jpeg-dir=/usr",
      "--with-freetype-dir=/usr",
      "--enable-gd-native-ttf",
      "--with-pdo-pgsql=shared",
      "--with-pgsql=shared",
      "--with-pspell=shared",
      "--with-gettext=shared",
      "--with-gmp=shared",
      "--with-imap=shared",
      "--with-imap-ssl=shared",
      "--with-ldap=shared",
      "--with-ldap-sasl",
      "--with-zlib=shared",
      "--with-xsl=shared",
      "--with-snmp=shared",
      "--enable-mbstring=shared",
      "--enable-mbregex",
      "--enable-exif=shared",
      "--with-openssl=shared",
      "--enable-fpm",
      "--enable-pcntl=shared",
      "--with-readline=shared"
    ]
  end

  def url
    "https://php.net/get/php-#{version}.tar.bz2/from/this/mirror"
  end

  def archive_files
    [ "#{port_path}/*" ]
  end

  def archive_path_name
    "php"
  end

  def configure
    return if configured?

    md5_file = File.join(tmp_path, 'configure.md5')
    digest   = Digest::MD5.hexdigest(computed_options.to_s)
    File.open(md5_file, "w") { |f| f.write digest }

    #LIBS=-lz enables using zlib when configuring
    execute('configure',["bash","-c","LIBS=-lz ./configure #{computed_options.join ' '}"])
  end

  def tar
    system <<-eof
      cp #{@rabbitmq_path}/lib/librabbitmq.so.1 #{self.path}/lib/
    eof
    super
  end
end

class PhpMeal
  attr_accessor :files

  def initialize(name, version)
    @name    = name
    @version = version
    @files   = []
  end

  def cook
    system <<-eof
      sudo apt-get update
      sudo apt-get -y upgrade
      sudo apt-get -y install \
        automake \
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
        mercurial \
        snmp-mibs-downloader
      sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
      sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
    eof

    php_recipe.files = self.files
    php_recipe.cook
    php_recipe.activate

    rabbitmq_recipe.cook
    amqppecl_recipe.cook
    lua_recipe.cook
    luapecl_recipe.cook

    php_recipe.tar
  end

  def url
   php_recipe.url
  end


  private

  def php_recipe
    @php_recipe ||= PhpRecipe.new(@name, @version,
                                  rabbitmq_path: rabbitmq_recipe.path
                                 )
  end

  def rabbitmq_recipe
    @rabbitmq_recipe ||= RabbitMQRecipe.new('rabbitmq', '0.5.2',
                                            md5: 'aa8d4d0b949f508c0da25a9c20bd7da7'
                                           )
  end

  def lua_recipe
    @lua_recipe ||= LuaRecipe.new('lua', '5.2.4',
                                  md5: '913fdb32207046b273fdb17aad70be13'
                                 )
  end

  def luapecl_recipe
    @luapecl_recipe ||= LuaPeclRecipe.new('lua', '1.1.0',
                                          md5: '58bd532957473f2ac87f1032c4aa12b5',
                                          php_path: php_recipe.path,
                                          lua_path: lua_recipe.path
                                         )
  end

  def amqppecl_recipe
    @amqppecl_recipe ||= AmqpPeclRecipe.new('amqp', '1.4.0',
                                            md5: 'e7fefbd5c87eaad40c29e2ad5de7bd30',
                                            php_path: php_recipe.path,
                                            rabbitmq_path: rabbitmq_recipe.path
                                           )
  end
end


