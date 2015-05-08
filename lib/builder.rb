require_relative 'architect/architects'
require 'fileutils'

module BinaryBuilder
  class Builder
    attr_reader :binary_name, :git_tag, :docker_image, :architect

    def initialize(binary_name:, git_tag:, docker_image:)
      @architect = architect_for_binary(binary_name).new(git_tag: git_tag)
      @binary_name, @git_tag, @docker_image = binary_name, git_tag, docker_image
    end

    def set_foundation
      FileUtils.rm_rf(foundation_path) if Dir.exists?(foundation_path)
      FileUtils.mkdir_p(foundation_path)

      File.write(blueprint_path, architect.blueprint)
    end

    def install_via_docker
      run!(docker_command)
    end

    def tar_installed_binary
      FileUtils.rm(blueprint_path)
      run!(tar_command)
      FileUtils.mv(File.join(foundation_path, tarball_name), Dir.pwd)
      FileUtils.rm_rf(foundation_path)
    end

    private
    BINARY_ARCHITECT_MAP = {
      'node' => NodeArchitect
    }

    def architect_for_binary(binary)
      BINARY_ARCHITECT_MAP[binary]
    end

    def foundation_path
      @foundation_path ||= File.join(ENV['HOME'], '.binary-builder', "#{binary_name}-#{git_tag}-foundation")
    end

    def blueprint_path
      @blueprint_path ||= File.join(foundation_path, 'blueprint.sh')
    end

    def tarball_name
      "#{binary_name}-#{git_tag}-#{docker_image.gsub('/', '_')}.tgz"
    end

    def docker_command
      "docker run -v #{foundation_path}:/binary-builder #{docker_image} bash /binary-builder/blueprint.sh"
    end

    def tar_command
      "cd #{foundation_path} && tar czf #{tarball_name} ."
    end

    def run!(command)
      system(command)
    end
  end
end
