# frozen_string_literal: true

require 'English'
require_relative '../recipe/base'

class OpenSSLReplace
  def self.run(*args)
    system({ 'DEBIAN_FRONTEND' => 'noninteractive' }, *args)
    raise "Could not run #{args}" unless $CHILD_STATUS.success?
  end

  def self.replace_openssl
    file_base = 'OpenSSL_1_1_0g'
    file_name = "#{file_base}.tar.gz"
    openssl_tar = "https://github.com/openssl/openssl/archive/#{file_name}"

    Dir.mktmpdir do |_dir|
      run('wget', openssl_tar)
      run('tar', 'xf', file_name)
      Dir.chdir("openssl-#{file_base}") do
        run('./config',
            '--prefix=/usr',
            '--libdir=/lib/x86_64-linux-gnu',
            '--openssldir=/include/x86_64-linux-gnu/openssl')
        run('make')
        run('make', 'install')
      end
    end
  end
end
