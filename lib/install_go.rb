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

  Dir.chdir("/usr/local") do
    go_download = "https://go.dev/dl/go#{go_version}.linux-amd64.tar.gz"
    go_tar = "go.tar.gz"

    system("curl -L #{go_download} -o #{go_tar}")

    downloaded_sha = Digest::SHA256.file(go_tar).hexdigest

    if go_sha256 != downloaded_sha
      raise "sha256 verification failed: expected #{go_sha256}, got #{downloaded_sha}"
    end

    system("tar xf #{go_tar}")
  end
end
