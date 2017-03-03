# encoding: utf-8
require 'spec_helper'
require 'fileutils'
require 'tmpdir'

describe 'building a binary', :integration do
  context 'when hwc is specified' do

    before(:all) do
      run_binary_builder('hwc', '1.0.1', '--sha256=43839c40ffc0833192c2dac3fcc02ae7015ccab1df7717d2fc1c6c684368d81f')
      @binary_zip_location = File.join(Dir.pwd, 'hwc-1.0.1-windows-amd64.zip')
      @unzip_dir = Dir.mktmpdir
    end

    after(:all) do
      FileUtils.rm(@binary_zip_location)
      FileUtils.rm_rf(@unzip_dir)
    end

    it 'builds the specified binary, zips it, and places it in your current working directory' do
      expect(File).to exist(@binary_zip_location)

      zip_file_cmd = "file hwc-1.0.1-windows-amd64.zip"
      output, status = run(zip_file_cmd)

      expect(status).to be_success
      expect(output).to include('Zip archive data')
    end

    it 'builds a windows binary' do
      Dir.chdir(@unzip_dir) do
        FileUtils.cp(@binary_zip_location, Dir.pwd)
        system "unzip hwc-1.0.1-windows-amd64.zip"
        file_output = `file hwc.exe`.strip
        expect(file_output).to eq('hwc.exe: PE32+ executable for MS Windows (console) Mono/.Net assembly')
      end
    end
  end
end

