module BinaryBuilder
  class PythonArchitect < Architect
    PYTHON_TEMPLATE_PATH = File.expand_path('../../../templates/python_blueprint', __FILE__)

    def blueprint
      blueprint_string = read_file(PYTHON_TEMPLATE_PATH)
      blueprint_string.gsub('BINARY_VERSION', binary_version)
    end
  end
end
