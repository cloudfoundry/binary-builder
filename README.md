## Binary Builder

This tool provides a mechanism for building binaries.

### Running within Docker

When building binaries for CloudFoundry, it may be useful to run `binary-builder` from within a CF rootfs. The cflinuxfs2 rootfs may be used as follows:

```bash
docker run -w /binary-builder -v `pwd`:/binary-builder -it cloudfoundry/cflinuxfs2 bash
./bin/binary-builder [binary_name] [binary_version]
```
