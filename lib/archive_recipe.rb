require 'tmpdir'
require_relative 'yaml_presenter'

class ArchiveRecipe
  def initialize(recipe)
    @recipe = recipe
  end

  def tar!
    return if @recipe.archive_files.empty?

    @recipe.setup_tar if @recipe.respond_to? :setup_tar

    Dir.mktmpdir do |dir|
      archive_path = File.join(dir, @recipe.archive_path_name)
      FileUtils.mkdir_p(archive_path)

      @recipe.archive_files.each do |glob|
        `cp -r #{glob} #{archive_path}`
      end

      File.write("#{dir}/sources.yml", YAMLPresenter.new(@recipe).to_yaml)

      print "Running 'archive' for #{@recipe.name} #{@recipe.version}... "
      `ls -A #{dir} | xargs tar czf #{@recipe.archive_filename} -C #{dir}`
      puts "OK"
    end
  end
end
