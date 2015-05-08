require 'spec_helper'
require 'blueprint/blueprints'

module BinaryBuilder
  describe 'binary-builder script' do
    context 'without valid arguments' do
      it 'prints out a helpful usage message if no arguments are provided' do
        expect(run_binary_builder('', '', '')).to include('USAGE', 'interpreter', 'git-tag', 'docker-image')
      end
    end
  end
end
