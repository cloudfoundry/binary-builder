# encoding: utf-8
require 'spec_helper'
require 'fileutils'
require 'tmpdir'

describe 'building a binary', :integration do
  context 'when hwc is specified' do

    before(:all) do
      run_binary_builder('hwc', '20.0.0', '--sha256=643fd1225881bd6206eec205ba818cf60be00bd3a1029c86b0e5bf74a3a978ab')
      @binary_zip_location = File.join(Dir.pwd, 'hwc-20.0.0-windows-x86-64.zip')
      @unzip_dir = Dir.mktmpdir
    end

    after(:all) do
      FileUtils.rm(@binary_zip_location)
      FileUtils.rm_rf(@unzip_dir)
    end

    it 'builds the specified binary, zips it, and places it in your current working directory' do
      expect(File).to exist(@binary_zip_location)

      zip_file_cmd = "file hwc-20.0.0-windows-x86-64.zip"
      output, status = run(zip_file_cmd)

      expect(status).to be_success
      expect(output).to include('Zip archive data')
    end

    it 'builds a windows binary' do
      Dir.chdir(@unzip_dir) do
        FileUtils.cp(@binary_zip_location, Dir.pwd)
        system "unzip hwc-20.0.0-windows-x86-64.zip"
        file_output = `file hwc.exe`
        expect(file_output).to include('hwc.exe: PE32+ executable')
        expect(file_output).to include('for MS Windows')

        file_output = `file hwc_x86.exe`
        expect(file_output).to include('hwc_x86.exe: PE32 executable')
        expect(file_output).to include('for MS Windows')
      end
    end
  end
end

