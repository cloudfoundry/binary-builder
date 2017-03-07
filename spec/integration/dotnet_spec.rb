# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when dotnet is specified' do
    before(:all) do
      run_binary_builder('dotnet', 'v1.0.1', '--git-commit-sha=005db40cd17afb1c435bcce66d6c703220e6602c')
      @binary_tarball_location = File.join(Dir.pwd, 'dotnet.1.0.1.linux-amd64.tar.gz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      dotnet_version_cmd = './spec/assets/binary-exerciser.sh dotnet.1.0.1.linux-amd64.tar.gz ./dotnet --version'
      output, status = run(dotnet_version_cmd)

      expect(status).to be_success
      expect(output).to include('1.0.1')
    end

   it 'returns true' do
     expect(true).to be_truthy
   end
  end
end
