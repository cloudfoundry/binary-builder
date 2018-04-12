# PHP & GeoIP Support

The binary builder has support for building the `geoip` extension for PHP 5.5, 5.6 and 7.0.  In order for the `geoip` extension to function properly though it requires a database of geoip data.  The company MaxMind provides both commercial and less accurate open source versions of these databases.

In the default mode, binary-builder will not bundle any database files with PHP, however it will bundle a script that can be run to download an up-to-date version of the open source (also called "lite") databases.

If you would like binary-builder to download the open source version of the databases and bundle them with PHP, it can do that too.  To instruct binary-builder to do that, create a file called `BUNDLE_GEOIP_LITE` in the top level of the project and set the contents of the file to `true`.  When binary-builder sees that this file exists and that the contents are `true` it will download and bundle the open source version of the databases with the resulting PHP binary.

Right now binary-builder can't be configured to download and bundle commercial versions of the geoip data from MaxMind.  You can download commercial versions via the included script (see below), but there is no option exposed to configure the script through binary-builder.

## What's Included

When you build PHP binaries with GeoIP support you get the following included with the PHP binary:

1. The `geoip` extension (from PECL)
2. The library file `geoipdb/lib/geoip_downloader.rb` and script `geoipdb/bin/download_geoip_db.rb` which can be run at a later date to download a copy of the geoip databases from MaxMind.  By default they will download the "lite" or open source versions but the script can be configured to use a user id & license key to download paid versions as well.
3. If bundled, the geoip databases will be under `geoipdb/dbs`.  If not bundled, that directory will be empty.  If bundled, the following databases will be included:  `GeoLiteCityv6.dat`, `GeoLiteASNum.dat`, `GeoLiteCountry.dat`, `GeoIPv6.dat`, and `GeoLiteCity.dat`.
