require 'erb'

module BinaryBuilder
  class PHPArchitect < Architect
    PHP_TEMPLATE_PATH = File.expand_path('../../../templates/php_blueprint.sh.erb', __FILE__)

    EXTERNAL_LIBRARIES = {
      '5.4' => {
        ZTS_VERSION:          "20100525",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      },
      '5.5' => {
        ZTS_VERSION:          "20121212",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      },
      '5.6' => {
        ZTS_VERSION:          "20131226",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      }
    }

    class PHPExtension < Struct.new(:name, :version, :package_php_extension_args)
      def package_php_extension_args
        super || [ name ]
      end
    end

    AMQPExtension = PHPExtension.new('amqp', '1.4.0', ['amqp', "$APP_DIR/librmq-$RABBITMQ_C_VERSION/lib/librabbitmq.so.1"])
    IntlExtension = PHPExtension.new('intl', '3.0.0', ['intl', "libicui18n.so.52", "libicuuc.so.52", "libicudata.so.52", "libicuio.so.52"])
    MemcachedExtension = PHPExtension.new('memcached', '2.2.0', ['memcached', "libmemcached.so.10"])
    PHPIRedisExtension = PHPExtension.new('phpiredis', 'trunk', ['phpiredis', "$APP_DIR/hiredis-$HIREDIS_VERSION/lib/libhiredis.so.0.10"])

    EXTERNAL_EXTENSIONS = {
      '5.4' => [
        AMQPExtension, IntlExtension, MemcachedExtension, PHPIRedisExtension,
        PHPExtension.new('APC', '3.1.9', ['apc']),
        PHPExtension.new('apcu', '4.0.7'),
        PHPExtension.new('igbinary', '1.2.1'),
        PHPExtension.new('imagick', '3.1.2'),
        PHPExtension.new('ioncube', '4.7.5'),
        PHPExtension.new('lua', '1.1.0'),
        PHPExtension.new('mailparse', '2.1.6'),
        PHPExtension.new('memcache', '2.2.7'),
        PHPExtension.new('mongo', '1.6.5'),
        PHPExtension.new('msgpack', '0.5.5'),
        PHPExtension.new('phalcon', '1.3.4'),
        PHPExtension.new('protocolbuffers', '0.2.6'),
        PHPExtension.new('protobuf', 'trunk'),
        PHPExtension.new('redis', '2.2.5'),
        PHPExtension.new('suhosin', '0.9.37.1'),
        PHPExtension.new('sundown', '0.3.11'),
        PHPExtension.new('twig', '1.18.0'),
        PHPExtension.new('xcache', '3.2.0'),
        PHPExtension.new('xdebug', '2.3.1'),
        PHPExtension.new('xhprof', 'trunk'),
        PHPExtension.new('yaf', '2.2.9'),
        PHPExtension.new('zendopcache', '7.0.4', ['opcache']),
        PHPExtension.new('zookeeper', '0.2.2')
      ],
      '5.5' => [
        AMQPExtension, IntlExtension, MemcachedExtension, PHPIRedisExtension,
        PHPExtension.new('APC', '3.1.9', ['apc']),
        PHPExtension.new('igbinary', '1.2.1'),
        PHPExtension.new('imagick', '3.1.2'),
        PHPExtension.new('ioncube', '4.7.5'),
        PHPExtension.new('lua', '1.1.0'),
        PHPExtension.new('mailparse', '2.1.6'),
        PHPExtension.new('memcache', '2.2.7'),
        PHPExtension.new('mongo', '1.6.5'),
        PHPExtension.new('msgpack', '0.5.5'),
        PHPExtension.new('phalcon', '1.3.4'),
        PHPExtension.new('protocolbuffers', '0.2.6'),
        PHPExtension.new('protobuf', 'trunk'),
        PHPExtension.new('redis', '2.2.5'),
        PHPExtension.new('suhosin', '0.9.37.1'),
        PHPExtension.new('sundown', '0.3.11'),
        PHPExtension.new('twig', '1.18.0'),
        PHPExtension.new('xcache', '3.2.0'),
        PHPExtension.new('xdebug', '2.3.1'),
        PHPExtension.new('xhprof', 'trunk'),
        PHPExtension.new('yaf', '2.2.9')
      ],
      '5.6' => [
        AMQPExtension, IntlExtension, MemcachedExtension, PHPIRedisExtension,
        PHPExtension.new('igbinary', '1.2.1'),
        PHPExtension.new('imagick', '3.1.2'),
        PHPExtension.new('ioncube', '4.7.5'),
        PHPExtension.new('lua', '1.1.0'),
        PHPExtension.new('mailparse', '2.1.6'),
        PHPExtension.new('memcache', '2.2.7'),
        PHPExtension.new('mongo', '1.6.5'),
        PHPExtension.new('msgpack', '0.5.5'),
        PHPExtension.new('phalcon', '1.3.4'),
        PHPExtension.new('protocolbuffers', '0.2.6'),
        PHPExtension.new('protobuf', 'trunk'),
        PHPExtension.new('redis', '2.2.5'),
        PHPExtension.new('suhosin', '0.9.37.1'),
        PHPExtension.new('sundown', '0.3.11'),
        PHPExtension.new('twig', '1.18.0'),
        PHPExtension.new('xcache', '3.2.0'),
        PHPExtension.new('xdebug', '2.3.1'),
        PHPExtension.new('yaf', '2.3.3')
      ]
    }

    def blueprint
      contents = read_file(PHP_TEMPLATE_PATH)
      Template.new(
        contents: contents,
        minor_version: minor_version,
        binary_version: binary_version
      ).result
    end

    private
    def minor_version
      binary_version.match(/(\d+\.\d+)/)[1]
    end

    class Template
      def initialize(options)
        @erb = ERB.new(options[:contents])
        @minor_version = options[:minor_version]
        @binary_version = options[:binary_version]
      end

      def external_extensions
        EXTERNAL_EXTENSIONS[@minor_version]
      end

      def external_libraries
        EXTERNAL_LIBRARIES[@minor_version]
      end

      attr_reader :binary_version

      def result
        @erb.result(binding)
      end
    end
  end
end
