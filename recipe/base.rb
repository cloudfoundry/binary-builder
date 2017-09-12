# encoding: utf-8
require 'mini_portile'
require 'tmpdir'
require 'fileutils'
require_relative 'determine_checksum'
require_relative '../lib/yaml_presenter'

class BaseRecipe < MiniPortile
  def initialize(name, version, options = {})
    super name, version

    options.each do |key, value|
      instance_variable_set("@#{key}", value)
    end

    @files = [{
      url: url
    }.merge(DetermineChecksum.new(options).to_h)]
  end

  def configure_options
    []
  end

  def compile
    execute('compile', [make_cmd, '-j4'])
  end

  def archive_filename
    "#{name}-#{version}-linux-x64.tgz"
  end

  def archive_files
    []
  end

  def archive_path_name
    ''
  end

  private

  # NOTE: https://www.virtualbox.org/ticket/10085
  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end

  def download_file(url, full_path, count = 3)
    super

    if @sources_export_dir
      target_filename = File.basename(full_path)
      target_filename.prepend("#{name}-#{}") unless target_filename.downcase.include?(name)

      FileUtils.mkdir_p @sources_export_dir
      FileUtils.cp full_path, File.join(@sources_export_dir, target_filename)
    end
  end
end
