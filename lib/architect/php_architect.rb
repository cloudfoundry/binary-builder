require 'erb'

module BinaryBuilder
  class PHPArchitect < Architect
    PHP_TEMPLATE_PATH = File.expand_path('../../../templates/php_blueprint.sh.erb', __FILE__)

    EXTERNAL_LIBRARIES = {
      '5.4' => {
        ZTS_VERSION:          "20100525",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4",
        LIBMEMCACHED_VERSION: "1.0.18"
      },
      '5.5' => {
        ZTS_VERSION:          "20121212",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4",
        LIBMEMCACHED_VERSION: "1.0.18"
      },
      '5.6' => {
        ZTS_VERSION:          "20131226",
        RABBITMQ_C_VERSION:   "0.5.2",
        HIREDIS_VERSION:      "0.11.0",
        LUA_VERSION:          "5.2.4",
        LIBMEMCACHED_VERSION: "1.0.18"
      }
    }

    EXTERNAL_MODULES = {
      '5.4' => {
        amqp:            "1.4.0",
        APC:             "3.1.9",
        apcu:            "4.0.7",
        igbinary:        "1.2.1",
        imagick:         "3.1.2",
        intl:            "3.0.0",
        ioncube:         "4.7.5",
        lua:             "1.1.0",
        mailparse:       "2.1.6",
        memcache:        "2.2.7",
        memcached:       "2.2.0",
        mongo:           "1.6.5",
        msgpack:         "0.5.5",
        phpiredis:       "trunk",
        phalcon:         "1.3.4",
        protocolbuffers: "0.2.6",
        protobuf:        "trunk",
        redis:           "2.2.5",
        suhosin:         "0.9.37.1",
        sundown:         "0.3.11",
        twig:            "1.18.0",
        xcache:          "3.2.0",
        xdebug:          "2.3.1",
        xhprof:          "trunk",
        yaf:             "2.2.9",
        zendopcache:     "7.0.4",
        zookeeper:       "0.2.2",
        libmemcached:    "1.0.18"
      },
      '5.5' => {
        amqp:            "1.4.0",
        igbinary:        "1.2.1",
        imagick:         "3.1.2",
        intl:            "3.0.0",
        ioncube:         "4.7.5",
        lua:             "1.1.0",
        mailparse:       "2.1.6",
        memcache:        "2.2.7",
        memcached:       "2.2.0",
        mongo:           "1.6.5",
        msgpack:         "0.5.5",
        phpiredis:       "trunk",
        phalcon:         "1.3.4",
        protocolbuffers: "0.2.6",
        protobuf:        "trunk",
        redis:           "2.2.5",
        suhosin:         "0.9.37.1",
        sundown:         "0.3.11",
        twig:            "1.18.0",
        xcache:          "3.2.0",
        xdebug:          "2.3.1",
        xhprof:          "trunk",
        yaf:             "2.2.9"
      },
      '5.6' => {
        amqp:            "1.4.0",
        igbinary:        "1.2.1",
        imagick:         "3.1.2",
        intl:            "3.0.0",
        ioncube:         "4.7.5",
        lua:             "1.1.0",
        mailparse:       "2.1.6",
        memcache:        "2.2.7",
        memcached:       "2.2.0",
        mongo:           "1.6.5",
        msgpack:         "0.5.5",
        phpiredis:       "trunk",
        phalcon:         "1.3.4",
        protocolbuffers: "0.2.6",
        protobuf:        "trunk",
        redis:           "2.2.5",
        suhosin:         "0.9.37.1",
        sundown:         "0.3.11",
        twig:            "1.18.0",
        xcache:          "3.2.0",
        xdebug:          "2.3.1",
        yaf:             "2.3.3"
      }
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

      def external_modules
        EXTERNAL_MODULES[@minor_version]
      end

      def external_module_dependencies
        {
          amqp:      ["$APP_DIR/librmq-$RABBITMQ_C_VERSION/lib/librabbitmq.so.1"],
          intl:      ["libicui18n.so.52", "libicuuc.so.52", "libicudata.so.52", "libicuio.so.52"],
          memcached: ["libmemcached.so.10"],
          phpiredis: ["$APP_DIR/hiredis-$HIREDIS_VERSION/lib/libhiredis.so.0.10"]
        }
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
