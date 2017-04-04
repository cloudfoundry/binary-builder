# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when python is specified' do
    before(:all) do
      run_binary_builder('python', '2.7.13', '--md5=17add4bf0ad0ec2f08e0cae6d205c700')
      @binary_tarball_location = File.join(Dir.pwd, 'python-2.7.13-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      python_version_cmd = "env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh python-2.7.13-linux-x64.tgz ./bin/python -c 'import sys;print(sys.version)'"
      output, status = run(python_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.7.13')
    end

    it 'python is built with ucs4 support' do
      expect(File).to exist(@binary_tarball_location)

      python_version_cmd = "env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh python-2.7.13-linux-x64.tgz ./bin/python -c 'import sys;print(sys.maxunicode)'"
      output, status = run(python_version_cmd)

      expect(status).to be_success
      expect(output).to include('1114111')
    end
  end
end
