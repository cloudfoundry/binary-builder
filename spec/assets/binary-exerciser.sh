#!/bin/sh
set +e

TAR_NAME=node-v0.12.2-cloudfoundry_cflinuxfs2.tgz
EXECUTABLE_PATH=node-v0.12.2-linux-x64/bin/node
EXECUTABLE_TEST='console.log(process.version)'

mkdir node-exerciser
cd node-exerciser

cp /binary-builder/${TAR_NAME} .
tar xzf ${TAR_NAME}
${EXECUTABLE_PATH} -e "${EXECUTABLE_TEST}"
