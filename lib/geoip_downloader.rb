# encoding: utf-8
require "net/http"
require "uri"
require "digest"
require "tempfile"

class MaxMindGeoIpUpdater
    def initialize(user_id, license, output_dir)
        @proto = 'http'
        @host = 'updates.maxmind.com'
        @user_id = user_id
        @license = license
        @output_dir = output_dir
        @client_ip = nil
        @challenge_digest = nil
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
                file = Tempfile.new('geiop_db_download')
                begin
                    if resp['content-type'] == 'text/plain; charset=utf-8'
                        puts "\talready up-to-date."
                    else
                        resp.read_body do |chunk|
                            file.write(chunk)
                        end
                        file.rewind
                        extract_file(file, file_path)
                        puts "\tdatabase updated."
                    end
                ensure
                    file.close()
                    file.unlink()
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
        puts "\tchallenge: #{challenge_digest}"
        download_database(db_digest, challenge_digest, product_id, file_path)
    end

    def db_digest(path)
        return File::exist?(path) ? Digest::MD5.file(path) : '00000000000000000000000000000000'
    end

    def challenge_digest
        return Digest::MD5.hexdigest("#{@license}#{client_ip}")
    end
end
