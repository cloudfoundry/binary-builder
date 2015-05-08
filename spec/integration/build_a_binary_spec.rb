require 'digest'
require 'fileutils'
require 'spec_helper'

describe 'building a binary' do
  context 'when node is specified' do
    it 'builds the specified binary, tars it, and places it in your current working directory' do
      run_binary_builder('node', 'v0.12.2', 'cloudfoundry/cflinuxfs2')

      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-cloudfoundry_cflinuxfs2.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)
      expect(Digest::MD5.file(binary_tarball_location).hexdigest).to eq('3dd3819cd7700bb63a740e4f372c545c')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
