# encoding: utf-8
require_relative 'jruby'
require_relative 'maven'
require 'fileutils'
require 'digest'

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
    java_jdk_sha256 = '28d7e4757649d9dc5f40e0de572ad27c9e96830be9764a58948a58bb78b1d954'
    java_buildpack_java_sdk = "https://java-buildpack.cloudfoundry.org/openjdk/trusty/x86_64/openjdk-1.8.0_172.tar.gz"

    FileUtils.mkdir_p(java_jdk_dir)
    raise "Downloading openjdk-8-jdk failed." unless system("wget #{java_buildpack_java_sdk} -O #{java_jdk_tar_file}")

    downloaded_sha = Digest::SHA256.file(java_jdk_tar_file).hexdigest

    if java_jdk_sha256 != downloaded_sha
      raise "sha256 verification failed: expected #{java_jdk_sha256}, got #{downloaded_sha}"
    end

    raise "Untarring openjdk-8-jdk failed." unless system("tar xvf #{java_jdk_tar_file} -C #{java_jdk_dir}")

    ENV['JAVA_HOME'] = java_jdk_dir
    ENV['PATH'] = "#{ENV['PATH']}:#{java_jdk_bin_dir}"

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
    maven.send(:files_hashs) +
    jruby.send(:files_hashs)
  end

  def jruby
    @jruby ||= JRubyRecipe.new(@name, @version, @options)
  end

  def maven
    @maven ||= MavenRecipe.new('maven', '3.5.3', md5: '240b880cd7294665d7228f74f6453984')
  end
end
