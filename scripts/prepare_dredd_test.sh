#!/usr/bin/env bash

 echo "Install compile tools"
 apt-get update
 apt-get install -y make wget git gcc libc-dev

 VERSION=1.14.2
 OS=linux
 GOLANG_FULL=go${VERSION}.${OS}-amd64.tar.gz

 echo "Install golang"
 wget https://dl.google.com/go/${GOLANG_FULL}
 tar -C /usr/local -xzf ${GOLANG_FULL}
 export PATH=$PATH:/usr/local/go/bin

 echo "Build contract-tests"
 make build-contract-tests-hooks
