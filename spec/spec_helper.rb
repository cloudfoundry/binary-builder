require 'open3'
require 'fileutils'
require 'builder'

RSpec.configure do |config|
  if RUBY_PLATFORM.include?('darwin')
    DOCKER_CONTAINER_NAME = "test-suite-binary-builder-#{Time.now.to_i}"

    config.before(:all, :integration) do
      docker_image = 'cloudfoundry/cflinuxfs2'

      %x{docker run --name #{DOCKER_CONTAINER_NAME} -dit -v #{Dir.pwd}:/binary-builder -w /binary-builder #{docker_image} bash}
      `docker exec #{DOCKER_CONTAINER_NAME} gem install bundler`
      `docker exec #{DOCKER_CONTAINER_NAME} bundle install -j4`
    end

    config.after(:all, :integration) do
      `docker stop #{DOCKER_CONTAINER_NAME}`
    end

    def run(cmd)
      docker_cmd = "docker exec #{DOCKER_CONTAINER_NAME} #{cmd}"
      output, status = Open3.capture2e(docker_cmd)
    end
  else
    def run(cmd)
      output, status = Open3.capture2e(cmd)
    end
  end

  def run_binary_builder(binary_name, binary_version, flags = '')
    binary_builder_cmd = "#{File.join('./bin', 'binary-builder')} #{binary_name} #{binary_version} #{flags}"
    run(binary_builder_cmd)[0]
  end

end
