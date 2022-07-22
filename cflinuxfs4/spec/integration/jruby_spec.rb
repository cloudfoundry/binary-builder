# frozen_string_literal: true

require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when jruby is specified' do
    before(:all) do
      output = run_binary_builder('jruby', '9.2.8.0-ruby-2.5', '--sha256=287ae0e946c2d969613465c738cc3b09098f9f25805893ab707dce19a7b98c43')
      @binary_tarball_location = File.join(Dir.pwd, 'jruby-9.2.8.0-ruby-2.5-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      jruby_version_cmd = './spec/assets/jruby-exerciser.sh'
      output, status = run(jruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('java 2.5.3')
    end
  end
end
