require 'mini_portile'

class BaseRecipe < MiniPortile
  def compile
    execute('compile', [make_cmd, '-j4'])
  end

  def cook
    super
    tar
  end
end

