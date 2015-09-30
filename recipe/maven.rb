require_relative 'base'

class MavenRecipe < BaseRecipe
  def url
    "http://www.gtlib.gatech.edu/pub/apache/maven/maven-3/#{version}/binaries/apache-maven-#{version}-bin.tar.gz"
  end

  def cook
    download unless downloaded?
    extract
    install
  end

  def install
    FileUtils.mkdir_p(path)
    execute("install", ['cp', '-r', '.', path])
  end
end


