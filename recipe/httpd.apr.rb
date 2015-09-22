require 'mini_portile'

class HttpdAprRecipe < MiniPortile

  def configure_options
    []
  end

  def port_path
    "#{@staging_dir}/libapr-#{version}"
  end

  def tmp_path
    "/tmp/#{@host}/ports/#{@name}/#{@version}"
  end

  def initialize name, version, staging_dir
    super name, version, {}
    @staging_dir = staging_dir
    @files = [{ url: "http://apache.mirrors.tds.net/apr/apr-#{version}.tar.gz" }]
  end
end

