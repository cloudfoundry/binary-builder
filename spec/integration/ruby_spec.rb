require 'spec_helper'
require 'fileutils'


describe 'building a binary', :integration do
  context 'when ruby is specified', binary: 'ruby' do
    before do
      run_binary_builder('ruby', '2.2.3', 'df795f2f99860745a416092a4004b016ccf77e8b82dec956b120f18bdc71edce')
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      binary_tarball_location = File.join(Dir.pwd, 'ruby-2.2.3-linux-x64.tgz')
      expect(File).to exist(binary_tarball_location)

      ruby_version_cmd = %q{./spec/assets/binary-exerciser.sh ruby-2.2.3-linux-x64.tgz ./bin/ruby -e 'puts RUBY_VERSION'}
      output, status = run(ruby_version_cmd)

      expect(status).to be_success
      expect(output).to include('2.2.3')
      FileUtils.rm(binary_tarball_location)
    end
  end
end
