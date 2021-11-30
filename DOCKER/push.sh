#!/usr/bin/env bash
set -e

if [ -z "$TAG" ]; then
	echo "Please specify a tag."
	exit 1
fi

TAG_NO_PATCH=${TAG%.*}

read -p "==> Push 3 docker images with the following tags (latest, $TAG, $TAG_NO_PATCH)? y/n" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
	docker push "ostracon/ostracon:latest"
	docker push "ostracon/ostracon:$TAG"
	docker push "ostracon/ostracon:$TAG_NO_PATCH"
fi
