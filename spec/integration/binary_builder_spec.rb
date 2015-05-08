require 'open3'
require 'digest'
require 'fileutils'

describe 'binary-builder binary' do
  def run_binary_builder(interpreter, tag, docker_image, flags = '')
    binary_builder_path = File.join(Dir.pwd, 'bin', 'binary-builder')
    Open3.capture2e("#{binary_builder_path} #{interpreter} #{tag} #{flags}")[0]
  end

  context 'without valid arguments' do
    it 'prints out a helpful usage message if no arguments are provided' do
      expect(run_binary_builder('', '', '')).to include('USAGE', 'interpreter', 'git-tag', 'docker-image')
    end
  end

  context 'when node is specified' do
    it 'builds the specified binary, tars it, and places it in your current working directory' do
      run_binary_builder('node', 'v0.12.2', 'cloudfoundry/cflinuxfs2')

      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-cflinuxfs2.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)
      expect(Digest::MD5.file(binary_tarball_location).hexdigest).to eq('7ceff90ab98af7ce42cf704400ec6e64')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
