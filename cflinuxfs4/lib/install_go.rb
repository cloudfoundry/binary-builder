# frozen_string_literal: true
require_relative 'utils'

def install_go_compiler
  go_compiler_info = YAML.load_file(File.join(__dir__, '..', 'go-version.yml'))['go'].first
  go_version = go_compiler_info['version']
  go_sha256 = go_compiler_info['sha256']

  Dir.chdir('/usr/local') do
    go_download = "https://dl.google.com/go/go#{go_version}.linux-amd64.tar.gz"
    go_tar = 'go.tar.gz'

    HTTPHelper.download(go_download, go_tar, "sha256", go_sha256)

    system("tar xf #{go_tar}")
  end
end
