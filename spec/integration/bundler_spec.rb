# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when bundler is specified' do
    before(:all) do
      run_binary_builder('bundler', '1.11.2', '--sha256=c7aa8ffe0af6e0c75d0dad8dd7749cb8493b834f0ed90830d4843deb61906768')
      @binary_tarball_location = File.join(Dir.pwd, 'bundler-1.11.2.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      bundler_version_cmd = "./spec/assets/bundler-exerciser.sh bundler-1.11.2.tgz ./bin/bundle -v"
      output, status = run(bundler_version_cmd)

      expect(status).to be_success
      expect(output).to include('Bundler version 1.11.2')
    end
  end
end

