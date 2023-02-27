require 'net/http'

module HTTPHelper
  class << self
    def download_with_follow_redirects(uri)
      uri = URI(uri)
      Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == 'https') do |httpRequest|
        response = httpRequest.request_get(uri)
        if response.is_a?(Net::HTTPRedirection)
          download_with_follow_redirects(response['location'])
        else
          response
        end
      end
    end

    def download(uri, filename, digest_algorithm, sha)
      response = download_with_follow_redirects(uri)
      if response.code == '200'
        Sha.verify_digest(response.body, digest_algorithm, sha)
        File.write(filename, response.body)
      else
        str = "Failed to download #{uri} with code #{response.code} error: \n#{response.body}"
        raise str
      end
    end

    def read_file(url)
      uri = URI.parse(url)
      response = Net::HTTP.get_response(uri)
      response.body if response.code == '200'
    end
  end
end

module Sha
  class << self
    def verify_digest(content, algorithm, expected_digest)
      file_digest = get_digest(content, algorithm)
      raise "sha256 verification failed: expected #{expected_digest}, got #{file_digest}" if expected_digest != file_digest
    end

    def get_digest(content, algorithm)
      case algorithm
      when 'sha256'
        Digest::SHA2.new(256).hexdigest(content)
      when 'md5'
        Digest::MD5.hexdigest(content)
      when 'sha1'
        Digest::SHA1.hexdigest(content)
      else
        raise 'Unknown digest algorithm'
      end
    end
  end
end
