require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when httpd is specified' do
    before do
      output, _ = run_binary_builder('httpd', '2.4.12', '--md5=b8dc8367a57a8d548a9b4ce16d264a13')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'httpd-2.4.12-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      httpd_version_cmd = %q{env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh httpd-2.4.12-linux-x64.tgz ./httpd/bin/httpd -v}

      output, status = run(httpd_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.4.12')
      FileUtils.rm(binary_tarball_location)
    end
  end
end

