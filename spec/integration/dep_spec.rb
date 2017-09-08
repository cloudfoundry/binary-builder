# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when dep is specified' do
    before(:all) do
      run_binary_builder('dep', 'v0.3.0', '--sha256=7d816ffb14f57c4b01352676998a8cda9e4fb24eaec92bd79526e1045c5a0c83')
      @binary_tarball_location = File.join(Dir.pwd, 'dep-v0.3.0-linux-amd64.tar.gz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      dep_version_cmd = './spec/assets/binary-exerciser.sh dep-v0.3.0-linux-amd64.tar.gz ./bin/dep version'
      output, status = run(dep_version_cmd)

      expect(status).to be_success
      expect(output).to include('v0.3.0')
    end

    it 'includes the license in the tar file.' do
      expect(tar_contains_file('go/LICENSE')).to eq true
    end
  end
end
