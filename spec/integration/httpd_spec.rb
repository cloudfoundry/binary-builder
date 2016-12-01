# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when httpd is specified' do
    before(:all) do
      run_binary_builder('httpd', '2.4.17', '--md5=cf4dfee11132cde836022f196611a8b7')
      @binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'httpd-2.4.17-linux-x64*.tgz')).first
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      httpd_version_cmd = %(env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh #{File.basename(@binary_tarball_location)} ./httpd/bin/httpd -v)

      output, status = run(httpd_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.4.17')
    end

    it 'copies in *.so files for some of the compiled extensions' do
      expect(tar_contains_file('httpd/lib/libapr-1.so.0')).to eq true
      expect(tar_contains_file('httpd/lib/libaprutil-1.so.0')).to eq true
      expect(tar_contains_file('httpd/lib/libapriconv-1.so.0')).to eq true
      expect(tar_contains_file('httpd/lib/apr-util-1/apr_ldap.so')).to eq true
      expect(tar_contains_file('httpd/lib/iconv/utf-8.so')).to eq true
    end
  end
end
