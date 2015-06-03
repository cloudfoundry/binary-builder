require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  def run_binary_builder(binary_name, binary_version, flags = '')
    binary_builder_cmd = "#{File.join('./bin', 'binary-builder')} #{binary_name} #{binary_version} #{flags}"
    run(binary_builder_cmd)[0]
  end

  before do
    output, _ = run_binary_builder(binary_name, binary_version)
  end

  context 'when node is specified', binary: 'node' do
    let(:binary_name) { 'node' }
    let(:binary_version) { 'v0.12.2' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'node-v0.12.2-linux-x64.tar.gz')
      expect(File).to exist(binary_tarball_location)

      node_version_cmd = %q{./spec/assets/binary-exerciser.sh node-v0.12.2-linux-x64.tar.gz node-v0.12.2-linux-x64/bin/node -e 'console.log(process.version)'}
      output, status = run(node_version_cmd)

      expect(status).to be_success
      expect(output).to include('v0.12.2')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when ruby is specified', binary: 'ruby' do
    let(:binary_name) { 'ruby' }
    let(:binary_version) { 'v2_0_0_645' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'ruby-v2_0_0_645-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      ruby_version_cmd = %q{./spec/assets/binary-exerciser.sh ruby-v2_0_0_645-linux-x64.tgz ./bin/ruby -e 'puts RUBY_VERSION'}
      output, status = run(ruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.0.0')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when jruby is specified', binary: 'jruby' do
    let(:binary_name) { 'jruby' }
    let(:binary_version) { 'ruby-2.2.0-jruby-9.0.0.0.pre1' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'jruby-ruby-2.2.0-jruby-9.0.0.0.pre1-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      jruby_version_cmd = %q{./spec/assets/jruby-exerciser.sh}
      output, status = run(jruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('java 2.2.0')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when python is specified', binary: 'python' do
    let(:binary_name) { 'python' }
    let(:binary_version) { '3.4.3' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'python-3.4.3-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      python_version_cmd = %q{env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh python-3.4.3-linux-x64.tgz ./bin/python -c 'import sys;print(sys.version[:5])'}
      output, status = run(python_version_cmd)

      expect(status).to be_success
      expect(output).to include('3.4.3')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when httpd is specified', binary: 'httpd' do
    let(:binary_name) { 'httpd' }
    let(:binary_version) { '2.4.12' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'httpd-2.4.12-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      httpd_version_cmd = %q{env LD_LIBRARY_PATH=/tmp/binary-exerciser/lib ./spec/assets/binary-exerciser.sh httpd-2.4.12-linux-x64.tgz ./bin/httpd -v}

      output, status = run(httpd_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.4.12')
      FileUtils.rm(binary_tarball_location)
    end
  end

  context 'when php is specified', binary: 'php' do
    let(:binary_name) { 'php' }
    let(:binary_version) { '5.6.7' }

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'php-5.6.7-linux-x64.tgz')
      expect(File.exist?(binary_tarball_location)).to be(true)

      docker_exerciser = "docker run -v #{File.expand_path('../../..', __FILE__)}:/binary-builder:ro cloudfoundry/cflinuxfs2 /binary-builder/spec/assets/binary-exerciser.sh"
      exerciser_args = "php-5.6.7-linux-x64.tgz ./bin/php -r 'echo phpversion();'"

      script_output = `#{docker_exerciser} #{exerciser_args}`.chomp
      expect($?).to be_success
      expect(script_output).to eq('5.6.7')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
