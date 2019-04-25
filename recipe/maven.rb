# encoding: utf-8
require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "https://www.apache.org/dist/maven/maven-3/#{version}/source/apache-maven-#{version}-src.tar.gz"
  end

  def cook
    download unless downloaded?
    extract

    #install maven 3.6.1 to $HOME/apache-maven-3.6.1
    mvn361_sha512 = 'b4880fb7a3d81edd190a029440cdf17f308621af68475a4fe976296e71ff4a4b546dd6d8a58aaafba334d309cc11e638c52808a4b0e818fc0fd544226d952544'

    Dir.chdir("#{ENV['HOME']}") do
      maven_download = 'https://www.apache.org/dist/maven/maven-3/3.6.1/binaries/apache-maven-3.6.1-bin.tar.gz'
      maven_tar = "apache-maven-3.6.1-bin.tar.gz"
      system("curl -L #{maven_download} -o #{maven_tar}")

      downloaded_sha = Digest::SHA512.file(maven_tar).hexdigest

      if mvn361_sha512 != downloaded_sha
        raise "sha512 verification failed: expected #{mvn361_sha512}, got #{downloaded_sha}"
      end

      system("tar xf #{maven_tar}")
    end

    old_path = ENV['PATH']
    ENV['PATH'] = "#{ENV['HOME']}/apache-maven-3.6.1/bin:#{old_path}"

    install
    ENV['PATH'] = old_path
    FileUtils.rm_rf(File.join(ENV['HOME'], 'apache-maven-3.6.1'))
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
