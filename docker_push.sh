#!/bin/bash

make build-docker
TM_VERSION="$(awk -F\" '/TMCoreSemVer *=/ {print $2; exit }' < ./version/version.go)"
LINE_VERSION="$(awk -F\" '/LINECoreSemVer *=/ {print $2; exit }' < ./version/version.go)"
DATE_VERSION="`date "+%y%m%d"`"
GIT_COMMIT="$(git rev-parse --short=8 HEAD)"
TAG=${TM_VERSION}_${LINE_VERSION}-${DATE_VERSION}-${GIT_COMMIT}
docker tag tendermint/tendermint:latest docker-registry.linecorp.com/link-network/tendermint:$TAG
echo "New tendermint version: $TAG"

read -p "==> Do you push docker image to repository as [$TAG]? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    docker push docker-registry.linecorp.com/link-network/tendermint:latest
    docker push docker-registry.linecorp.com/link-network/tendermint:$TAG
fi

