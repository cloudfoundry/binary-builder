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
      let(:template_file) { double(read: '$NGINX_VERSION') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the httpd_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../../templates/nginx_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the binary version' do
        expect(architect.blueprint).to include '1.7.10'
      end
    end
  end
end
