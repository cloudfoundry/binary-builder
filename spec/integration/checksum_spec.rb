require 'spec_helper'

describe 'With a checksum', :integration do
  context 'that is not valid for the source tarball' do
    it 'displays and exits with an error' do
      output, status = run_binary_builder('httpd', '2.4.12', 'invalid-checksum')

      expect(output).to include 'Checksum is not matching!'
      expect(status.exitstatus).to eq 1
    end
  end
end
