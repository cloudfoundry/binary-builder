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

# Usage

The scripts are meant to be run on a Cloud Foundry [stack](https://docs.cloudfoundry.org/concepts/stacks.html).

## Running within Docker

To run `binary-builder` from within the cflinuxfs2 rootfs, use [Docker](https://docker.io):

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2 bash
./bin/binary-builder [binary_name] [binary_version] [checksum_value(MD5|SHA256)]
```

This generates a gzipped tarball in the binary-builder directory with the filename format `binary_name-binary_version-linux-x64`.

For example, if you were building nginx 1.7.11, you'd run the following commands:

```bash
$ docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2 ./bin/binary-builder ruby 2.2.3 150a5efc5f5d8a8011f30aa2594a7654 
$ ls
ruby-2.2.3-linux-x64.tgz
```

# Contributing

Find our guidelines [here](./CONTRIBUTING.md).

# Reporting Issues

Open an issue on this project

# Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)
