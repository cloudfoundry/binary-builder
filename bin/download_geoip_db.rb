#!/usr/bin/env ruby
# encoding: utf-8

require "net/http"
require "uri"
require "digest"
require "tempfile"
require "optparse"
require_relative "../lib/geoip_downloader"

options = {}
optparser = OptionParser.new do |opts|
    opts.banner = 'USAGE: download_geoip_db [options]'

    opts.on('-uUSER', '--user=USER', 'User Id from MaxMind.  Default "999999".') do |n|
        options[:user] = n
    end

    opts.on('-lLICENSE', '--license=LICENSE', 'License from MaxMind.  Default "000000000000".') do |n|
        options[:license] = n
    end

    opts.on('-oOUTPUTDIR', '--output_dir=OUTPUTDIR', 'Directory where databases might exist and will be written / updated.  Default "."') do |n|
        options[:output_dir] = n
    end

    opts.on('-pPRODUCTS', '--products=PRODUCTS', 'Space separated list of product ids.  Default "GeoLite-Legacy-IPv6-City GeoLite-Legacy-IPv6-Country 506 517 533".') do |n|
        options[:products] = n
    end
end
optparser.parse!

options[:user] ||= MaxMindGeoIpUpdater.FREE_USER
options[:license] ||= MaxMindGeoIpUpdater.FREE_LICENSE
options[:output_dir] ||= '.'
options[:products] ||= 'GeoLite-Legacy-IPv6-City GeoLite-Legacy-IPv6-Country 506 517 533'

updater = MaxMindGeoIpUpdater.new(options[:user], options[:license], options[:output_dir])

options[:products].split(" ").each do |product|
    updater.download_product(product)
end
