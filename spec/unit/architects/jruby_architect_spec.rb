require 'spec_helper'

module BinaryBuilder
  describe JRubyArchitect do
    subject(:architect) { JRubyArchitect.new(binary_version: '1.7.19+ruby-2.0.0') }

    describe '#new' do
      it 'sets the correct versions' do
        expect(architect.jruby_version).to eq('1.7.19')
        expect(architect.ruby_version).to eq('2.0')
      end
    end

    describe '#blueprint' do
      it 'adds the binary_version value' do
        expect(architect.blueprint).to include '1.7.19'
      end

      it 'adds the default Ruby version' do
        expect(architect.blueprint).to include '2.0'
      end
    end
  end
end
