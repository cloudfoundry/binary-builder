require 'mini_portile'
require 'tmpdir'

class BaseRecipe < MiniPortile
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

  def tar
    Dir.mktmpdir do |dir|
      archive_files.each do |glob|
        `cp -r #{glob} #{dir}`
      end
      system "ls -A #{dir} | xargs tar czf #{archive_filename} -C #{dir}"
    end
  end

  private

  # NOTE: https://www.virtualbox.org/ticket/10085
  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end
end

