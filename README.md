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

The scripts are meant to be run on a Cloud Foundry [stack](https://docs.cloudfoundry.org/concepts/stacks.html).

## Running within Docker

To run `binary-builder` from within the cflinuxfs2 rootfs, use [Docker](https://docker.io):

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2 bash
./bin/binary-builder --name=[binary_name] --version=[binary_version] --(md5|sha256)=[checksum_value]
```

This generates a gzipped tarball in the binary-builder directory with the filename format `binary_name-binary_version-linux-x64`.

For example, if you were building ruby 2.2.3, you'd run the following commands:

```bash
$ docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2:ruby-2.2.4 ./bin/binary-builder --name=ruby --version=2.2.3 --md5=150a5efc5f5d8a8011f30aa2594a7654
$ ls
ruby-2.2.3-linux-x64.tgz
```

# Building PHP

To build PHP, you also need to pass in a YAML file containing information about the various PHP extensions to be built. For example

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2 bash
./bin/binary-builder --name=php --version=5.6.14 --md5=ae625e0cfcfdacea3e7a70a075e47155 --php-extensions-file=./php-extensions.yml
```

For an example of what this file looks like, see: https://github.com/cloudfoundry/public-buildpacks-ci-robots/blob/master/binary-builds/php-extensions.yml

# Contributing

Find our guidelines [here](./CONTRIBUTING.md).

# Reporting Issues

Open an issue on this project

# Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)

# Running the tests

The integration test suite includes specs that test the functionality for building [PHP with Oracle client libraries](./PHP-Oracle.md). These tests are tagged `:run_oracle_php_tests` and require access to an S3 bucket containing the Oracle client libraries. This is configured using the environment variables `AWS_ACCESS_KEY` and `AWS_SECRET_ACCESS_KEY`

If you do not need to test this functionality, exclude the tag `:run_oracle_php_tests` when you run `rspec`.
