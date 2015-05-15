module BinaryBuilder
  class PythonArchitect < Architect
    PYTHON_TEMPLATE_PATH = File.expand_path('../../../templates/python_blueprint', __FILE__)

    attr_reader :binary_version

    def initialize(options)
      @binary_version = options[:binary_version]
    end

    def blueprint
      blueprint_string = read_file(PYTHON_TEMPLATE_PATH)
      blueprint_string.gsub('BINARY_VERSION', binary_version)
    end

    private
    def read_file(file)
      File.open(file).read
    end
  end
end
