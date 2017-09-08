# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when dep is specified' do
    before(:all) do
      run_binary_builder('dep', 'v0.3.0', '--sha256=b8a43e8c95fee236ae8f366b2f7411f35908b981195699abdd47340053e6dd7f')
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
