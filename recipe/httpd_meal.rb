class AprRecipe < BaseRecipe
  def configure_options
    []
  end

  def url
    "http://apache.mirrors.tds.net/apr/apr-#{version}.tar.gz"
  end
end

class AprIconvRecipe < BaseRecipe
  def initialize(name, version, options={})
    super name, version
    @apr_path = options[:apr_path]
  end

  def configure_options
    [
      "--with-apr=#{@apr_path}/bin/apr-1-config"
    ]
  end

  def url
    "http://apache.mirrors.tds.net/apr/apr-iconv-#{version}.tar.gz"
  end
end

class AprUtilRecipe < BaseRecipe
  def initialize(name, version, options={})
    super name, version
    @apr_path = options[:apr_path]
    @apr_iconv_path = options[:apr_iconv_path]
  end

  def configure_options
    [
      "--with-apr=#{@apr_path}",
      "--with-iconv=#{@apr_iconv_path}",
      "--with-crypto",
      "--with-openssl",
      "--with-mysql",
      "--with-pgsql",
      "--with-gdbm",
      "--with-ldap"
    ]
  end

  def url
    "http://apache.mirrors.tds.net/apr/apr-util-#{version}.tar.gz"
  end
end

class HTTPdRecipe < BaseRecipe
  def initialize(name, version, options={})
    super name, version
    @apr_path = options[:apr_path]
    @apr_iconv_path = options[:apr_iconv_path]
    @apr_util_path = options[:apr_util_path]
  end

  def configure_options
    [
      "--with-apr=#{@apr_path}" ,
      "--with-apr-util=#{@apr_util_path}" ,
      "--enable-mpms-shared=worker event" ,
      "--enable-mods-shared=reallyall" ,
      "--disable-isapi" ,
      "--disable-dav" ,
      "--disable-dialup"
    ]
  end

  def url
    "https://archive.apache.org/dist/httpd/httpd-#{version}.tar.bz2"
  end

  def archive_files
    [ File.join(path,"../httpd") ]
  end

  def tar
    system  <<-eof
      cd #{self.path}

      rm -rf build/ cgi-bin/ error/ icons/ include/ man/ manual/ htdocs/
      rm -rf conf/extra/* conf/httpd.conf conf/httpd.conf.bak conf/magic conf/original

      mkdir -p lib
      cp "#{@apr_path}/lib/libapr-1.so.0" ./lib
      cp "#{@apr_util_path}/lib/libaprutil-1.so.0" ./lib
      cp "#{@apr_iconv_path}/lib/libapriconv-1.so.0" ./lib
      cp -r "#{self.path}" ../httpd
    eof
    super
  end
end

class HTTPdMeal
  attr_accessor :files

  def initialize(name, version)
    @name    = name
    @version = version
    @files   = []
  end

  def cook
    apr_recipe.files << {
      url: apr_recipe.url,
      md5: '98492e965963f852ab29f9e61b2ad700'
    }
    apr_recipe.cook

    apr_iconv_recipe.files << {
      url: apr_iconv_recipe.url,
      md5: '4a27a1480e6862543396e59c4ffcdeb4'
    }
    apr_iconv_recipe.cook

    apr_util_recipe.files << {
      url: apr_util_recipe.url,
      md5: '866825c04da827c6e5f53daff5569f42'
    }
    apr_util_recipe.cook

    httpd_recipe.files = self.files
    httpd_recipe.cook

  end

  def url
    httpd_recipe.url
  end

  private

  def httpd_recipe
    @http_recipe ||= HTTPdRecipe.new(@name, @version,
                                     apr_path: apr_recipe.path,
                                     apr_util_path: apr_util_recipe.path,
                                     apr_iconv_path: apr_iconv_recipe.path
                                    )
  end

  def apr_util_recipe
    @apr_util_recipe ||= AprUtilRecipe.new('apr-util', '1.5.4', apr_path: apr_recipe.path, apr_iconv_path: apr_iconv_recipe.path)
  end

  def apr_iconv_recipe
    @apr_iconv_recipe ||= AprIconvRecipe.new('apr-iconv', '1.2.1', apr_path: apr_recipe.path)
  end

  def apr_recipe
    @apr_recipe ||= AprRecipe.new('apr', '1.5.2')
  end
end
