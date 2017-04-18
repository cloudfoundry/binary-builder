# encoding: utf-8
require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "https://www.apache.org/dist/maven/maven-3/#{version}/source/apache-maven-#{version}-src.tar.gz"
  end

  def cook
    download unless downloaded?
    extract

    #install maven 3.3.9 to $HOME/apache-maven-3.3.9
    mvn339_sha256 = '6e3e9c949ab4695a204f74038717aa7b2689b1be94875899ac1b3fe42800ff82'

    Dir.chdir("#{ENV['HOME']}") do
      maven_download = 'https://www.apache.org/dist/maven/maven-3/3.3.9/binaries/apache-maven-3.3.9-bin.tar.gz'
      maven_tar = "apache-maven-3.3.9-bin.tar.gz"
      system("curl -L #{maven_download} -o #{maven_tar}")

      downloaded_sha = Digest::SHA256.file(maven_tar).hexdigest

      if mvn339_sha256 != downloaded_sha
        raise "sha256 verification failed: expected #{go_sha256}, got #{downloaded_sha}"
      end

      system("tar xf #{maven_tar}")
    end

    old_path = ENV['PATH']
    ENV['PATH'] = "#{ENV['HOME']}/apache-maven-3.3.9/bin:#{old_path}"

    install
    ENV['PATH'] = old_path
    FileUtils.rm_rf(File.join(ENV['HOME'], 'apache-maven-3.3.9'))
  end

  def install
    FileUtils.rm_rf(path)
    execute('install', [
              'mvn',
              "-DdistributionTargetDir=#{path}",
              'clean',
              'package'
            ])
  end
end
