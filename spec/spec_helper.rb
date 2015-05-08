require 'open3'
require 'fileutils'

RSpec.configure do |config|
  def run_binary_builder(binary_name, tag, docker_image, flags = '')
    binary_builder_path = File.join(Dir.pwd, 'bin', 'binary-builder')
    Open3.capture2e("#{binary_builder_path} #{binary_name} #{tag} #{docker_image} #{flags}")[0]
  end
end

