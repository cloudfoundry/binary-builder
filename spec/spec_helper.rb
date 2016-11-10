# encoding: utf-8
require 'open3'
require 'fileutils'

RSpec.configure do |config|
  config.color = true
  config.tty = true

  if RUBY_PLATFORM.include?('darwin')
    DOCKER_CONTAINER_NAME = "test-suite-binary-builder-#{Time.now.to_i}".freeze

    config.before(:all, :integration) do
      directory_mapping = "-v #{Dir.pwd}:/binary-builder"
      setup_docker_container(DOCKER_CONTAINER_NAME, directory_mapping)
    end

    config.after(:all, :integration) do
      cleanup_docker_artifacts(DOCKER_CONTAINER_NAME)
    end

    config.before(:all, :run_oracle_php_tests) do
      dir_to_contain_oracle = File.join(Dir.pwd, 'oracle_client_libs')
      FileUtils.mkdir_p(dir_to_contain_oracle)
      setup_oracle_libs(dir_to_contain_oracle)

      oracle_dir = File.join(dir_to_contain_oracle, 'oracle')
      directory_mapping = "-v #{Dir.pwd}:/binary-builder -v #{oracle_dir}:/oracle"
      setup_docker_container(DOCKER_CONTAINER_NAME, directory_mapping)
    end

    config.after(:all, :run_oracle_php_tests) do
      cleanup_docker_artifacts(DOCKER_CONTAINER_NAME)

      dir_containing_oracle = File.join(Dir.pwd, 'oracle_client_libs')
      FileUtils.rm_rf(dir_containing_oracle)
    end
  else
    config.before(:all, :run_oracle_php_tests) do
      setup_oracle_libs('/')
    end
  end

  def cleanup_docker_artifacts(docker_container_name)
    `docker stop #{docker_container_name}`
    `docker rm #{docker_container_name}`

    Dir['*deb*'].each do |deb_file|
      FileUtils.rm(deb_file)
    end
  end

  def setup_oracle_libs(dir_to_contain_oracle)
    Dir.chdir(dir_to_contain_oracle) do
      s3_bucket = ENV['ORACLE_LIBS_AWS_BUCKET']
      libs_filename = ENV['ORACLE_LIBS_FILENAME']

      system "aws s3 cp s3://#{s3_bucket}/#{libs_filename} ."
      system "tar -xvf #{libs_filename}"
    end
  end

  def setup_docker_container(docker_container_name, directory_mapping)
    docker_image = 'cloudfoundry/cflinuxfs2'
    `docker run --name #{docker_container_name} -dit #{directory_mapping} -e CCACHE_DIR=/binary-builder/.ccache -w /binary-builder #{docker_image} sh -c 'env PATH=/usr/lib/ccache:$PATH bash'`
    `docker exec #{docker_container_name} apt-get -y install ccache`
    `docker exec #{docker_container_name} gem install bundler --no-ri --no-rdoc`
    `docker exec #{docker_container_name} bundle install -j4`
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

  def tar_contains_file(filename)
    system("tar --wildcards -tf #{@binary_tarball_location} #{filename} >/dev/null 2>&1")
  end
end
