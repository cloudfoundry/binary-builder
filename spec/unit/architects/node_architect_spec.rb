require 'spec_helper'

module BinaryBuilder
  describe NodeArchitect do
    subject(:architect) { NodeArchitect.new(binary_version: 'v0.12.2') }

    describe 'blueprint' do
      let(:template_file) { double(read: 'GIT_TAG') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the node_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../../templates/node_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the git tag value' do
        expect(architect.blueprint).to include 'v0.12.2'
      end
    end
  end
end
