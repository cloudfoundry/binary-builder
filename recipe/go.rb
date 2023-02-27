# encoding: utf-8
require_relative 'base'

class GoRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go1.20.1 to $HOME/go1.20
    go1201_sha256 = '000a5b1fca4f75895f78befeb2eecf10bfff3c428597f3f1e69133b63b911b02'

    Dir.chdir("#{ENV['HOME']}") do
      go_download_uri = "https://go.dev/dl/go1.20.1.linux-amd64.tar.gz"
      go_tar = "go.tar.gz"
      HTTPHelper.download(go_download_uri, go_tar, "sha256", go1201_sha256)

      system("tar xf #{go_tar}")
      system("mv ./go ./go1.20")
    end

    # The GOROOT_BOOTSTRAP defaults to $HOME/go1.4 so we need to update it for this command
    Dir.chdir("#{tmp_path}/go/src") do
      system(
        'GOROOT_BOOTSTRAP=$HOME/go1.20 ./make.bash'
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
