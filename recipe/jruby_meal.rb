require_relative 'openjdk7'
require_relative 'maven'
require_relative 'jruby'


class JRubyMeal
  attr_reader :name, :version

  def initialize(name, version, options={})
    @name    = name
    @version = version
    @options = options
  end

  def cook
    # NOTE: We compile against OpenJDK7 because trusty does not support
    # OpenJDK8. Unable to use java-buildpack OpenJDK8 because it only contains
    # the JRE, not the JDK.
    # https://www.pivotaltracker.com/story/show/106836266
    openjdk.cook

    maven.cook
    maven.activate

    jruby.cook
  end

  def url
    jruby.url
  end

  def archive_files
    jruby.archive_files
  end

  def archive_path_name
    jruby.archive_path_name
  end

  def archive_filename
    jruby.archive_filename
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
