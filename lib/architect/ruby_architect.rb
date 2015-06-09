module BinaryBuilder
  class RubyArchitect < Architect
    def blueprint
      RubyTemplate.new(binding).result
    end

    def patchless_version
      @patchless_version = binary_version.match(/\D*(\d+\.\d+\.\d+)/)[1]
    end
  end

  class RubyTemplate < Template
    def template_path
      File.expand_path('../../../templates/ruby_blueprint.sh.erb', __FILE__)
    end
  end
end
