require 'spec_helper'

module BinaryBuilder
  describe JRubyArchitect do
    subject(:architect) { JRubyArchitect.new(binary_version: '1.7.19_ruby-2.0.0') }

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

      context "maven version" do
        before do
          allow(architect).to receive(:maven_version).and_return('1.2.3')
          allow(architect).to receive(:maven_md5).and_return('MyFakeMD5-789789789')
        end

        it 'specifies downloading the correct maven version' do
          expect(architect.blueprint).to include "apache-maven-1.2.3-bin.tar.gz"
        end

        it 'specifies checking the downloaded maven tarball with the appropriate checksum' do
          expect(architect.blueprint).to include '-bin.tar.gz) != *"MyFakeMD5-789789789"* ]]; then'
        end
      end
    end
  end
end
