# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when go is specified' do
    before(:all) do
      run_binary_builder('go', '1.6.3', '--sha256=6326aeed5f86cf18f16d6dc831405614f855e2d416a91fd3fdc334f772345b00')
      @binary_tarball_location = File.join(Dir.pwd, 'go1.6.3.linux-amd64.tar.gz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      go_version_cmd = './spec/assets/binary-exerciser.sh go1.6.3.linux-amd64.tar.gz GOROOT=/tmp/binary-exerciser/go ./go/bin/go version'
      output, status = run(go_version_cmd)

      expect(status).to be_success
      expect(output).to include('go1.6.3')
    end

    it 'includes the license in the tar file.' do
      expect(tar_contains_file('go/LICENSE')).to eq true
    end
  end
end
