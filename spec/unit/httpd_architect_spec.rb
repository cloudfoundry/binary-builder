require 'spec_helper'

module BinaryBuilder
  describe HttpdArchitect do
    subject(:architect) { HttpdArchitect.new(binary_version: '2.4.12') }

    describe '#new' do

      it 'sets a binary version' do
        expect(architect.binary_version).to eq('2.4.12')
      end
    end

    describe 'blueprint' do
      let(:template_file) { double(read: 'BINARY_VERSION') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the httpd_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../templates/httpd_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the binary version' do
        expect(architect.blueprint).to include '2.4.12'
      end
    end
  end
end
