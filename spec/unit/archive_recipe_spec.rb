# encoding: utf-8
require 'spec_helper'
require_relative '../../lib/archive_recipe'
require_relative '../../recipe/base'

describe ArchiveRecipe do
  class FakeRecipe < BaseRecipe
    def url; end

    def archive_files
      [1]
    end
  end

  context 'when the recipe has #setup_tar' do
    it 'it invokes' do
      recipe = FakeRecipe.new('fake', '1.1.1')
      def recipe.setup_tar; end
      allow(YAMLPresenter).to receive(:new).and_return('')

      expect(recipe).to receive(:setup_tar)
      described_class.new(recipe).compress!
    end
  end

  context 'when the recipe does not have #setup_tar' do
    it 'does not invoke' do
      recipe = FakeRecipe.new('fake', '1.1.1')
      allow(YAMLPresenter).to receive(:new).and_return('')

      expect do
        described_class.new(recipe).compress!
      end.not_to raise_error
    end
  end
end
