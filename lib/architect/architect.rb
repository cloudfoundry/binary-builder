module BinaryBuilder
  class Architect
    attr_reader :binary_version

    def initialize(options)
      @binary_version = options[:binary_version]
    end

    def minor_version
      @minor_version ||= binary_version.match(/\D*(\d+\.\d+)/)[1]
    end
  end

  protected
  class Template
    def initialize(architect_binding)
      @architect_binding = architect_binding
      @erb = ERB.new(read_template)
    end

    attr_reader :binary_version

    def result
      @erb.result(@architect_binding)
    end

    def template_path
      raise 'You must override the #template_path method!'
    end

    private
    def read_template
      @read_template ||= File.open(template_path).read
    end
  end
end
