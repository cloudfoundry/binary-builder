require_relative '../recipe/base'

class OpenSSLReplace

  def self.run(*args)
    system({'DEBIAN_FRONTEND' => 'noninteractive'}, *args)
    raise "Could not run #{args}" unless $?.success?
  end

  def self.replace_openssl()
    filebase = 'OpenSSL_1_1_0g'
    filename = "#{filebase}.tar.gz"
    openssltar = "https://github.com/openssl/openssl/archive/#{filename}"

    Dir.mktmpdir do |dir|
      run('wget', openssltar)
      run('tar', 'xf', filename)
      Dir.chdir("openssl-#{filebase}") do
        run("./config",
                   "--prefix=/usr",
                   "--libdir=/lib/x86_64-linux-gnu",
                   "--openssldir=/include/x86_64-linux-gnu/openssl")
        run('make')
        run('make', 'install')
      end
    end
  end

end