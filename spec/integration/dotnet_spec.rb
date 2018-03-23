# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when dotnet is specified' do
    before(:all) do
      @dotnet_version = '2.1.102'
      @dotnet_sha = '8d409357dbac391dad0270f34058856e562b1b8e'

      run_binary_builder('dotnet', "v#{@dotnet_version}", "--git-commit-sha=#{@dotnet_sha}")
      @binary_tarball_location = File.join(Dir.pwd, "dotnet.#{@dotnet_version}.linux-amd64.tar.gz")
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      dotnet_version_cmd = "./spec/assets/binary-exerciser.sh dotnet.#{@dotnet_version}.linux-amd64.tar.gz ./dotnet --version"
      output, status = run(dotnet_version_cmd)

      expect(status).to be_success
      expect(output).to include(@dotnet_version)
    end
  end
end
