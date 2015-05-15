require 'digest'
require 'fileutils'
require 'spec_helper'

describe 'building a binary' do
  before do
    run_binary_builder(binary_name, binary_version)
  end

  context 'when node is specified', binary: 'node' do
    let(:binary_name) { 'node' }
    let(:binary_version) { 'v0.12.2' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = "node-v0.12.2-linux-x64.tgz node-v0.12.2-linux-x64/bin/node -e 'console.log(process.version)'"

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('v0.12.2')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when ruby is specified', binary: 'ruby' do
    let(:binary_name) { 'ruby' }
    let(:binary_version) { 'v2_0_0_645' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'ruby-v2_0_0_645-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = %q{ruby-v2_0_0_645-linux-x64.tgz ./bin/ruby -e 'puts RUBY_VERSION'}

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('2.0.0')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when jruby is specified', binary: 'jruby' do
    let(:binary_name) { 'jruby' }
    let(:binary_version) { 'ruby-2.2.0-jruby-9.0.0.0.pre1' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'jruby-ruby-2.2.0-jruby-9.0.0.0.pre1-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/jruby-exerciser.sh"

      script_output = `#{docker_exerciser}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('java 2.2.0')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when python is specified', binary: 'python' do
    let(:binary_name) { 'python' }
    let(:binary_version) { '3.4.3' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'python-3.4.3-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro -e LD_LIBRARY_PATH=/binary-exerciser/lib cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = "python-3.4.3-linux-x64.tgz ./bin/python -c 'import sys;print(sys.version[:5])'"

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('3.4.3')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
