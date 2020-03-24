#!/bin/bash

# get the base commit hash of this branch. The branch should be brached from master.
base_commit=$(diff -u <(git rev-list --first-parent HEAD) \
													<(git rev-list --first-parent master) | \
													sed -ne 's/^ //p' | head -1)

if [ "$base_commit" ]; then
	#if base commit of this branch is exist, run linter from base commit to HEAD
	golangci-lint run -v --new-from-rev "$base_commit"
else
	# if not, check current don't commited files.
	golangci-lint run -n -v
fi
