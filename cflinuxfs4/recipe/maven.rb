# frozen_string_literal: true

require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "https://archive.apache.org/dist/maven/maven-3/#{version}/source/apache-maven-#{version}-src.tar.gz"
  end

  def cook
    download unless downloaded?
    extract

    # install maven 3.6.3 to $HOME/apache-maven-3.6.3
    sha512 = 'c35a1803a6e70a126e80b2b3ae33eed961f83ed74d18fcd16909b2d44d7dada3203f1ffe726c17ef8dcca2dcaa9fca676987befeadc9b9f759967a8cb77181c0'

    Dir.chdir((ENV['HOME']).to_s) do
      maven_download = "https://archive.apache.org/dist/maven/maven-3/#{version}/binaries/apache-maven-#{version}-bin.tar.gz"
      maven_tar = "apache-maven-#{version}-bin.tar.gz"
      system("curl -L #{maven_download} -o #{maven_tar}")

      downloaded_sha = Digest::SHA512.file(maven_tar).hexdigest

      raise "sha512 verification failed: expected #{sha512}, got #{downloaded_sha}" if sha512 != downloaded_sha

      system("tar xf #{maven_tar}")
    end

    old_path = ENV['PATH']
    ENV['PATH'] = "#{ENV['HOME']}/apache-maven-3.6.3/bin:#{old_path}"

    install
    ENV['PATH'] = old_path
    FileUtils.rm_rf(File.join(ENV['HOME'], 'apache-maven-3.6.3'))
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
