# frozen_string_literal: true

require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when node allows openssl-use-def-ca-store' do
    before(:all) do
      run_binary_builder('node', '8.8.1', '--sha256=1725bbbe623d6a13ee14522730dfc90eac1c9ebe9a0a8f4c3322a402dd7e75a2')
      @binary_tarball_location = File.join(Dir.pwd, 'node-8.8.1-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      node_version_cmd = "./spec/assets/binary-exerciser.sh node-8.8.1-linux-x64.tgz node-v8.8.1-linux-x64/bin/node -e 'console.log(process.version)'"

      output, status = run(node_version_cmd)

      expect(status).to be_success
      expect(output).to include('v8.8.1')
    end
  end

  context 'when node DOES NOT allow openssl-use-def-ca-store' do
    before(:all) do
      run_binary_builder('node', '4.8.5', '--sha256=23980b1d31c6b0e05eff2102ffa0059a6f7a93e27e5288eb5551b9b003ec0c07')
      @binary_tarball_location = File.join(Dir.pwd, 'node-4.8.5-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      node_version_cmd = "./spec/assets/binary-exerciser.sh node-4.8.5-linux-x64.tgz node-v4.8.5-linux-x64/bin/node -e 'console.log(process.version)'"

      output, status = run(node_version_cmd)

      expect(status).to be_success
      expect(output).to include('v4.8.5')
    end
  end
end
