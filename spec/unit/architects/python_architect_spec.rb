require 'spec_helper'

module BinaryBuilder
  describe PythonArchitect do
    subject(:architect) { PythonArchitect.new(binary_version: '3.4.3') }

    describe '#blueprint' do
      it 'adds the binary version' do
        expect(architect.blueprint).to include '3.4.3'
      end
    end
  end
end
