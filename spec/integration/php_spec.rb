require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when php is specified' do
    before do
      run_binary_builder('php', '5.6.9', '--md5=b83de326a3bdb3802266d304ca5ac5e4 --source-yaml')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'php-5.6.9-linux-x64-*.tgz')).first
      expect(File).to exist(binary_tarball_location)

      php_version_cmd = %{./spec/assets/php-exerciser.sh 5.6.9 #{File.basename(binary_tarball_location)} ./php/bin/php -r 'echo phpversion();'}

      output, status = run(php_version_cmd)

      expect(status).to be_success
      expect(output).to include('5.6.9')
      FileUtils.rm(binary_tarball_location)
    end
  end
end

