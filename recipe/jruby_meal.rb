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
    java_jdk_sha256 = '6254bacb4d9e6cae1e8fdd44f06bbb2c6f26081f937e3786e6c53aea57f94a4d'
    java_buildpack_java_sdk = "https://java-buildpack.cloudfoundry.org/openjdk-jdk/trusty/x86_64/openjdk-1.8.0_192.tar.gz"

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
    @maven ||= MavenRecipe.new('maven', '3.6.0', md5: '15bbfec555ec1b17d0d1a0fd14b7c329')
  end
end
