require 'spec_helper'

describe 'With a checksum', :integration do
  context 'that is not valid for the source tarball' do
    it 'displays and exits with an error' do
      output, status = run_binary_builder('ruby', '2.2.3', '--sha256=invalid-checksum')

      expect(output).to include 'Checksum is not matching!'
      expect(status.exitstatus).to eq 1
    end
  end

  context 'when nginx is specified with invalid GPG key' do
    it 'builds the specified binary, tars it, and places it in your current working directory' do
      output, status = run_binary_builder('nginx', '1.9.4', '--gpg-rsa-key-id=A1C052F8 --gpg-key="-----BEGIN PGP SIGNATURE-----
Version: GnuPG v2

iQEcBAABCAAGBQJV002uAAoJEFIKmZOhwFL41AcH/2VX1/5mD3dAUXfDaYMG92IV
aA8vHlsvXpCEPfCYBnPGYYFa/P0qPyw6hsWXZhWEGEm+BqZK6dWCLFaxTVTtsjOE
vhSR+LL+FNxYmGbK2lYq61PDDL45x5Qnhy3WK1e40F7CqmElSfMOjLuCNC7xR9Jc
zAZ014ADQ5yfH+Ma40K997AxZeCVGU+A5IEHGoZ2i8pyqx0Jhh6cbpC18yHu5ciN
0o4E4cLSFFckYB3FnUpDowRonBDNUpDRJVKMo5cvvskc/GWVUVomPuWyNGFPPmMJ
=zjw3
-----END PGP SIGNATURE-----"')
      expect(output).to include 'Checksum is not matching!'
      expect(status.exitstatus).to eq 1
    end
  end
end
