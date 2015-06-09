module BinaryBuilder
  class PythonArchitect < Architect

    def blueprint
      PythonTemplate.new(binding).result
    end
  end

  class PythonTemplate < Template
    def template_path
      File.expand_path('../../../templates/python_blueprint.sh.erb', __FILE__)
    end
  end
end
