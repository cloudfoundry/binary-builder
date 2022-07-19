# frozen_string_literal: true

require 'spec_helper'
require 'yaml'

describe 'building a binary', :integration do
  context 'when a recipe is specified' do
    before(:all) do
      @output, = run_binary_builder('glide', 'v0.11.0', '--sha256=7a7023aff20ba695706a262b8c07840ee28b939ea6358efbb69ab77da04f0052')
      @binary_tarball_location = File.join(Dir.pwd, 'glide-v0.11.0-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'prints the url of the source used to build the binary to stdout' do
      puts @output
      expect(@output).to include('Source URL: https://github.com/Masterminds/glide/archive/v0.11.0.tar.gz')
    end
  end

  context 'when a meal is specified' do
    before(:all) do
      @output, = run_binary_builder('httpd', '2.4.41', '--sha256=133d48298fe5315ae9366a0ec66282fa4040efa5d566174481077ade7d18ea40')
      @binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'httpd-2.4.41-linux-x64*.tgz')).first
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'prints the url of the source used to build the binary to stdout' do
      puts @output
      expect(@output).to include('Source URL: https://archive.apache.org/dist/httpd/httpd-2.4.41.tar.bz2')
    end
  end
end
