def install_go_compiler
  go_compiler_info = YAML.load_file(File.join(__dir__, '..', "go-version.yml"))['go'].first
  go_version = go_compiler_info['version']
  go_sha256 = go_compiler_info['sha256']

  Dir.chdir("/usr/local") do
    go_download = "https://storage.googleapis.com/golang/go#{go_version}.linux-amd64.tar.gz"
    go_tar = "go.tar.gz"

    system("curl -L #{go_download} -o #{go_tar}")

    downloaded_sha = Digest::SHA256.file(go_tar).hexdigest

    if go_sha256 != downloaded_sha
      raise "sha256 verification failed: expected #{go_sha256}, got #{downloaded_sha}"
    end

    system("tar xf #{go_tar}")
  end
end
