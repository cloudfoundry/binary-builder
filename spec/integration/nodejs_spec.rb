require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when node is specified' do
    before do
      run_binary_builder('node', '0.12.2', '--sha256=ac7e78ade93e633e7ed628532bb8e650caba0c9c33af33581957f3382e2a772d')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'node-0.12.2-linux-x64.tar.gz')
      expect(File).to exist(binary_tarball_location)

      node_version_cmd = %q{./spec/assets/binary-exerciser.sh node-0.12.2-linux-x64.tar.gz node-v0.12.2-linux-x64/bin/node -e 'console.log(process.version)'}
      output, status = run(node_version_cmd)

      expect(status).to be_success
      expect(output).to include('v0.12.2')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
