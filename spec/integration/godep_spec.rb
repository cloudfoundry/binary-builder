# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when godep is specified' do
    before(:all) do
      run_binary_builder('godep', 'v14', '--sha256=0f212bcf903d5b01db0e93a4218b79f228c6f080d5a409dd4e2ec5edfbc2aad5')
      @binary_tarball_location = File.join(Dir.pwd, 'godep-v14-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      godep_version_cmd = './spec/assets/binary-exerciser.sh godep-v14-linux-x64.tgz ./bin/godep version'
      output, status = run(godep_version_cmd)

      expect(status).to be_success
      expect(output).to include('v14')
    end

    it 'includes the license in the tar file.' do
      expect(tar_contains_file('bin/License')).to eq true
    end
  end
end
