require 'open3'
require 'fileutils'
require 'builder'

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

  def run(cmd, log = false)
    cmd = "docker exec #{DOCKER_CONTAINER_NAME} #{cmd}" if RUBY_PLATFORM.include?('darwin')

    return exec_with_logs(cmd) if log
    Bundler.with_clean_env { Open3.capture2e(cmd) }
  end

  def run_binary_builder(binary_name, binary_version, checksum)
    binary_builder_cmd = "#{File.join('./bin', 'binary-builder')} #{binary_name} #{binary_version} #{checksum}"
    run(binary_builder_cmd, true)
  end

  private
  def exec_with_logs(cmd)
    cmd = "#{cmd} 2>&1"
    output = ''
    FileUtils.mkdir_p('logs')

    IO.popen(cmd) do |io|
      file_location = File.join(Dir.pwd, 'logs', "build-#{Time.now.strftime('%Y%m%d%H%M%S')}.log")

      puts "Writing output from `#{cmd}` to #{file_location}"
      File.open(file_location, 'w') do |f|
        while line = io.gets
          f.write(line)
          output << line
        end
      end
    end

    [output, $?]
  end
end
