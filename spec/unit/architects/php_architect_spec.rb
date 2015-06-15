require 'spec_helper'

module BinaryBuilder
  describe PHPArchitect do
    subject(:architect) { PHPArchitect.new(binary_version: binary_version) }

    context 'when building any php interpreter and modules' do
      let(:binary_version) { '5.6.9' }

      describe 'blueprint' do
        it 'adds the BINARY_VERSION value' do
          expect(architect.blueprint).to include 'PHP_VERSION=5.6.9'
        end

        it 'adds the ZTS_VERSION value' do
          expect(architect.blueprint).to include 'ZTS_VERSION=20131226'
        end

        it 'adds the RABBITMQ_C_VERSION value' do
          expect(architect.blueprint).to include 'RABBITMQ_C_VERSION=0.5.2'
        end

        it 'adds the HIREDIS_VERSION value' do
          expect(architect.blueprint).to include 'HIREDIS_VERSION=0.11.0'
        end

        it 'adds the LUA_VERSION value' do
          expect(architect.blueprint).to include 'LUA_VERSION=5.2.4'
        end

        it 'adds the module version values' do
          expect(architect.blueprint).to include %q{MODULES[amqp]="1.4.0"}
        end

        it 'packages third party dependencies for the imap extension' do
          expect(architect.blueprint).to include 'package_php_extension "libc-client.so.2007e"'
        end

        it 'packages third party dependencies for the mcrypt extension' do
          expect(architect.blueprint).to include 'package_php_extension "libmcrypt.so.4"'
        end

        it 'packages third party dependencies for the pspell extension' do
          expect(architect.blueprint).to include 'package_php_extension "libaspell.so.15" "libpspell.so.15"'
        end

        it 'packages third party dependencies for the amqp extension' do
          expect(architect.blueprint).to include 'package_php_extension "$APP_DIR/librmq-$RABBITMQ_C_VERSION/lib/librabbitmq.so.1"'
        end

        it 'packages third party dependencies for the intl extension' do
          expect(architect.blueprint).to include 'package_php_extension "libicui18n.so.52" "libicuuc.so.52" "libicudata.so.52" "libicuio.so.52"'
        end

        it 'packages third party dependencies for the memcached extension' do
          expect(architect.blueprint).to include 'package_php_extension "libmemcached.so.10"'
        end

        it 'packages third party dependencies for the phpiredis extension' do
          expect(architect.blueprint).to include 'package_php_extension "$APP_DIR/librmq-$RABBITMQ_C_VERSION/lib/librabbitmq.so.1"'
        end
      end
    end
  end
end
