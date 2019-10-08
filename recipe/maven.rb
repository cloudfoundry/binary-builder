# encoding: utf-8
require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "https://archive.apache.org/dist/maven/maven-3/#{version}/source/apache-maven-#{version}-src.tar.gz"
  end

  def cook
    download unless downloaded?
    extract

    #install maven 3.6.1 to $HOME/apache-maven-3.6.1
    sha512 = 'fae9c12b570c3ba18116a4e26ea524b29f7279c17cbaadc3326ca72927368924d9131d11b9e851b8dc9162228b6fdea955446be41207a5cfc61283dd8a561d2f'

    Dir.chdir("#{ENV['HOME']}") do
      maven_download = 'https://archive.apache.org/dist/maven/maven-3/3.6.0/binaries/apache-maven-3.6.0-bin.tar.gz'
      maven_tar = "apache-maven-3.6.0-bin.tar.gz"
      system("curl -L #{maven_download} -o #{maven_tar}")

      downloaded_sha = Digest::SHA512.file(maven_tar).hexdigest

      if sha512 != downloaded_sha
        raise "sha512 verification failed: expected #{sha512}, got #{downloaded_sha}"
      end

      system("tar xf #{maven_tar}")
    end

    old_path = ENV['PATH']
    ENV['PATH'] = "#{ENV['HOME']}/apache-maven-3.6.0/bin:#{old_path}"

    install
    ENV['PATH'] = old_path
    FileUtils.rm_rf(File.join(ENV['HOME'], 'apache-maven-3.6.0'))
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
