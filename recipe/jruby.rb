# encoding: utf-8
require 'mini_portile'
require_relative 'base'

class JRubyRecipe < BaseRecipe
  def archive_files
    [
      "#{work_path}/bin",
      "#{work_path}/lib"
    ]
  end

  def url
    "https://repo1.maven.org/maven2/org/jruby/jruby-dist/#{jruby_version}/jruby-dist-#{jruby_version}-src.zip"
  end

  def cook
    download unless downloaded?
    extract_zip
    compile
  end

  def compile
    execute('compile', ['mvn', '-P', '!truffle', "-Djruby.default.ruby.version=#{ruby_version}"])
  end

  def extract_zip
    files_hashs.each do |file|
      verify_file(file)

      filename = File.basename(file[:local_path])
      message "Unzipping #{filename} into #{tmp_path}... "
      FileUtils.mkdir_p tmp_path
      execute('unzip', ["unzip", "-o", file[:local_path], "-d", tmp_path], {:cd => Dir.pwd, :initial_message => false})
    end
  end

  def ruby_version
    @ruby_version ||= version.match(/.*-ruby-(\d+\.\d+)/)[1]
  end

  def jruby_version
    @jruby_version ||= version.match(/(.*)-ruby-\d+\.\d+/)[1]
  end
end
