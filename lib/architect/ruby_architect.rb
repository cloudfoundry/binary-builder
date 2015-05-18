module BinaryBuilder
  class RubyArchitect < Architect
    RUBY_TEMPLATE_PATH = File.expand_path('../../../templates/ruby_blueprint', __FILE__)

    attr_reader :binary_version

    def initialize(options)
      @binary_version = options[:binary_version]
    end

    def blueprint
      contents = read_file(RUBY_TEMPLATE_PATH)
      contents
        .gsub('GIT_TAG', binary_version)
        .gsub('RUBY_DIRECTORY', "ruby-#{binary_version[1..-1].split('_')[0..2].join('.')}")
    end
  end
end
