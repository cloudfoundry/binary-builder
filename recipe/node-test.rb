require 'mini_portile'
require 'fileutils'
require_relative 'node'

class NodeTestRecipe < NodeRecipe
  def install
    execute('test', [make_cmd, "test"])
  end
end

