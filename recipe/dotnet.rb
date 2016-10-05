# encoding: utf-8
require_relative 'base'

class DotNetRecipe < BaseRecipe
  attr_reader :name, :version

  def extract_file(source, target)
    FileUtils.mkdir_p target

    message "Copying #{source} into #{target}... "
    system("cp -r #{source} #{target}")
  end

  def cook
    download unless downloaded?
    extract

    system(<<~CMD)
              sudo apt-get update
              sudo apt-get -y upgrade
              sudo apt-get -y install \
                clang \
                devscripts \
                debhelper \
                libunwind8 \
                liburcu1 \
                libpython2.7 \
                liblttng-ust0 \
                libllvm3.6 \
                liblldb-3.6
              CMD

    Dir.chdir("#{tmp_path}/cli") do
      raise 'Could not build dotnet' unless system('./build.sh --targets Prepare,Compile')
    end
  end

  def archive_files
    ["#{tmp_path}/cli/artifacts/ubuntu.14.04-x64/stage2/*"]
  end

  def archive_filename
    dotnet_version = `#{tmp_path}/cli/artifacts/ubuntu.14.04-x64/stage2/dotnet --version`.strip
    "#{name}.#{dotnet_version}.linux-amd64.tar.gz"
  end

  def url
    "https://github.com/dotnet/cli"
  end

end
