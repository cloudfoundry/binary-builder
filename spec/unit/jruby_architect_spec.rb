require 'spec_helper'

module BinaryBuilder
  describe JRubyArchitect do
    subject(:architect) { JRubyArchitect.new(binary_version: 'ruby-2.0.0-jruby-1.7.19') }

    describe '#new' do
      it 'sets the correct versions' do
        expect(architect.jruby_version).to eq('1.7.19')
        expect(architect.ruby_version).to eq('2.0')
      end
    end

    describe '#blueprint' do
      let(:template_file) { double(read: 'GIT_TAG RUBY_VERSION') }

      before do
        allow(File).to receive(:open).and_return(template_file)
      end

      it 'uses the jruby_blueprint template' do
        expect(File).to receive(:open).with(File.expand_path('../../../templates/jruby_blueprint', __FILE__))
        architect.blueprint
      end

      it 'adds the git tag value' do
        expect(architect.blueprint).to include '1.7.19'
      end

      it 'adds the default Ruby version' do
        expect(architect.blueprint).to include '2.0'
      end
    end
  end
end
