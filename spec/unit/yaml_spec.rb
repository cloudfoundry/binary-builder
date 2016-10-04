# encoding: utf-8
require 'spec_helper'
require_relative '../../lib/yaml_presenter'

describe YAMLPresenter do
  it 'encodes the SHA256 as a raw string' do
    recipe = double(:recipe, files_hashs: [
                      {
                        local_path: File.expand_path(__FILE__)
                      }
                    ])
    presenter = described_class.new(recipe)
    expect(presenter.to_yaml).to_not include "!binary |-\n"
  end

  context 'the source is a github repo' do
    it 'displays the git commit sha' do
      recipe = double(:recipe, files_hashs: [
                        {
                          git: {commit_sha: 'a_mocked_commit_sha'},
                          local_path: File.expand_path(__FILE__)
                        }
                      ])
      presenter = described_class.new(recipe)
      expect(presenter.to_yaml).to_not include "!binary |-\n"
      expect(presenter.to_yaml).to include 'a_mocked_commit_sha'
    end
  end
end
