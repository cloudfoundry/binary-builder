# encoding: utf-8
require_relative 'base'

class GoRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go1.4.3 to $HOME/go1.4
    go143_sha256 = 'ce3140662f45356eb78bc16a88fc7cfb29fb00e18d7c632608245b789b2086d2'

    Dir.chdir("#{ENV['HOME']}") do
      go_download = "https://storage.googleapis.com/golang/go1.4.3.linux-amd64.tar.gz"
      go_tar = "go.tar.gz"
      system("curl -L #{go_download} -o #{go_tar}")

      downloaded_sha = Digest::SHA256.file(go_tar).hexdigest

      if go143_sha256 != downloaded_sha
        raise "sha256 verification failed: expected #{go_sha256}, got #{downloaded_sha}"
      end

      system("tar xf #{go_tar}")
      system("mv ./go ./go1.4")
    end

    Dir.chdir("#{tmp_path}/go/src") do
      system(
        './make.bash'
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
    "https://storage.googleapis.com/golang/go#{version}.src.tar.gz"
  end

end
