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
    openjdk.cook

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

  def files_hashs
    maven.send(:files_hashs)   +
    jruby.send(:files_hashs)
  end

  def jruby
    @jruby ||= JRubyRecipe.new(@name, @version, @options)
  end

  def openjdk
    @openjdk ||= OpenJDK7Recipe.new('openjdk', '7')
  end

  def maven
    @maven ||= MavenRecipe.new('maven', '3.3.3', {
      md5: '794b3b7961200c542a7292682d21ba36'
    })
  end
end
