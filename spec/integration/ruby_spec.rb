# encoding: utf-8
require 'spec_helper'
require 'fileutils'

describe 'building a binary', :integration do
  context 'when ruby is specified' do
    before(:all) do
      run_binary_builder('ruby', '2.2.3', '--sha256=df795f2f99860745a416092a4004b016ccf77e8b82dec956b120f18bdc71edce')
      @binary_tarball_location = File.join(Dir.pwd, 'ruby-2.2.3-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      ruby_version_cmd = "./spec/assets/binary-exerciser.sh ruby-2.2.3-linux-x64.tgz ./bin/ruby -e 'puts RUBY_VERSION'"
      output, status = run(ruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.2.3')

      libgmp_cmd = "./spec/assets/binary-exerciser.sh ruby-2.2.3-linux-x64.tgz grep LIBS= lib/pkgconfig/ruby-2.2.pc"
      output, status = run(libgmp_cmd)

      expect(status).to be_success
      expect(output).to include('LIBS=')
      expect(output).not_to include('lgmp')
    end
  end
end
