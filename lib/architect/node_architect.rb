module BinaryBuilder
  class NodeArchitect < Architect
    def blueprint
      NodeTemplate.new(binding).result
    end
  end

  class NodeTemplate < Template
    def template_path
      File.expand_path('../../../templates/node_blueprint.sh.erb', __FILE__)
    end
  end
end
