# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when python is specified' do
    before do
      run_binary_builder('python', '3.4.3', '--md5=4281ff86778db65892c05151d5de738d')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'python-3.4.3-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      python_version_cmd = "env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh python-3.4.3-linux-x64.tgz ./bin/python -c 'import sys;print(sys.version[:5])'"
      output, status = run(python_version_cmd)

      expect(status).to be_success
      expect(output).to include('3.4.3')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
