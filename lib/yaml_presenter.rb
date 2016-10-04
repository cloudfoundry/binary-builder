# encoding: utf-8
require 'yaml'
require 'digest'

class YAMLPresenter
  def initialize(recipe)
    @recipe = recipe
  end

  def to_yaml
    @recipe.send(:files_hashs).collect do |file|
      if file.has_key?(:git)
      {
        'url'    => file[:url],
        'git_commit_sha' => file[:git][:commit_sha]
      }
      else
        {
          'url'    => file[:url],
          'sha256' => Digest::SHA256.file(file[:local_path]).hexdigest.force_encoding('UTF-8')
        }
      end
    end.to_yaml
  end
end
