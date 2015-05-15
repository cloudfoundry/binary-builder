module BinaryBuilder
  class JRubyArchitect < Architect
    JRUBY_TEMPLATE_PATH = File.expand_path('../../../templates/jruby_blueprint', __FILE__)

    attr_reader :jruby_version, :ruby_version

    def initialize(options)
      match_data = options[:binary_version].match(/ruby-(.*)\.\d*-jruby-(.*)/)
      @ruby_version = match_data[1]
      @jruby_version = match_data[2]
    end

    def blueprint
      content = read_file(JRUBY_TEMPLATE_PATH)
      content
        .gsub('GIT_TAG', jruby_version)
        .gsub('RUBY_VERSION', ruby_version)
    end
  end
end
