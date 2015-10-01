require 'spec_helper'
require_relative '../../lib/yaml_presenter'

describe YAMLPresenter do
  it 'encodes the SHA256 as a raw string' do
    recipe = double(:recipe, files_hashs: [
      {
        local_path: File.expand_path(__FILE__)
      }
    ])
    presenter = YAMLPresenter.new(recipe)
    expect(presenter.to_yaml).to_not include "!binary |-\n"
  end
end
