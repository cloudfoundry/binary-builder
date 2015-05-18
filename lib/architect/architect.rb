module BinaryBuilder
  class Architect
    attr_reader :binary_version

    def initialize(options)
      @binary_version = options[:binary_version]
    end

    private
    def read_file(file)
      @contents ||= File.open(file).read
    end
  end
end
