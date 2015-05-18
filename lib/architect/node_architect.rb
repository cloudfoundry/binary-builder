module BinaryBuilder
  class NodeArchitect < Architect
    NODE_TEMPLATE_PATH = File.expand_path('../../../templates/node_blueprint', __FILE__)

    attr_reader :binary_version

    def initialize(options)
      @binary_version = options[:binary_version]
    end

    def blueprint
      blueprint_string = read_file(NODE_TEMPLATE_PATH)
      blueprint_string.gsub('GIT_TAG', binary_version)
    end
  end
end
