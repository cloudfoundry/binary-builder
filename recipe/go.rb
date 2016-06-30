# encoding: utf-8

class GoRecipe < MiniPortile
  def cook
    install
  end

  def install
    raise 'Require `apt-get` package manager' unless which('apt-get')
    FileUtils.mkdir_p tmp_path
    execute('install', [which('apt-get'), 'update'], cd: Dir.pwd)
    execute('install', [which('apt-get'), '-y', 'install', 'golang'], cd: Dir.pwd)
  end
end
