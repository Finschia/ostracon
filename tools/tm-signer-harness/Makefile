.PHONY: build install docker-image

OSTRACON_VERSION?=latest
BUILD_TAGS?='ostracon'
VERSION := $(shell git describe --always)
BUILD_FLAGS = -ldflags "-X github.com/line/ostracon/version.OCCoreSemVer=$(VERSION)"

.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -tags $(BUILD_TAGS) -o ../../build/tm-signer-harness main.go

install:
	CGO_ENABLED=0 go install $(BUILD_FLAGS) -tags $(BUILD_TAGS) .

docker-image:
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -tags $(BUILD_TAGS) -o tm-signer-harness main.go
	docker build --rm \
		--build-arg OSTRACON_VERSION=$(OSTRACON_VERSION) \
		-t ostracon/tm-signer-harness:$(OSTRACON_VERSION) .
	rm -rf tm-signer-harness
