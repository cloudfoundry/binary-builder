# frozen_string_literal: true

require_relative 'base'
require_relative '../lib/utils'

class GoRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    # Installs go1.4.3 to $HOME/go1.4
    go143_sha256 = 'ce3140662f45356eb78bc16a88fc7cfb29fb00e18d7c632608245b789b2086d2'

    Dir.chdir((ENV['HOME']).to_s) do
      go_download_uri = 'https://dl.google.com/go/go1.4.3.linux-amd64.tar.gz'
      go_tar = 'go.tar.gz'
      HTTPHelper.download(go_download_uri, go_tar, "sha256", go143_sha256)

      system("tar xf #{go_tar}")
      system('mv ./go ./go1.4')
    end

    Dir.chdir("#{tmp_path}/go/src") do
      system(
        './make.bash'
      ) or raise 'Could not install go'
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
    "https://dl.google.com/go/go#{version}.src.tar.gz"
  end
end
