# encoding: utf-8
require 'spec_helper'
require 'yaml'

describe 'building a binary', :integration do
  context 'when a recipe is specified' do
    before(:all) do
        @output, _ = run_binary_builder('go', '1.6.3', '--sha256=6326aeed5f86cf18f16d6dc831405614f855e2d416a91fd3fdc334f772345b00')
        @tarball_name = 'go1.6.3.linux-amd64.tar.gz'
        @binary_tarball_location = File.join(Dir.pwd, @tarball_name)
    end

    after(:all) do
      FileUtils.rm(@binary_tarball_location)
    end

    it 'prints a yaml representation of the source used to build the binary to stdout' do
      yaml_source = @output.match(/Source YAML:(.*)/m)[1]
      expect(YAML.load(yaml_source)).to eq([
                                            {
                                              "sha256"=>"6326aeed5f86cf18f16d6dc831405614f855e2d416a91fd3fdc334f772345b00",
                                              "url"=>"https://storage.googleapis.com/golang/go1.6.3.src.tar.gz"
                                            }
                                          ])
    end

    it 'includes the yaml representation of the source inside the resulting tarball' do
      yaml_source = `tar xzf #{@tarball_name} -O sources.yml`
      expect(YAML.load(yaml_source)).to eq([
                                              {
                                                "sha256"=>"6326aeed5f86cf18f16d6dc831405614f855e2d416a91fd3fdc334f772345b00",
                                                "url"=>"https://storage.googleapis.com/golang/go1.6.3.src.tar.gz"
                                              }
                                          ])
    end
  end

  context 'when a meal is specified' do
    before(:all) do
      @output, = run_binary_builder('httpd', '2.4.41', '--sha256=133d48298fe5315ae9366a0ec66282fa4040efa5d566174481077ade7d18ea40')
      @binary_tarball_location = Dir.glob(File.join(Dir.pwd, 'httpd-2.4.41-linux-x64*.tgz')).first
    end

    it 'prints a yaml representation of the source used to build the binary to stdout' do
      yaml_source = @output.match(/Source YAML:(.*)/m)[1]
      expect(YAML.load(yaml_source)).to match_array([
        {
          "sha256" => "133d48298fe5315ae9366a0ec66282fa4040efa5d566174481077ade7d18ea40",
          "url" => "https://archive.apache.org/dist/httpd/httpd-2.4.41.tar.bz2"
        },
        {
          "sha256" => "48e9dbf45ae3fdc7b491259ffb6ccf7d63049ffacbc1c0977cced095e4c2d5a2",
          "url" => "https://apache.osuosl.org/apr/apr-1.7.0.tar.gz"
        },
        {
          "sha256" => "ce94c7722ede927ce1e5a368675ace17d96d60ff9b8918df216ee5c1298c6a5e",
          "url" => "https://apache.osuosl.org/apr/apr-iconv-1.2.2.tar.gz"
        },
        {
          "sha256" => "b65e40713da57d004123b6319828be7f1273fbc6490e145874ee1177e112c459",
          "url" => "https://apache.osuosl.org/apr/apr-util-1.6.1.tar.gz"
        },
        {
          "sha256" => "0f078444fed34085bc83e27eb3439556718f50dcea275307ffb66d498bdabb8f",
          "url" => "https://github.com/zmartzone/mod_auth_openidc/releases/download/v2.3.8/mod_auth_openidc-2.3.8.tar.gz"
        }
      ])
    end

    it 'includes the yaml representation of the source inside the resulting tarball' do
      yaml_source = `tar xzf httpd-2.4.41-linux-x64.tgz sources.yml -O`
      expect(YAML.load(yaml_source)).to match_array([
        {
          "sha256" => "133d48298fe5315ae9366a0ec66282fa4040efa5d566174481077ade7d18ea40",
          "url" => "https://archive.apache.org/dist/httpd/httpd-2.4.41.tar.bz2"
        },
        {
          "sha256" => "48e9dbf45ae3fdc7b491259ffb6ccf7d63049ffacbc1c0977cced095e4c2d5a2",
          "url" => "https://apache.osuosl.org/apr/apr-1.7.0.tar.gz"
        },
        {
          "sha256" => "ce94c7722ede927ce1e5a368675ace17d96d60ff9b8918df216ee5c1298c6a5e",
          "url" => "https://apache.osuosl.org/apr/apr-iconv-1.2.2.tar.gz"
        },
        {
          "sha256" => "b65e40713da57d004123b6319828be7f1273fbc6490e145874ee1177e112c459",
          "url" => "https://apache.osuosl.org/apr/apr-util-1.6.1.tar.gz"
        },
        {
          "sha256" => "0f078444fed34085bc83e27eb3439556718f50dcea275307ffb66d498bdabb8f",
          "url" => "https://github.com/zmartzone/mod_auth_openidc/releases/download/v2.3.8/mod_auth_openidc-2.3.8.tar.gz"
        }
      ])
    end
  end
end
