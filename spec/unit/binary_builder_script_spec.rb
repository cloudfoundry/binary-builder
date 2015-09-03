require 'spec_helper'

module BinaryBuilder
  describe 'binary-builder script' do
    context 'without valid arguments' do
      it 'prints out a helpful usage message if no arguments are provided' do
        expect(`#{File.join('./bin', 'binary-builder')} 2>&1`).to include('USAGE', 'binary')
      end
    end
  end
end
