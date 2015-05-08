require 'spec_helper'

module BinaryBuilder
  describe NodeArchitect do
    subject(:architect) { NodeArchitect.new(git_tag: 'v0.12.2') }

    describe '#new' do

      it 'sets a git tag' do
        expect(architect.git_tag).to eq('v0.12.2')
      end
    end

    describe 'blueprint' do
      let(:template_string) { double(:template_string) }

      before do
        allow(architect).to receive(:read_file).and_return(template_string)
        allow(template_string).to receive(:gsub)
      end

      it 'uses the node_blueprint template' do
        expect(architect).to receive(:read_file).with(File.expand_path('../../../templates/node_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the git tag value' do
        expect(template_string).to receive(:gsub).with('GIT_TAG', 'v0.12.2')
        architect.blueprint
      end
    end
  end
end
