require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when jruby is specified' do
    before do
      output, _ = run_binary_builder('jruby', '9.0.0.0_ruby-2.2.0', '--sha256=cef101e4265b65e2c729eba97838546c8e08123d8ee18f0e12fd0dd8d0db16b6')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'jruby-9.0.0.0_ruby-2.2.0-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      jruby_version_cmd = %q{./spec/assets/jruby-exerciser.sh}
      output, status = run(jruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('java 2.2.2')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
