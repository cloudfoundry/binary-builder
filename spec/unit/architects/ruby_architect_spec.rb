require 'spec_helper'

module BinaryBuilder
  describe RubyArchitect do
    subject(:architect) { RubyArchitect.new(binary_version: '2.0.0') }

    describe '#blueprint' do
      it 'adds the binary version' do
        expect(architect.blueprint).to include '2.0.0'
      end
    end
  end
end
