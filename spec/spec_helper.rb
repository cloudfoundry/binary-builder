require 'open3'
require 'blueprint/blueprints'

RSpec.configure do |config|
  def run_binary_builder(interpreter, tag, docker_image, flags = '')
    binary_builder_path = File.join(Dir.pwd, 'bin', 'binary-builder')
    Open3.capture2e("#{binary_builder_path} #{interpreter} #{tag} #{flags}")[0]
  end
end

