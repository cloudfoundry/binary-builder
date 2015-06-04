require 'spec_helper'

describe 'building a binary', :integration do
  context 'when nginx is specified', binary: 'nginx' do
    before do
      run_binary_builder('nginx', '1.7.10')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'nginx-1.7.10-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      httpd_version_cmd = %q{./spec/assets/binary-exerciser.sh nginx-1.7.10-linux-x64.tgz ./nginx/sbin/nginx -v}
      output, status = run(httpd_version_cmd)

      expect(status).to be_success
      expect(output).to include('1.7.10')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
