#!/bin/bash
 
make build-docker
LINE_VERSION="$(awk -F\" '/LINECoreSemVer =/ {print $2; exit }' < ./version/version.go)"
GIT_COMMIT="$(git rev-parse --short=8 HEAD)"
TAG=docker-registry.linecorp.com/link-network/tendermint:latest
docker tag tendermint/tendermint:latest $TAG
 
read -p "==> Do you push docker image to [$TAG]? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    docker push $TAG
fi

