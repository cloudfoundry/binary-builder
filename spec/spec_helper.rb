require 'open3'
require 'fileutils'

RSpec.configure do |config|
  if RUBY_PLATFORM.include?('darwin')
    DOCKER_CONTAINER_NAME = "test-suite-binary-builder-#{Time.now.to_i}"

    config.before(:all, :integration) do
      docker_image = 'cloudfoundry/cflinuxfs2'

      %x{docker run --name #{DOCKER_CONTAINER_NAME} -dit -v #{Dir.pwd}:/binary-builder -e CCACHE_DIR=/binary-builder/.ccache -w /binary-builder #{docker_image} sh -c 'env PATH=/usr/lib/ccache:$PATH bash'}
      `docker exec #{DOCKER_CONTAINER_NAME} apt-get -y install ccache`
      `docker exec #{DOCKER_CONTAINER_NAME} gem install bundler --no-ri --no-rdoc`
      `docker exec #{DOCKER_CONTAINER_NAME} bundle install -j4`
    end

    config.after(:all, :integration) do
      `docker stop #{DOCKER_CONTAINER_NAME}`
      `docker rm #{DOCKER_CONTAINER_NAME}`
    end
  end

  def run(cmd)
    cmd = "docker exec #{DOCKER_CONTAINER_NAME} #{cmd}" if RUBY_PLATFORM.include?('darwin')

    Bundler.with_clean_env do
      Open3.capture2e(cmd).tap do |output, status|
        expect(status).to be_success, (lambda do
          puts "command output: #{output}"
          puts "expected command to return a success status code, got: #{status}"
        end)
      end
    end
  end

  def run_binary_builder(binary_name, binary_version, flags)
    binary_builder_cmd = "bundle exec ./bin/binary-builder --name=#{binary_name} --version=#{binary_version} #{flags}"
    run(binary_builder_cmd)
  end
end
