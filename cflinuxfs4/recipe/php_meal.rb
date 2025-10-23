# frozen_string_literal: true

require_relative 'php_common_recipes'
require_relative 'php_recipe'

class PhpMeal
  attr_reader :name, :version

  def initialize(name, version, options)
    @name = name
    @version = version
    version_parts = version.split('.')
    @major_version = version_parts[0]
    @minor_version = version_parts[1]
    @options = options
    @native_modules = []
    @extensions = []

    create_native_module_recipes
    create_extension_recipes

    (@native_modules + @extensions).each do |recipe|
      recipe.instance_variable_set('@php_path', php_recipe.path)
      recipe.instance_variable_set('@php_source', "#{php_recipe.send(:tmp_path)}/php-#{@version}")

      if recipe.is_a? FakePeclRecipe
        recipe.instance_variable_set('@version', @version)
        recipe.instance_variable_set('@files', [{ url: recipe.url, md5: nil }])
      end
    end
  end

  def cook
    run('apt-get -y update') or raise 'Failed to apt-get update'
    run('apt-get -y upgrade') or raise 'Failed to apt-get upgrade'
    run("apt-get -y install #{apt_packages}") or raise 'Failed to apt-get install packages'
    symlink_commands

    if OraclePeclRecipe.oracle_sdk?
      Dir.chdir('/oracle') do
        system 'ln -s libclntsh.so.* libclntsh.so'
      end
    end

    php_recipe.cook
    php_recipe.activate

    # native libraries
    @native_modules.each(&:cook)

    # php extensions
    @extensions.each do |recipe|
      recipe.cook if should_cook?(recipe)
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
      @extensions.detect { |r| r.name == 'oci8' }.setup_tar
      @extensions.detect { |r| r.name == 'pdo_oci' }.setup_tar
    end
    @extensions.detect { |r| r.name == 'odbc' }&.setup_tar
    @extensions.detect { |r| r.name == 'pdo_odbc' }&.setup_tar
    @extensions.detect { |r| r.name == 'sodium' }&.setup_tar
  end

  private

  def create_native_module_recipes
    return unless @options[:php_extensions_file]

    php_extensions_hash = YAML.safe_load_file(@options[:php_extensions_file])

    php_extensions_hash['native_modules'].each do |hash|
      klass = Kernel.const_get(hash['klass'])

      @native_modules << klass.new(
        hash['name'],
        hash['version'],
        md5: hash['md5']
      )
    end
  end

  def create_extension_recipes
    return unless @options[:php_extensions_file]

    php_extensions_hash = YAML.safe_load_file(@options[:php_extensions_file])

    php_extensions_hash['extensions'].each do |hash|
      klass = Kernel.const_get(hash['klass'])

      @extensions << klass.new(
        hash['name'],
        hash['version'],
        md5: hash['md5']
      )
    end

    @extensions.each do |recipe|
      case recipe.name
      when 'amqp'
        recipe.instance_variable_set('@rabbitmq_path', @native_modules.detect { |r| r.name == 'rabbitmq' }.work_path)
      when 'lua'
        recipe.instance_variable_set('@lua_path', @native_modules.detect { |r| r.name == 'lua' }.path)
      when 'phpiredis'
        recipe.instance_variable_set('@hiredis_path', @native_modules.detect { |r| r.name == 'hiredis' }.path)
      when 'sodium'
        recipe.instance_variable_set('@libsodium_path', @native_modules.detect { |r| r.name == 'libsodium' }.path)
      end
    end
  end

  def apt_packages
    %w[automake
       firebird-dev
       libaspell-dev
       libc-client2007e-dev
       libcurl4-openssl-dev
       libdb-dev
       libedit-dev
       libenchant-2-dev
       libexpat1-dev
       libgdbm-dev
       libgeoip-dev
       libgmp-dev
       libgpgme11-dev
       libjpeg-dev
       libkrb5-dev
       libldap2-dev
       libmagickwand-dev
       libmagickcore-dev
       libmaxminddb-dev
       libmcrypt-dev
       libmemcached-dev
       libonig-dev
       libpng-dev
       libpspell-dev
       librecode-dev
       libsasl2-dev
       libsnmp-dev
       libsqlite3-dev
       libssh2-1-dev
       libssl-dev
       libtidy-dev
       libtool
       libwebp-dev
       libxml2-dev
       libzip-dev
       libzookeeper-mt-dev
       snmp-mibs-downloader
       sqlite3
       unixodbc-dev].join(' ')
  end

  # @todo: remove this method when all the tests run for cflinuxfs4 without errors
  def install_libuv
    %q((
       if [ "$(pkg-config libuv --print-provides | awk '{print $3}')" != "1.12.0" ]; then
          cd /tmp
          wget http://dist.libuv.org/dist/v1.12.0/libuv-v1.12.0.tar.gz
          tar zxf libuv-v1.12.0.tar.gz
          cd libuv-v1.12.0
          sh autogen.sh
          ./configure
          make install
       fi
       )
    )
  end

  def symlink_commands
    run('sudo ln -s /usr/include/x86_64-linux-gnu/curl /usr/local/include/curl')
    run('sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h')
    run('sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so')
    run('sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so')
  end

  def should_cook?(recipe)
    case recipe.name
    when 'ioncube'
      IonCubeRecipe.build_ioncube?(version)
    when 'oci8', 'pdo_oci'
      OraclePeclRecipe.oracle_sdk?
    else
      true
    end
  end

  def files_hashs
    native_module_hashes = @native_modules.map do |recipe|
      recipe.send(:files_hashs)
    end.flatten

    extension_hashes = @extensions.map do |recipe|
      recipe.send(:files_hashs) if should_cook?(recipe)
    end.flatten.compact

    extension_hashes + native_module_hashes
  end

  def php_recipe
    php_recipe_options = {}

    hiredis_recipe = @native_modules.detect { |r| r.name == 'hiredis' }
    libmemcached_recipe = @native_modules.detect { |r| r.name == 'libmemcached' }
    ioncube_recipe = @extensions.detect { |r| r.name == 'ioncube' }

    php_recipe_options[:hiredis_path] = hiredis_recipe.path unless hiredis_recipe.nil?
    php_recipe_options[:libmemcached_path] = libmemcached_recipe.path unless libmemcached_recipe.nil?
    php_recipe_options[:ioncube_path] = ioncube_recipe.path unless ioncube_recipe.nil?

    php_recipe_options.merge(DetermineChecksum.new(@options).to_h)

    @php_recipe ||= PhpRecipe.new(@name, @version, php_recipe_options)
  end

  def run(command)
    output = `#{command}`
    if $CHILD_STATUS.success?
      true
    else
      $stdout.puts 'ERROR, output was:'
      $stdout.puts output
      false
    end
  end
end
