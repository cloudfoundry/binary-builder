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
      let(:template_string) { double(:template_string) }

      before do
        allow(architect).to receive(:read_file).and_return(template_string)
        allow(template_string).to receive(:gsub)
      end

      it 'uses the python_blueprint template' do
        expect(architect).to receive(:read_file).with(File.expand_path('../../../templates/python_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the binary version value' do
        expect(template_string).to receive(:gsub).with('BINARY_VERSION', '3.4.3')
        architect.blueprint
      end
    end
  end
end
