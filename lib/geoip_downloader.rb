# encoding: utf-8
require "net/http"
require "uri"
require "digest"
require "tempfile"

class MaxMindGeoIpUpdater
    @@FREE_LICENSE = '000000000000'
    @@FREE_USER = '999999'

    def initialize(user_id, license, output_dir)
        @proto = 'http'
        @host = 'updates.maxmind.com'
        @user_id = user_id
        @license = license
        @output_dir = output_dir
        @client_ip = nil
        @challenge_digest = nil
    end

    def self.FREE_LICENSE
        @@FREE_LICENSE
    end

    def self.FREE_USER
        @@FREE_USER
    end

    def get_filename(product_id)
        uri = URI.parse("#{@proto}://#{@host}/app/update_getfilename")
        uri.query = URI.encode_www_form({ :product_id => product_id})
        resp = Net::HTTP.get_response(uri)
        resp.body
    end

    def client_ip
        @client_ip ||= begin
            uri = URI.parse("#{@proto}://#{@host}/app/update_getipaddr")
            resp = Net::HTTP.get_response(uri)
            resp.body
        end
    end

    def download_database(db_digest, challenge_digest, product_id, file_path)
        uri = URI.parse("#{@proto}://#{@host}/app/update_secure")
        uri.query = URI.encode_www_form({
            :db_md5 => db_digest,
            :challenge_md5 => challenge_digest,
            :user_id => @user_id,
            :edition_id => product_id
        })

        Net::HTTP.start(uri.host, uri.port) do |http|
            req = Net::HTTP::Get.new(uri.request_uri)

            http.request(req) do |resp|
                file = Tempfile.new('geoip_db_download')
                begin
                    if resp['content-type'] == 'text/plain; charset=utf-8'
                        puts "\tAlready up-to-date."
                    else
                        resp.read_body do |chunk|
                            file.write(chunk)
                        end
                        file.rewind
                        extract_file(file, file_path)
                        puts "\tDatabase updated."
                    end
                ensure
                    file.close()
                    file.unlink()
                end
            end
        end
    end

    def download_free_database(product_id, file_path)
        product_uris = {
            "GeoLite-Legacy-IPv6-City" => "http://geolite.maxmind.com/download/geoip/database/GeoLiteCityv6-beta/GeoLiteCityv6.dat.gz",
            "GeoLite-Legacy-IPv6-Country" => "http://geolite.maxmind.com/download/geoip/database/GeoIPv6.dat.gz"
        }

        if !product_uris.include?(product_id)
            puts "\tProduct '#{product_id}' is not available under free license. Available products are: #{product_uris.keys().join(', ')}."
        else
            uri = URI.parse(product_uris[product_id])
            Net::HTTP.start(uri.host, uri.port) do |http|
                req = Net::HTTP::Get.new(uri.request_uri)

                http.request(req) do |resp|
                    file = Tempfile.new('geoip_db_download')
                    begin
                        resp.read_body do |chunk|
                            file.write(chunk)
                        end
                        file.rewind
                        extract_file(file, file_path)
                        puts "\tDatabase updated."
                    ensure
                        file.close()
                        file.unlink()
                    end
                end
            end
        end
    end

    def extract_file(file, file_path)
        gz = Zlib::GzipReader.new(file)
        begin
            File.open(file_path, 'w') do |out|
                IO.copy_stream(gz, out)
            end
        ensure
            gz.close
        end
    end

    def download_product(product_id)
        puts "Downloading..."
        file_name = get_filename(product_id)
        file_path = File.join(@output_dir, file_name)
        db_digest = db_digest(file_path)
        puts "\tproduct_id: #{product_id}"
        puts "\tfile_name: #{file_name}"
        puts "\tip: #{client_ip}"
        puts "\tdb: #{db_digest}"

        if @license == @@FREE_LICENSE
            # As of April 1, 2018, free legacy databases are no longer available through the GeoIP update
            # API. Therefore, we'll fetch them from static URLs they've provided.
            # This will NOT work using free GeoIP2 databases.
            download_free_database(product_id, file_path)
        else
            puts "\tchallenge: #{challenge_digest}"
            download_database(db_digest, challenge_digest, product_id, file_path)
        end
    end

    def db_digest(path)
        return File::exist?(path) ? Digest::MD5.file(path) : '00000000000000000000000000000000'
    end

    def challenge_digest
        return Digest::MD5.hexdigest("#{@license}#{client_ip}")
    end
end
