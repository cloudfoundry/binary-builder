module BinaryBuilder
  class Architect
    attr_reader :binary_version, :checksum_value

    def initialize(options)
      @binary_version = options[:binary_version]
      @checksum_value = options[:checksum_value]
    end

    def minor_version
      @minor_version ||= binary_version.match(/\D*(\d+\.\d+)/)[1]
    end
  end

  protected
  class Template
    def initialize(architect_binding)
      @architect_binding = architect_binding
      base_template = File.expand_path('../../../templates/base.sh.erb', __FILE__)
      @base_erb = ERB.new(read_template(base_template))
      @erb = ERB.new(read_template(template_path))
    end

    attr_reader :binary_version

    def result
      @base_erb.result(@architect_binding) + @erb.result(@architect_binding)
    end

    def template_path
      raise 'You must override the #template_path method!'
    end

    private
    def read_template(file_path)
      File.open(file_path).read
    end
  end
end
