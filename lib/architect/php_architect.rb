require 'erb'

module BinaryBuilder
  class PHPArchitect < Architect

    EXTERNAL_LIBRARIES = {
      '5.4' => {
        ZTS_VERSION:          "20100525",
        RABBITMQ_C_VERSION:   "0.5.2",
        LIBMEMCACHED_VERSION: "1.0.18",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      },
      '5.5' => {
        ZTS_VERSION:          "20121212",
        RABBITMQ_C_VERSION:   "0.5.2",
        LIBMEMCACHED_VERSION: "1.0.18",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      },
      '5.6' => {
        ZTS_VERSION:          "20131226",
        RABBITMQ_C_VERSION:   "0.5.2",
        LIBMEMCACHED_VERSION: "1.0.18",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4"
      }
    }

    class PHPExtension < Struct.new(:name, :version); end

    AMQPExtension = PHPExtension.new('amqp', '1.4.0')
    IntlExtension = PHPExtension.new('intl', '3.0.0')
    MemcachedExtension = PHPExtension.new('memcached', '2.2.0')
    PHPIRedisExtension = PHPExtension.new('phpiredis', 'trunk')

    EXTERNAL_EXTENSIONS = {
      '5.4' => [
        AMQPExtension, IntlExtension, MemcachedExtension, PHPIRedisExtension,
        PHPExtension.new('APC', '3.1.9'),
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
        PHPExtension.new('zendopcache', '7.0.4'),
        PHPExtension.new('zookeeper', '0.2.2')
      ],
      '5.5' => [
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
      PHPTemplate.new(binding).result
    end

    def external_extensions
      EXTERNAL_EXTENSIONS[minor_version]
    end

    def external_libraries
      EXTERNAL_LIBRARIES[minor_version]
    end
  end

  protected
  class PHPTemplate < Template
    def template_path
      File.expand_path('../../../templates/php_blueprint.sh.erb', __FILE__)
    end
  end
end
