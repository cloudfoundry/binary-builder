module BinaryBuilder
  class NginxArchitect < Architect
    def blueprint
      NginxTemplate.new(binding).result
    end
  end

  class NginxTemplate < Template
    def template_path
      File.expand_path('../../../templates/nginx_blueprint.sh.erb', __FILE__)
    end
  end
end
