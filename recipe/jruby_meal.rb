require_relative 'openjdk7'
require_relative 'maven'
require_relative 'jruby'


class JRubyMeal
  attr_accessor :files

  def initialize(name, version)
    @name    = name
    @version = version
    @files   = []
  end

  def cook
    openjdk = OpenJDK7Recipe.new('openjdk', '7')
    openjdk.cook

    maven = MavenRecipe.new('maven', '3.3.3')
    maven.files << {
      url: maven.url,
      md5: '794b3b7961200c542a7292682d21ba36'
    }
    maven.cook
    maven.activate

    jruby.files = self.files
    jruby.cook
  end

  def url
    jruby.url
  end

  private

  def jruby
    @jruby ||= JRubyRecipe.new(@name, @version)
  end
end
