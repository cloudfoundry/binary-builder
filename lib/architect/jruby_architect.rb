module BinaryBuilder
  class JRubyArchitect < Architect
    attr_reader :jruby_version, :ruby_version

    def initialize(options)
      super
      match_data = options[:binary_version].match(/(.*)_ruby-(\d+\.\d).*/)
      @jruby_version, @ruby_version = match_data[1], match_data[2]
    end

    def maven_version
      '3.3.3'
    end

    def maven_md5
      '794b3b7961200c542a7292682d21ba36'
    end

    def blueprint
      JRubyTemplate.new(binding).result
    end
  end

  class JRubyTemplate < Template
    def template_path
      File.expand_path('../../../templates/jruby_blueprint.sh.erb', __FILE__)
    end
  end
end
