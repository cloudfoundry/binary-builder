# frozen_string_literal: true
require_relative 'utils'
require 'net/http'
require 'json'

def install_go_compiler
  url = URI('https://go.dev/dl/?mode=json')
  res = Net::HTTP.get(url)
  latest_go = JSON.parse(res).first

  go_version = latest_go['version'].delete_prefix('go')
  go_sha256 = ""

  latest_go['files'].each do |file|
    if file['filename'] == "go#{go_version}.linux-amd64.tar.gz"
      go_sha256 = file['sha256']
      break
    end
  end

  Dir.chdir('/usr/local') do
    go_download = "https://go.dev/dl/go#{go_version}.linux-amd64.tar.gz"
    go_tar = 'go.tar.gz'

    HTTPHelper.download(go_download, go_tar, "sha256", go_sha256)

    system("tar xf #{go_tar}")
  end
end
