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
      let(:template_string) { double(:template_string) }

      before do
        allow(architect).to receive(:read_file).and_return(template_string)
        allow(template_string).to receive(:gsub!)
      end

      it 'uses the ruby_blueprint template' do
        expect(architect).to receive(:read_file).with(File.expand_path('../../../templates/ruby_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the git tag and Ruby directory values' do
        expect(template_string).to receive(:gsub!).with('GIT_TAG', 'v2_0_0_645')
        expect(template_string).to receive(:gsub!).with('RUBY_DIRECTORY', 'ruby-2.0.0')
        architect.blueprint
      end
    end
  end
end
