require 'spec_helper'
require 'builder'
require 'architect/architects'

module BinaryBuilder
  describe Builder do
    subject(:builder) { Builder.new(options) }
    let(:node_architect) { double(:node_architect) }
    let(:options) do
      {
        binary_name: 'node',
        git_tag: 'v0.12.2',
        docker_image: 'cloudfoundry/cflinuxfs2'
      }
    end

    before do
      allow(NodeArchitect).to receive(:new)
    end

    describe '#new' do
      context 'for a node binary' do
        it 'sets binary_name, git_tag, and docker_image values' do
          expect(builder.binary_name).to eq('node')
          expect(builder.git_tag).to eq('v0.12.2')
          expect(builder.docker_image).to eq('cloudfoundry/cflinuxfs2')
        end

        it 'creates a node architect' do
          expect(NodeArchitect).to receive(:new).with({git_tag: 'v0.12.2'}).and_return(node_architect)
          builder
        end
      end
    end

    describe '#set_foundation' do
      let(:blueprint) { double(:blueprint) }

      before do
        allow(NodeArchitect).to receive(:new).and_return(node_architect)
        allow(FileUtils).to receive(:rm_rf)
      end

      it "writes the architect's blueprint to a temporary file within $HOME" do
        foundation_path = File.join(ENV['HOME'], '.binary-builder', 'node-v0.12.2-foundation')

        expect(node_architect).to receive(:blueprint).and_return(blueprint)
        expect(FileUtils).to receive(:mkdir_p).with(foundation_path)
        expect_any_instance_of(File).to receive(:write).with(blueprint)
        builder.set_foundation
      end
    end

    describe '#install' do
      it 'exercises the blueprint script' do
        foundation_path = File.join(ENV['HOME'], '.binary-builder', 'node-v0.12.2-foundation')
        blueprint_path = File.join(foundation_path, 'blueprint.sh')
        expect(builder).to receive(:run!).with(blueprint_path)
        builder.install
      end
    end

    describe '#tar_installed_binary' do
      let (:foundation_path) { File.join(ENV['HOME'], '.binary-builder', 'node-v0.12.2-foundation') }

      before do
        allow(FileUtils).to receive(:rm)
        allow(FileUtils).to receive(:rm_rf)
        allow(builder).to receive(:run!)
      end

      it 'removes the blueprint' do
        expect(FileUtils).to receive(:rm).with(File.join(foundation_path, 'blueprint.sh'))
        builder.tar_installed_binary
      end

      it 'tars the remaining files from their directory' do
        foundation_path = File.join(ENV['HOME'], '.binary-builder', 'node-v0.12.2-foundation')
        expect(builder).to receive(:run!).with("tar czf node-v0.12.2-linux-x64.tgz -C #{foundation_path} .")
        builder.tar_installed_binary
      end

      it 'removes all evidence (big files are big)' do
        expect(FileUtils).to receive(:rm_rf).with(foundation_path)
        builder.tar_installed_binary
      end
    end

    describe '#build' do
      let(:builder) { double(:builder) }

      it 'sets a foundation, installs via docker, and tars the installed binary' do
        allow(Builder).to receive(:new).with(options).and_return(builder)

        expect(builder).to receive(:set_foundation)
        expect(builder).to receive(:install)
        expect(builder).to receive(:tar_installed_binary)
        Builder.build(options)
      end
    end
  end
end
