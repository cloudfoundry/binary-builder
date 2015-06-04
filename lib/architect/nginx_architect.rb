module BinaryBuilder
  class NginxArchitect < Architect
    NGINX_TEMPLATE_PATH = File.expand_path('../../../templates/nginx_blueprint', __FILE__)

    def blueprint
      contents = read_file(NGINX_TEMPLATE_PATH)
      contents
        .gsub('$NGINX_VERSION', binary_version)
    end
  end
end
