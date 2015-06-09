require 'spec_helper'

module BinaryBuilder
  describe NodeArchitect do
    subject(:architect) { NodeArchitect.new(binary_version: 'v0.12.2') }

    describe '#blueprint' do
      it 'adds the binary_version value' do
        expect(architect.blueprint).to include 'v0.12.2'
      end
    end
  end
end
