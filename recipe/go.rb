# encoding: utf-8
require_relative 'base'
require_relative '../lib/utils'

class GoRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go1.24.2 to $HOME/go1.24
    go124_sha256 = '68097bd680839cbc9d464a0edce4f7c333975e27a90246890e9f1078c7e702ad'

    Dir.chdir("#{ENV['HOME']}") do
      go_download_uri = "https://go.dev/dl/go1.24.2.linux-amd64.tar.gz"
      go_tar = "go.tar.gz"
      HTTPHelper.download(go_download_uri, go_tar, "sha256", go124_sha256)

      system("tar xf #{go_tar}")
      system("mv ./go ./go1.24")
    end

    # The GOROOT_BOOTSTRAP defaults to $HOME/go1.4 so we need to update it for this command
    Dir.chdir("#{tmp_path}/go/src") do
      system(
        'GOROOT_BOOTSTRAP=$HOME/go1.24 ./make.bash'
      ) or raise "Could not install go"
    end
  end

  def archive_files
    ["#{tmp_path}/go/*"]
  end

  def archive_path_name
    'go'
  end

  def archive_filename
    "#{name}#{version}.linux-amd64.tar.gz"
  end

  def url
    "https://go.dev/dl/go#{version}.src.tar.gz"
  end

end
