# Introduction

This tool provides a mechanism for building binaries for the Cloud Foundry buildpacks.

## Currently supported binaries

* NodeJS
* Ruby
* JRuby
* Python
* PHP
* Nginx
* Apache HTTPD Server
* Go
* Glide
* Godep
* Bundler

# Usage

The scripts are meant to be run as root on a Cloud Foundry [stack](https://docs.cloudfoundry.org/concepts/stacks.html).

## Running within Docker

To run `binary-builder` from within the cflinuxfs3 rootfs, use [Docker](https://docker.io):

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs3 bash
export STACK=cflinuxfs3
./bin/binary-builder --name=[binary_name] --version=[binary_version] --(md5|sha256)=[checksum_value]
```

This generates a gzipped tarball in the binary-builder directory with the filename format `binary_name-binary_version-linux-x64`.

For example, if you were building ruby 2.2.3, you'd run the following commands:

```bash
$ docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs3:ruby-2.2.4 ./bin/binary-builder --name=ruby --version=2.2.3 --md5=150a5efc5f5d8a8011f30aa2594a7654
$ ls
ruby-2.2.3-linux-x64.tgz
```

# Building PHP

To build PHP, you also need to pass in a YAML file containing information about the various PHP extensions to be built. For example

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs3 bash
export STACK=cflinuxfs3
./bin/binary-builder --name=php7 --version=7.3.14 --sha256=6aff532a380b0f30c9e295b67dc91d023fee3b0ae14b4771468bf5dda4cbf108 --php-extensions-file=./php7-extensions.yml
```

For an example of what this file looks like, see: [php7-base-extensions.yml](https://github.com/cloudfoundry/buildpacks-ci/tree/master/tasks/build-binary-new) and the various `php*-extensions-patch.yml` files in that same directory. Patch files adjust the base-extensions.yml file by adding/removing extensions. The `--php-extensions-file` argument will need the base-extensions file with one of the patch files applied. That normally happens automatically through the pipeline, so if you are building manually you need to manually create this file.

**TIP** If you are updating or building a specific PHP extension, remove everything except that specific extension from your `--php-extensions-file` file. This will decrease the build times & make it easier for you to test your changes.

# Building nginx

Nginx uses GPG keys to verify the source tarball, so you'll need something like the following code to build the NGinx binary:

```
version=1.15.9
gpg_signature_url="http://nginx.org/download/nginx-${version}.tar.gz.asc"
gpg_signature=`curl -sL ${gpg_signature_url}`

docker run -w /binary-builder -v `pwd`:/binary-builder \
  -it cloudfoundry/cflinuxfs3 ./bin/binary-builder \
  --name=nginx-static --gpg-rsa-key-id=A1C052F8 \
  --version=$version --gpg-signature="${gpg_signature}"
```

# Contributing

Find our guidelines [here](./CONTRIBUTING.md).

# Reporting Issues

Open an issue on this project

# Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)

# Running the tests

The integration test suite includes specs that test the functionality for building [PHP with Oracle client libraries](./PHP-Oracle.md). These tests are tagged `:run_oracle_php_tests` and require access to an S3 bucket containing the Oracle client libraries. This is configured using the environment variables `AWS_ACCESS_KEY` and `AWS_SECRET_ACCESS_KEY`

If you do not need to test this functionality, exclude the tag `:run_oracle_php_tests` when you run `rspec`.
