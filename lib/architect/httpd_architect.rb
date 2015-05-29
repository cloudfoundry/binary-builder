module BinaryBuilder
  class HttpdArchitect < Architect
    HTTPD_TEMPLATE_PATH = File.expand_path('../../../templates/httpd_blueprint', __FILE__)

    def blueprint
      contents = read_file(HTTPD_TEMPLATE_PATH)
      contents
        .gsub('$HTTPD_VERSION', binary_version)
        .gsub('$APR_UTIL_VERSION', '1.5.4')
        .gsub('$APR_ICONV_VERSION', '1.2.1')
        .gsub('$APR_VERSION', '1.5.2')
    end
  end
end
