require 'spec_helper'

module BinaryBuilder
  describe RubyArchitect do
    subject(:architect) { RubyArchitect.new(binary_version: 'v2_0_0_645') }

    describe '#new' do

      it 'sets a binary version' do
        expect(architect.binary_version).to eq('v2_0_0_645')
      end
    end

    describe 'blueprint' do
      let(:template_file) { double(read: 'GIT_TAG RUBY_DIRECTORY') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the ruby_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../templates/ruby_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the git tag and Ruby directory values' do
        expect(architect.blueprint).to include 'v2_0_0_645'
        expect(architect.blueprint).to include 'ruby-2.0.0'
      end
    end
  end
end
