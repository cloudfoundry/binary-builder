require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when php is specified' do
    before(:all) do
      run_binary_builder('php', '5.6.14', '--md5=ae625e0cfcfdacea3e7a70a075e47155')
      @binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'php-5.6.14-linux-x64-*.tgz')).first
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      php_version_cmd = %{./spec/assets/php-exerciser.sh 5.6.14 #{File.basename(@binary_tarball_location)} ./php/bin/php -r 'echo phpversion();'}

      output, status = run(php_version_cmd)

      expect(status).to be_success
      expect(output).to include('5.6.14')
    end

    def tar_contains_file(filename)
      system("tar -tf #{@binary_tarball_location} #{filename} >/dev/null 2>&1")
    end

    it 'copies in *.so files for some of the compiled extensions' do
      expect(tar_contains_file('php/lib/librabbitmq.so.1')).to eq true
      expect(tar_contains_file('php/lib/libhiredis.so.0.10')).to eq true
      expect(tar_contains_file('php/lib/libc-client.so.2007e')).to eq true
      expect(tar_contains_file('php/lib/libmcrypt.so.4')).to eq true
      expect(tar_contains_file('php/lib/libaspell.so.15')).to eq true
      expect(tar_contains_file('php/lib/libpspell.so.15')).to eq true
      expect(tar_contains_file('php/lib/libmemcached.so.10')).to eq true
    end
  end
end

