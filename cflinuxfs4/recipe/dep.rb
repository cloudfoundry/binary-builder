# frozen_string_literal: true

require_relative 'base'

class DepRecipe < BaseRecipe
  attr_reader :name, :version

  def cook
    download unless downloaded?
    extract

    install_go_compiler

    FileUtils.rm_rf("#{tmp_path}/dep")
    FileUtils.mv(Dir.glob("#{tmp_path}/dep-*").first, "#{tmp_path}/dep")
    Dir.chdir("#{tmp_path}/dep") do
      system(
        { 'GOPATH' => "#{tmp_path}/dep/deps/_workspace:/tmp" },
        '/usr/local/go/bin/go get -asmflags -trimpath ./...'
      ) or raise 'Could not install dep'
    end
    FileUtils.mv("#{tmp_path}/dep/LICENSE", '/tmp/LICENSE')
  end

  def archive_files
    %w[/tmp/bin/dep /tmp/LICENSE]
  end

  def archive_path_name
    'bin'
  end

  def url
    "https://github.com/golang/dep/archive/#{version}.tar.gz"
  end

  def go_recipe
    @go_recipe ||= GoRecipe.new(@name, @version)
  end

  def tmp_path
    '/tmp/src/github.com/golang'
  end
end
