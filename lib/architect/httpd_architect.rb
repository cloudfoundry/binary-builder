module BinaryBuilder
  class HttpdArchitect < Architect
    BUILD_VERSIONS = {
      apr_util_version: '1.5.4',
      apr_iconv_version: '1.2.1',
      apr_version: '1.5.2'
    }

    def blueprint
      HttpdTemplate.new(binding).result
    end
  end

  class HttpdTemplate < Template
    def template_path
      File.expand_path('../../../templates/httpd_blueprint.sh.erb', __FILE__)
    end
  end
end
