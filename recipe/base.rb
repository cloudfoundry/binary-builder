require 'mini_portile'
require 'tmpdir'
require 'fileutils'

class BaseRecipe < MiniPortile
  def initialize(name, version, options = {})
    super name, version

    options.each do |key, value|
      instance_variable_set("@#{key}", value)
    end
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

  def cook
    super
    tar
  end

  def archive_files
    []
  end

  def archive_path_name
    ""
  end

  def tar
    return if archive_files.empty?

    Dir.mktmpdir do |dir|
      archive_path = File.join(dir, archive_path_name)
      FileUtils.mkdir_p(archive_path)

      archive_files.each do |glob|
        `cp -r #{glob} #{archive_path}`
      end

      execute('archiving', ['bash', '-c', "ls -A #{dir} | xargs tar czf #{archive_filename} -C #{dir}"], cd: Dir.pwd)
    end
  end

  private

  # NOTE: https://www.virtualbox.org/ticket/10085
  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end
end

