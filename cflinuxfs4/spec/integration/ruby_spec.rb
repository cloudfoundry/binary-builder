# frozen_string_literal: true

require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when ruby is specified' do
    before(:all) do
      run_binary_builder('ruby', '2.6.5', '--sha256=66976b716ecc1fd34f9b7c3c2b07bbd37631815377a2e3e85a5b194cfdcbed7d')
      @binary_tarball_location = File.join(Dir.pwd, 'ruby-2.6.5-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      ruby_version_cmd = "./spec/assets/binary-exerciser.sh ruby-2.6.5-linux-x64.tgz ./bin/ruby -e 'puts RUBY_VERSION'"
      output, status = run(ruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.6.5')

      libgmp_cmd = './spec/assets/binary-exerciser.sh ruby-2.6.5-linux-x64.tgz grep LIBS= lib/pkgconfig/ruby-2.6.pc'
      output, status = run(libgmp_cmd)

      expect(status).to be_success
      expect(output).to include('LIBS=')
      expect(output).not_to include('lgmp')
    end
  end
end
