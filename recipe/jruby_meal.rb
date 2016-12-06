# encoding: utf-8
require_relative 'ant'
require_relative 'jruby'
require_relative 'maven'
require 'fileutils'

class JRubyMeal
  attr_reader :name, :version

  def initialize(name, version, options = {})
    @name    = name
    @version = version
    @options = options
  end

  def cook
    # We compile against the OpenJDK8 that the java buildpack team builds
    # This is the openjdk-jdk that contains the openjdk-jre used in the ruby buildpack
    # Ubuntu Trusty itself does not provide openjdk 8

    java_jdk_dir = '/opt/java'
    java_jdk_tar_file = File.join(java_jdk_dir, 'openjdk-8-jdk.tar.gz')
    java_jdk_bin_dir = File.join(java_jdk_dir, 'bin')
    java_buildpack_java_sdk = "https://java-buildpack.cloudfoundry.org/openjdk-jdk/trusty/x86_64/openjdk-1.8.0_111.tar.gz"

    FileUtils.mkdir_p(java_jdk_dir)
    raise "Downloading openjdk-8-jdk failed." unless system("wget #{java_buildpack_java_sdk} -O #{java_jdk_tar_file}")
    raise "Untarring openjdk-8-jdk failed." unless system("tar xvf #{java_jdk_tar_file} -C #{java_jdk_dir}")

    ENV['JAVA_HOME'] = java_jdk_dir
    ENV['PATH'] = "#{ENV['PATH']}:#{java_jdk_bin_dir}"

    ant.cook
    ant.activate

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
    ant.send(:files_hashs) +
      maven.send(:files_hashs) +
      jruby.send(:files_hashs)
  end

  def jruby
    @jruby ||= JRubyRecipe.new(@name, @version, @options)
  end

  def maven
    @maven ||= MavenRecipe.new('maven', '3.3.9', md5: '030ce5b3d369f01aca6249b694d4ce03')
  end

  def ant
    @ant ||= AntRecipe.new('ant', '1.9.7', md5: 'a2fd9458c76700b7be51ef12f07d4bb1')
  end
end
