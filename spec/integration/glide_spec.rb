# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when glide is specified' do
    before(:all) do
      run_binary_builder('glide', '0.10.2', '--sha256=f0153d88f12fb36419cb616d9922ae95b274ac7c9ed9b043701f187da5834eac')
      @binary_tarball_location = File.join(Dir.pwd, 'glide-0.10.2-linux-x64.tgz')
    end
    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      glide_version_cmd = "./spec/assets/binary-exerciser.sh glide-0.10.2-linux-x64.tgz ./bin/glide -v"
      output, status = run(glide_version_cmd)

      expect(output).to include('glide version 0.10.2')
    end

    it 'includes the license in the tar file.' do
      expect(tar_contains_file('bin/LICENSE')).to eq true
    end
  end
end

