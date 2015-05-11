require 'open3'
require 'fileutils'
require 'builder'

RSpec.configure do |config|
  def run_binary_builder(binary_name, tag, docker_image, flags = '')
    boot2docker_shellinit_cmd = '$(boot2docker shellinit)'
    docker_run_cmd = "docker run -i -v #{Dir.pwd}:/binary-builder cloudfoundry/cflinuxfs2 bash -c"
    binary_builder_cmd = "cd /binary-builder; #{File.join('./bin', 'binary-builder')} #{binary_name} #{tag} #{flags}"
    Open3.capture2e("#{boot2docker_shellinit_cmd} && #{docker_run_cmd} '#{binary_builder_cmd}'")[0]
  end
end

