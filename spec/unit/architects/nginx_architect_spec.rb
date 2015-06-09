require 'spec_helper'

module BinaryBuilder
  describe NginxArchitect do
    subject(:architect) { NginxArchitect.new(binary_version: '1.7.10') }

    describe '#new' do
      it 'sets a binary version' do
        expect(architect.binary_version).to eq('1.7.10')
      end
    end

    describe 'blueprint' do
      it 'adds the binary version' do
        expect(architect.blueprint).to include '1.7.10'
      end
    end
  end
end
