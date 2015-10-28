require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "https://www.apache.org/dist/maven/maven-3/#{version}/source/apache-maven-#{version}-src.tar.gz"
  end

  def cook
    download unless downloaded?
    extract
    install
  end

  def install
    FileUtils.rm_rf(path)
    execute("install", [
      'ant',
      '-noinput',
      "-Dmaven.home=#{path}"
    ])
  end
end


