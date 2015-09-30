require_relative 'openjdk7'
require_relative 'maven'
require_relative 'jruby'


class JRubyMeal
  def initialize(name, version, options={})
    @name    = name
    @version = version
    @options = options
  end

  def cook
    openjdk = OpenJDK7Recipe.new('openjdk', '7')
    openjdk.cook

    maven = MavenRecipe.new('maven', '3.3.3', {
      md5: '794b3b7961200c542a7292682d21ba36'
    })
    maven.cook
    maven.activate

    jruby.cook
  end

  def url
    jruby.url
  end

  def tar
    jruby.tar
  end

  private

  def jruby
    @jruby ||= JRubyRecipe.new(@name, @version, @options)
  end
end
