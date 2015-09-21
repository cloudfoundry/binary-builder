# update repo and packages
sudo apt-get update
sudo apt-get -y upgrade
sudo apt-get -y install \
    autoconf \
    automake \
    build-essential \
    imagemagick \
    libaspell-dev \
    libbz2-dev \
    libc-client2007e-dev \
    libcurl4-openssl-dev \
    libexpat1-dev \
    libgdbm-dev \
    libgmp-dev \
    libicu-dev \
    libjpeg-dev \
    libldap2-dev \
    libmagickcore-dev \
    libmagickwand-dev \
    libmcrypt-dev \
    libmemcached-dev \
    libmysqlclient-dev \
    libpcre3-dev \
    libpng12-dev \
    libpq-dev \
    libpspell-dev \
    libreadline-dev \
    libsasl2-dev \
    libsnmp-dev \
    libsqlite3-dev \
    libssl-dev \
    libxml2-dev \
    libxslt1-dev \
    libyaml-dev \
    libzip-dev \
    libzookeeper-mt-dev \
    mercurial \
    snmp-mibs-downloader \
    unzip

# Ubuntu 14.04 puts these headers in weird locations, need to add symlinks so PHP finds them
sudo ln -fs /usr/include/x86_64-linux-gnu/gmp.h /usr/include/gmp.h
sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap.so /usr/lib/libldap.so
sudo ln -fs /usr/lib/x86_64-linux-gnu/libldap_r.so /usr/lib/libldap_r.so
