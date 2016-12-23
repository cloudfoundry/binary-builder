# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when dotnet is specified' do
    before(:all) do
      run_binary_builder('dotnet', 'v1.0.0-preview4-004233', '--git-commit-sha=8cec61c6f74cc9648c372388615613a6be156b0c')
      @binary_tarball_location = File.join(Dir.pwd, 'dotnet.1.0.0-preview4-004233.linux-amd64.tar.gz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      dotnet_version_cmd = './spec/assets/binary-exerciser.sh dotnet.1.0.0-preview4-004233.linux-amd64.tar.gz ./dotnet --version'
      output, status = run(dotnet_version_cmd)

      expect(status).to be_success
      expect(output).to include('1.0.0-preview4-004233')
    end
  end
end
