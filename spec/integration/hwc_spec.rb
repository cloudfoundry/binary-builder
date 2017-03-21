# encoding: utf-8
require 'spec_helper'
require 'fileutils'
require 'tmpdir'

describe 'building a binary', :integration do
  context 'when hwc is specified' do

    before(:all) do
      run_binary_builder('hwc', '2.0.0', '--sha256=7de05bf5baed5518b4880a982f7da60a8951c4a5950677d907aec8496a0cd952')
      @binary_zip_location = File.join(Dir.pwd, 'hwc-2.0.0-windows-amd64.zip')
      @unzip_dir = Dir.mktmpdir
    end

    after(:all) do
      FileUtils.rm(@binary_zip_location)
      FileUtils.rm_rf(@unzip_dir)
    end

    it 'builds the specified binary, zips it, and places it in your current working directory' do
      expect(File).to exist(@binary_zip_location)

      zip_file_cmd = "file hwc-2.0.0-windows-amd64.zip"
      output, status = run(zip_file_cmd)

      expect(status).to be_success
      expect(output).to include('Zip archive data')
    end

    it 'builds a windows binary' do
      Dir.chdir(@unzip_dir) do
        FileUtils.cp(@binary_zip_location, Dir.pwd)
        system "unzip hwc-2.0.0-windows-amd64.zip"
        file_output = `file hwc.exe`
        expect(file_output).to include('hwc.exe: PE32+ executable')
        expect(file_output).to include('for MS Windows')
      end
    end
  end
end

