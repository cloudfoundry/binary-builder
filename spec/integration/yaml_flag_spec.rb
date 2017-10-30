# encoding: utf-8
require 'spec_helper'
require 'yaml'

describe 'building a binary', :integration do
  context 'when a recipe is specified' do
    before(:all) do
      @output, = run_binary_builder('nginx', '1.9.4', '--gpg-rsa-key-id=A1C052F8 --gpg-signature="-----BEGIN PGP SIGNATURE-----
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

    it 'prints a yaml representation of the source used to build the binary to stdout' do
      yaml_source = @output.match(/Source YAML:(.*)/m)[1]
      expect(YAML.load(yaml_source)).to eq([
                                             {
                                               'url'    => 'http://nginx.org/download/nginx-1.9.4.tar.gz',
                                               'sha256' => '479b0c03747ee6b2d4a21046f89b06d178a2881ea80cfef160451325788f2ba8'
                                             }
                                           ])
    end

    it 'includes the yaml representation of the source inside the resulting tarball' do
      yaml_source = `tar xzf nginx-1.9.4-linux-x64.tgz sources.yml -O`
      expect(YAML.load(yaml_source)).to eq([
                                             {
                                               'url'    => 'http://nginx.org/download/nginx-1.9.4.tar.gz',
                                               'sha256' => '479b0c03747ee6b2d4a21046f89b06d178a2881ea80cfef160451325788f2ba8'
                                             }
                                           ])
    end
  end

  context 'when a meal is specified' do
    before(:all) do
      @output, = run_binary_builder('httpd', '2.4.12', '--md5=b8dc8367a57a8d548a9b4ce16d264a13')
      @binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'httpd-2.4.12-linux-x64*.tgz')).first
    end

    it 'prints a yaml representation of the source used to build the binary to stdout' do
      yaml_source = @output.match(/Source YAML:(.*)/m)[1]
      expect(YAML.load(yaml_source)).to match_array([
        {
          "url"=>"https://archive.apache.org/dist/httpd/httpd-2.4.12.tar.bz2",
          "sha256"=>"ad6d39edfe4621d8cc9a2791f6f8d6876943a9da41ac8533d77407a2e630eae4"
        }, {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-1.6.3.tar.gz",
          "sha256"=>"8fdabcc0004216c3588b7dca0f23d104dfe012a47e2bb6f13827534a6ee73aa7"
        }, {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-iconv-1.2.2.tar.gz",
          "sha256"=>"ce94c7722ede927ce1e5a368675ace17d96d60ff9b8918df216ee5c1298c6a5e"
        }, {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-util-1.6.1.tar.gz",
          "sha256"=>"b65e40713da57d004123b6319828be7f1273fbc6490e145874ee1177e112c459"
        }
      ])
    end

    it 'includes the yaml representation of the source inside the resulting tarball' do
      yaml_source = `tar xzf httpd-2.4.12-linux-x64.tgz sources.yml -O`
      expect(YAML.load(yaml_source)).to match_array([
        {
          "url"=>"https://archive.apache.org/dist/httpd/httpd-2.4.12.tar.bz2",
          "sha256"=>"ad6d39edfe4621d8cc9a2791f6f8d6876943a9da41ac8533d77407a2e630eae4"
        }, {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-1.6.3.tar.gz",
          "sha256"=>"8fdabcc0004216c3588b7dca0f23d104dfe012a47e2bb6f13827534a6ee73aa7"
        }, {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-iconv-1.2.2.tar.gz",
          "sha256"=>"ce94c7722ede927ce1e5a368675ace17d96d60ff9b8918df216ee5c1298c6a5e"
        },
        {
          "url"=>"http://apache.mirrors.tds.net/apr/apr-util-1.6.1.tar.gz",
          "sha256"=>"b65e40713da57d004123b6319828be7f1273fbc6490e145874ee1177e112c459"
        }
      ])
    end
  end
end
