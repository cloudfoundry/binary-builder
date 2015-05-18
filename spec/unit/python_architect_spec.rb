require 'spec_helper'

module BinaryBuilder
  describe PythonArchitect do
    subject(:architect) { PythonArchitect.new(binary_version: '3.4.3') }

    describe '#new' do

      it 'sets a binary version' do
        expect(architect.binary_version).to eq('3.4.3')
      end
    end

    describe 'blueprint' do
      let(:template_file) { double(read: 'BINARY_VERSION') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the python_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../templates/python_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the binary version value' do
        expect(architect.blueprint).to include '3.4.3'
      end
    end
  end
end
