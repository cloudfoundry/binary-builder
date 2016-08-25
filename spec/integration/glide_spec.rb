# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when glide is specified' do
    before(:all) do
      run_binary_builder('glide', 'v0.11.0', '--sha256=7a7023aff20ba695706a262b8c07840ee28b939ea6358efbb69ab77da04f0052')
      @binary_tarball_location = File.join(Dir.pwd, 'glide-v0.11.0-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      glide_version_cmd = "./spec/assets/binary-exerciser.sh glide-v0.11.0-linux-x64.tgz ./bin/glide -v"
      output, status = run(glide_version_cmd)

      expect(status).to be_success
      expect(output).to include('glide version 0.11.0')
    end

    it 'includes the license in the tar file.' do
      expect(tar_contains_file('bin/LICENSE')).to eq true
    end
  end
end

