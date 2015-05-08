module BinaryBuilder
  class NodeArchitect < Architect
    NODE_TEMPLATE_PATH = File.expand_path('../../../templates/node_blueprint', __FILE__)

    attr_reader :git_tag

    def initialize(git_tag:)
      @git_tag = git_tag
    end

    def blueprint
      blueprint_string = read_file(NODE_TEMPLATE_PATH)
      blueprint_string.gsub('GIT_TAG', git_tag)
    end

    private
    def read_file(file)
      File.open(file).read
    end
  end
end
