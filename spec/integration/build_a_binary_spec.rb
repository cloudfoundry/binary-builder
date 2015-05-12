require 'digest'
require 'fileutils'
require 'spec_helper'

describe 'building a binary' do
  before do
    run_binary_builder(binary_name, binary_version)
  end

  context 'when node is specified' do
    let(:binary_name) { 'node' }
    let(:binary_version) { 'v0.12.2' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = "node-v0.12.2-linux-x64.tgz node-v0.12.2-linux-x64/bin/node 'console.log(process.version)'"

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('v0.12.2')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when ruby is specified' do
    let(:binary_name) { 'ruby' }
    let(:binary_version) { 'v2_0_0_645' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'ruby-v2_0_0_645-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = "ruby-v2_0_0_645-linux-x64.tgz ./bin/ruby 'puts RUBY_VERSION'"

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('2.0.0')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
