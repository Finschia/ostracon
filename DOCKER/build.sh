#!/usr/bin/env bash
set -e

if [ -z "$TAG" ]; then
	echo "Please specify a tag."
	exit 1
fi

TAG_NO_PATCH=${TAG%.*}

read -p "==> Build 3 docker images with the following tags (latest, $TAG, $TAG_NO_PATCH)? y/n" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
	docker build -t "ostracon/ostracon" -t "ostracon/ostracon:$TAG" -t "ostracon/ostracon:$TAG_NO_PATCH" -f ./Dockerfile ../
fi
