# encoding: utf-8
require 'spec_helper'

describe 'building a binary', :integration do
  context 'when nginx is specified' do
    before(:all) do
      run_binary_builder('nginx', '1.9.4', '--gpg-rsa-key-id=A1C052F8 --gpg-signature="-----BEGIN PGP SIGNATURE-----
Version: GnuPG v2

iQEcBAABCAAGBQJV002uAAoJEFIKmZOhwFL41AcH/2VX1/5mD3dAUXfDaYMG92IV
aA8vHlsvXpCEPfCYBnPGYYFa/P0qPyw6hsWXZhWEGEm+BqZK6dWCLFaxTVTtsjOE
vhSR+LL+FNxYmGbK2lYq61PDDL45x5Qnhy3WK1e40F7CqmElSfMOjLuCNC7xR9Jc
zAZ014ADQ5yfH+Ma40K997AxZeCVGU+A5IEHGoZ2i8pyqx0Jhh6cbpC18yHu5ciN
0o4E4cLSFFckYB3FnUpDowRonBDNUpDRJVKMo5cvvskc/GWVUVomPuWyNGFPPmMJ
aySUQcOvO67Z14d9E9ziX/E24KWl6xRymmy9VhzawgSmf//3yZVaD6C/8om3qMw=
=zjw3
-----END PGP SIGNATURE-----"')
      @binary_tarball_location = File.join(Dir.pwd, 'nginx-1.9.4-linux-x64.tgz')
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'builds the specified binary, tars it, and places it in your current working directory' do
      expect(File).to exist(@binary_tarball_location)

      httpd_version_cmd = './spec/assets/binary-exerciser.sh nginx-1.9.4-linux-x64.tgz ./nginx/sbin/nginx -v'
      output, status = run(httpd_version_cmd)

      expect(status).to be_success
      expect(output).to include('1.9.4')
    end
  end
end
