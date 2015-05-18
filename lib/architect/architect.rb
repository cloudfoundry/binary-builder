module BinaryBuilder
  class Architect
    private

    def read_file(file)
      @contents ||= File.open(file).read
    end
  end
end
