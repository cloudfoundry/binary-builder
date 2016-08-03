# encoding: utf-8
require_relative 'base'

class GoRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go1.4.3 to $HOME/go1.4
    Dir.chdir("#{ENV['HOME']}") do
      go_download = "https://storage.googleapis.com/golang/go1.4.3.linux-amd64.tar.gz"
      go_tar = "go.tar.gz"
      system("curl -L #{go_download} -o #{go_tar}")
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
