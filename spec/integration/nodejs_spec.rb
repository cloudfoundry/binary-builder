require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when node is specified', binary: 'node' do
    before do
      run_binary_builder('node', '0.12.2', 'dontcare')
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
