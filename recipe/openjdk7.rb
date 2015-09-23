require 'mini_portile'

class OpenJDK7Recipe < MiniPortile
  def cook
    install
  end

  def install
    raise 'Require `apt-get` package manager' unless which('apt-get')
    FileUtils.mkdir_p tmp_path
    execute('install', [which('apt-get'), '-y', 'install', 'openjdk-7-jdk'], cd: Dir.pwd)
  end
end
