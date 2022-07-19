# PHP & Oracle Support

The binary builder has support for building the `oci8` and `pdo_oci` extensions of PHP 5.5, 5.6 and 7.0.  While the builder is capable of building them, it does *not* provide the Oracle libraries and SDK which are required to build these extensions.  These are not provided by binary-builder because of licensing restrictions and binary builder does not automatically download them because they are behind an Oracle pay-wall.  To build the extensions, you *must* download and provide these to the builder.

## What to Downwload

The requirements to build these extensions are listed here.

  http://php.net/manual/en/oci8.requirements.php

This basically boils down to installing the Oracle Instant Client Basic or Basic Lite plus the SDK.  Use the ZIP installs and extract them to a location on your local machine.  Then create the following symbolic library before building: `ln -s libclntsh.so.12.1 libclntsh.so` (version number must vary).

You only need to do this once.

## How to Build

To build, you just [follow the normal instructions for building PHP with binary builder & Docker](https://github.com/cloudfoundry/binary-builder/blob/master/README.md).  The only exception is that you need to map the path where you extracted the Oracle instant client and SDK to `/oracle` in the docker container used by binary builder.

This is done by adding an additiona `-v` argument to the `docker run` command.

Ex:

```
docker run -w /binary-builder -v `pwd`:/binary-builder -v /path/to/oracle:/oracle -it cloudfoundry/cflinuxfs3 bash
export STACK=cflinuxfs3 
./bin/binary-builder --name=php --version=7.1.29 --md5=ae625e0cfcfdacea3e7a70a075e47155 --php-extensions-file=./php71-extensions.yml
```

## What's Included

When you build PHP binaries with Oracle support you get the following included with the PHP binary:

1. The `oci8` extension (from PECL)
    - PHP 5.5 & 5.6 include oci8 2.0.x
    - PHP 7.0 includes oci 2.1.x
2. The `pdo_oci` extension bundled with PHP
3. The following libraries which are required by the extensions are include in `php/lib`
    - libclntshcore.so
    - libclntsh.so
    - libipc1.so
    - libmql1.so
    - libnnz12.so
    - libociicus.so
    - libons.so

Two notes on the included libraries:

1. The file `libociicus.so` is a US English specific library from Instant Client lite.  If you need multi-language support, you will need to install the full Instant Client and likely include additional libraries.  That's not supported at this time, but patches are welcome.

2. The PHP bundle that is built by binary builder has Oracle libraries from the Instant Client packaged with it.  As such, you should not publicly distribute these libraries unless you are licensed to do so by Oracle.

## Disabling Oracle Support

By default Oracle Support is *not* included.  The binary builder will not and cannot build these extensions unless you provide it with the Oracle libraries as listed in the *What to Download* section above.

If you want to build with and without Oracle support, you can control if binary-builder will include the Oracle support by adding or removing the volume map for `/oracle` on your `docker run` command.  If that volume is mounted then binary builder will attempt to build Oracle support.  If it's not mounted, Oracle support is disabled.

