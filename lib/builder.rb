require_relative 'architect/architects'
require 'fileutils'
require 'tmpdir'

module BinaryBuilder
  class Builder
    attr_reader :binary_name, :binary_version, :architect

    def self.build(options)
      builder = self.new(options)

      builder.set_foundation
      builder.install
      builder.tar_installed_binary
    end

    def initialize(options)
      @binary_name, @binary_version = options[:binary_name], options[:binary_version]
      @architect = architect_for_binary(binary_name).new(binary_version: @binary_version)
    end

    def set_foundation
      FileUtils.mkdir(installation_path)

      File.write(blueprint_path, architect.blueprint)
      FileUtils.chmod('+x', blueprint_path)
    end

    def install
      Dir.chdir(foundation_path) do
        run!("#{blueprint_path} #{installation_path}")
      end
    end

    def tar_installed_binary
      run!(tar_command)
    end

    private
    BINARY_ARCHITECT_MAP = {
      'node'   => NodeArchitect,
      'ruby'   => RubyArchitect,
      'jruby'  => JRubyArchitect,
      'python' => PythonArchitect
    }

    def architect_for_binary(binary)
      BINARY_ARCHITECT_MAP[binary]
    end

    def foundation_path
      @foundation_path ||= Dir.mktmpdir
    end

    def installation_path
      @installation_path ||= File.join(foundation_path, 'installation')
    end

    def blueprint_path
      @blueprint_path ||= File.join(foundation_path, 'blueprint.sh')
    end

    def tarball_name
      "#{binary_name}-#{binary_version}-linux-x64.tgz"
    end

    def tar_command
      "tar czf #{tarball_name} -C #{installation_path} ."
    end

    def run!(command)
      system(command) || raise("Failed to run command: #{command}")
    end
  end
end
