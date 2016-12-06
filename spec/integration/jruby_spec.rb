# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when jruby is specified' do
    before(:all) do
      output = run_binary_builder('jruby', '9.1.6.0_ruby-2.3.1', '--sha256=b648dc46bec6c91ce48c0d7da09972f4f63dd7b7579d374e94fde17c219b05ae')
      @binary_tarball_location = File.join(Dir.pwd, 'jruby-9.1.6.0_ruby-2.3.1-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      jruby_version_cmd = './spec/assets/jruby-exerciser.sh'
      output, status = run(jruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('java 2.3.1')
    end
  end
end
