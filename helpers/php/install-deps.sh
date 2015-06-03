# update repo and packages
sudo apt-get update
sudo apt-get -y upgrade
sudo apt-get -y install build-essential autoconf automake libssl-dev libsnmp-dev snmp-mibs-downloader mercurial libbz2-dev libldap2-dev libpcre3-dev libxml2-dev libpq-dev libzip-dev libcurl4-openssl-dev libgdbm-dev libmysqlclient-dev libgmp-dev libjpeg-dev libpng12-dev libc-client2007e-dev libsasl2-dev libmcrypt-dev libaspell-dev libpspell-dev libexpat1-dev imagemagick libmagickwand-dev libmagickcore-dev unzip libmemcached-dev libicu-dev libsqlite3-dev libzookeeper-mt-dev libreadline-dev libxslt1-dev libyaml-dev

# Ubuntu 14.04 puts these headers in weird locations, need to add symlinks so PHP finds them
sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
