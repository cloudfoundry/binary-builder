require 'spec_helper'
require 'timecop'

module BinaryBuilder
  describe Builder do
    subject(:builder) { Builder.new(options) }
    let(:node_architect) { double(:node_architect) }
    let(:options) do
      {
        binary_name: 'node',
        binary_version: 'v0.12.2',
        checksum_value: 'checksum'
      }
    end
    let (:foundation_dir) { 'tmp_dir' }

    before do
      allow(NodeArchitect).to receive(:new)
      allow(Dir).to receive(:mktmpdir).and_return('tmp_dir')
    end

    describe '#new' do
      context 'for a node binary' do
        it 'sets binary_name, binary_version, and checksum values' do
          expect(builder.binary_name).to eq('node')
          expect(builder.binary_version).to eq('v0.12.2')
          expect(builder.checksum_value).to eq('checksum')
        end

        it 'creates a node architect' do
          expect(NodeArchitect).to receive(:new).with({
            binary_version: 'v0.12.2',
            checksum_value: 'checksum'
          }).and_return(node_architect)
          builder
        end
      end
    end

    describe '#set_foundation' do
      let(:blueprint) { double(:blueprint) }

      before do
        allow(NodeArchitect).to receive(:new).and_return(node_architect)
        allow(FileUtils).to receive(:chmod)
      end

      it "writes the architect's blueprint to a temporary executable" do
        expect(node_architect).to receive(:blueprint).and_return(blueprint)
        expect(FileUtils).to receive(:mkdir).with(File.join(foundation_dir, 'installation'))

        blueprint_path = File.join(foundation_dir, 'blueprint.sh')
        expect(File).to receive(:write).with(blueprint_path, blueprint)
        expect(FileUtils).to receive(:chmod).with('+x', blueprint_path)
        builder.set_foundation
      end
    end

    describe '#install' do
      before do
        allow(FileUtils).to receive(:mkdir)
      end

      it 'exercises the blueprint script' do
        blueprint_path = File.join(foundation_dir, 'blueprint.sh')
        expect(Dir).to receive(:chdir).with('tmp_dir').and_yield
        expect(builder).to receive(:system).with("#{blueprint_path} tmp_dir/installation").and_return(true)
        builder.install
      end
    end

    describe '#tar_installed_binary' do

      before do
        allow(builder).to receive(:system).and_return(true)
      end

      it 'tars the remaining files from their directory' do
        expect(builder).to receive(:system).with("ls -A tmp_dir/installation | xargs tar czf node-v0.12.2-linux-x64.tar.gz -C tmp_dir/installation")
        builder.tar_installed_binary
      end

      context 'when the binary name is php' do
        let(:options) do
          {
            binary_name: 'php',
            binary_version: '5.6.11'
          }
        end

        it 'tars up the directory to a file with a timestamp in the name' do
          Timecop.freeze(Time.utc(2012, 12, 12)) do
            expect(builder).to receive(:system).with("ls -A tmp_dir/installation | xargs tar czf php-5.6.11-linux-x64-1355270400.tgz -C tmp_dir/installation")
            builder.tar_installed_binary
          end
        end
      end
    end

    describe '::build' do
      let(:builder) { double(:builder) }

      it 'sets a foundation, installs the binary, and tars the installed binary' do
        allow(Builder).to receive(:new).with(options).and_return(builder)

        expect(builder).to receive(:set_foundation)
        expect(builder).to receive(:install)
        expect(builder).to receive(:tar_installed_binary)
        Builder.build(options)
      end
    end
  end
end
