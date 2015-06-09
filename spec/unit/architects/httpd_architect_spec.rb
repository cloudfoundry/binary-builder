require 'spec_helper'

module BinaryBuilder
  describe HttpdArchitect do
    subject(:architect) { HttpdArchitect.new(binary_version: '2.4.12') }

    describe '#blueprint' do
      it 'adds the binary version' do
        expect(architect.blueprint).to include '2.4.12'
      end
    end
  end
end
