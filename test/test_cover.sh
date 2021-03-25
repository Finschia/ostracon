#! /bin/bash

PKGS=$(go list github.com/line/ostracon/...)

set -e

echo "mode: atomic" > coverage.txt
for pkg in ${PKGS[@]}; do
	go test -tags 'memdb goleveldb' -timeout 5m -race -coverprofile=profile.out -covermode=atomic "$pkg"
	if [ -f profile.out ]; then
		tail -n +2 profile.out >> coverage.txt;
		rm profile.out
	fi
done
