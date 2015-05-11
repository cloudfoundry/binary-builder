require 'digest'
require 'fileutils'
require 'spec_helper'

describe 'building a binary' do
  context 'when node is specified' do
    it 'builds the specified binary, tars it, and places it in your current working directory' do
      run_binary_builder('node', 'v0.12.2', 'cloudfoundry/cflinuxfs2')

      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-cloudfoundry_cflinuxfs2.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_command = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      script_output = `#{docker_command}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('v0.12.2')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
