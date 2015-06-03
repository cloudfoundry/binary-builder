require 'spec_helper'

module BinaryBuilder
  describe Architect do
    subject(:architect) { Architect.new(binary_version: 'v0.12.2') }

    describe '#new' do

      it 'sets a binary version' do
        expect(architect.binary_version).to eq('v0.12.2')
      end
    end
  end
end

