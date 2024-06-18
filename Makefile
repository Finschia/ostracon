PACKAGES=$(shell go list ./...)
SRCPATH=$(shell pwd)
OUTPUT?=build/ostracon

INCLUDE = -I=${GOPATH}/src/github.com/Finschia/ostracon -I=${GOPATH}/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf
BUILD_TAGS ?= ostracon
VERSION := $(shell git describe --always)
LD_FLAGS = -X github.com/Finschia/ostracon/version.OCCoreSemVer=$(VERSION)
BUILD_FLAGS = -mod=readonly -ldflags "$(LD_FLAGS)"
HTTPS_GIT := https://github.com/Finschia/ostracon.git
CGO_ENABLED ?= 0
TARGET_OS ?= $(shell go env GOOS)
TARGET_ARCH ?= $(shell go env GOARCH)

# handle nostrip
ifeq (,$(findstring nostrip,$(OSTRACON_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
  LD_FLAGS += -s -w
endif

# handle race
ifeq (race,$(findstring race,$(OSTRACON_BUILD_OPTIONS)))
  BUILD_FLAGS += -race
endif

# handle cleveldb
ifeq (cleveldb,$(findstring cleveldb,$(OSTRACON_BUILD_OPTIONS)))
  BUILD_TAGS += cleveldb
endif

# handle badgerdb
ifeq (badgerdb,$(findstring badgerdb,$(OSTRACON_BUILD_OPTIONS)))
  BUILD_TAGS += badgerdb
endif

# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(OSTRACON_BUILD_OPTIONS)))
  CGO_ENABLED=1
  BUILD_TAGS += rocksdb
endif

# handle boltdb
ifeq (boltdb,$(findstring boltdb,$(OSTRACON_BUILD_OPTIONS)))
  BUILD_TAGS += boltdb
endif

# allow users to pass additional flags via the conventional LDFLAGS variable
LD_FLAGS += $(LDFLAGS)

all: build test install
.PHONY: all

include tests.mk

###############################################################################
###                                Build Ostracon                           ###
###############################################################################

build:
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -tags "$(BUILD_TAGS)" -o $(OUTPUT) ./cmd/ostracon/
.PHONY: build

install:
	CGO_ENABLED=1 go install $(BUILD_FLAGS) -tags "$(BUILD_TAGS)" ./cmd/ostracon
.PHONY: install

###############################################################################
###                                Mocks                                    ###
###############################################################################

mockery:
	go generate -run="./scripts/mockery_generate.sh" ./...
.PHONY: mockery

###############################################################################
###                                Protobuf                                 ###
###############################################################################

check-proto-deps:
ifeq (,$(shell which protoc-gen-gogofaster))
	@go install github.com/gogo/protobuf/protoc-gen-gogofaster@latest
endif
.PHONY: check-proto-deps

check-proto-format-deps:
ifeq (,$(shell which clang-format))
	$(error "clang-format is required for Protobuf formatting. See instructions for your platform on how to install it.")
endif
.PHONY: check-proto-format-deps

proto-gen: check-proto-deps
	@echo "Generating Protobuf files"
	@go run github.com/bufbuild/buf/cmd/buf generate
	@mv ./proto/ostracon/abci/types.pb.go ./abci/types/
	@mv ./proto/ostracon/rpc/grpc/types.pb.go ./rpc/grpc/
	@rm -rf ./proto/tendermint
.PHONY: proto-gen

# These targets are provided for convenience and are intended for local
# execution only.
proto-lint: check-proto-deps
	@echo "Linting Protobuf files"
	@go run github.com/bufbuild/buf/cmd/buf lint
.PHONY: proto-lint

proto-format: check-proto-format-deps
	@echo "Formatting Protobuf files"
	@find . -name '*.proto' -path "./proto/*" -exec clang-format -i {} \;
.PHONY: proto-format

proto-check-breaking: check-proto-deps
	@echo "Checking for breaking changes in Protobuf files against local branch"
	@echo "Note: This is only useful if your changes have not yet been committed."
	@echo "      Otherwise read up on buf's \"breaking\" command usage:"
	@echo "      https://docs.buf.build/breaking/usage"
	@go run github.com/bufbuild/buf/cmd/buf breaking --against ".git"
.PHONY: proto-check-breaking

proto-check-breaking-ci:
	@go run github.com/bufbuild/buf/cmd/buf breaking --against ".git"#branch=main
.PHONY: proto-check-breaking-ci

###############################################################################
###                              Build ABCI                                 ###
###############################################################################

build_abci:
	@go build -mod=readonly -i ./abci/cmd/...
.PHONY: build_abci

install_abci:
	@go install -mod=readonly ./abci/cmd/...
.PHONY: install_abci

###############################################################################
###                              Distribution                               ###
###############################################################################

########################################
### Distribution

# dist builds binaries for all platforms and packages them for distribution
# TODO add abci to these scripts
dist:
	@BUILD_TAGS=$(BUILD_TAGS) sh -c "'$(CURDIR)/scripts/dist.sh'"
.PHONY: dist

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download
.PHONY: go-mod-cache

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify
	@go mod tidy

draw_deps:
	@# requires brew install graphviz or apt-get install graphviz
	go get github.com/RobotsAndPencils/goviz
	@goviz -i github.com/Finschia/ostracon/cmd/ostracon -d 3 | dot -Tpng -o dependency-graph.png
.PHONY: draw_deps

get_deps_bin_size:
	@# Copy of build recipe with additional flags to perform binary size analysis
	$(eval $(shell go build -work -a $(BUILD_FLAGS) -tags $(BUILD_TAGS) -o $(OUTPUT) ./cmd/ostracon/ 2>&1))
	@find $(WORK) -type f -name "*.a" | xargs -I{} du -hxs "{}" | sort -rh | sed -e s:${WORK}/::g > deps_bin_size.log
	@echo "Results can be found here: $(CURDIR)/deps_bin_size.log"
.PHONY: get_deps_bin_size

###############################################################################
###                                  Libs                                   ###
###############################################################################

# generates certificates for TLS testing in remotedb and RPC server
gen_certs: clean_certs
	certstrap init --common-name "finschia.org" --passphrase ""
	certstrap request-cert --common-name "server" -ip "127.0.0.1" --passphrase ""
	certstrap sign "server" --CA "finschia.org" --passphrase ""
	mv out/server.crt rpc/jsonrpc/server/test.crt
	mv out/server.key rpc/jsonrpc/server/test.key
	rm -rf out
.PHONY: gen_certs

# deletes generated certificates
clean_certs:
	rm -f rpc/jsonrpc/server/test.crt
	rm -f rpc/jsonrpc/server/test.key
.PHONY: clean_certs

###############################################################################
###                  Formatting, linting, and vetting                       ###
###############################################################################

format:
	find . -name '*.go' -type f -not -path "*.git*" -not -name '*.pb.go' -not -name '*pb_test.go' | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "*.git*"  -not -name '*.pb.go' -not -name '*pb_test.go' | xargs goimports -w -local github.com/Finschia/ostracon
.PHONY: format

lint:
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run
.PHONY: lint

DESTINATION = ./index.html.md

###############################################################################
###                           Documentation                                 ###
###############################################################################

BRANCH := $(shell git branch --show-current)
BRANCH_URI := $(shell git branch --show-current | sed 's/[\#]/%23/g')
build-docs:
	@cd docs && \
	npm install && \
	VUEPRESS_BASE="/$(BRANCH_URI)/" npm run build && \
	mkdir -p ~/output/$(BRANCH) && \
	cp -r .vuepress/dist/* ~/output/$(BRANCH)/ && \
	for f in `find . -name '*.png' | grep -v '/node_modules/'`; do if [ ! -e `dirname $$f` ]; then mkdir `dirname $$f`; fi; cp $$f ~/output/$(BRANCH)/`dirname $$f`; done && \
	echo '<html><head><meta http-equiv="refresh" content="0;/$(BRANCH_URI)/index.html"/></head></html>' > ~/output/index.html
.PHONY: build-docs

sync-docs:
	cd ~/output && \
	echo "role_arn = ${DEPLOYMENT_ROLE_ARN}" >> /root/.aws/config ; \
	echo "CI job = ${CIRCLE_BUILD_URL}" >> version.html ; \
	aws s3 sync . s3://${WEBSITE_BUCKET} --profile terraform --delete ; \
	aws cloudfront create-invalidation --distribution-id ${CF_DISTRIBUTION_ID} --profile terraform --path "/*" ;
.PHONY: sync-docs

###############################################################################
###                            Docker image                                 ###
###############################################################################

# Build linux binary on other platforms
# Should run from within a linux if CGO_ENABLED=1
build-linux:
	GOOS=linux GOARCH=$(TARGET_ARCH) $(MAKE) build
.PHONY: build-linux

build-linux-docker:
	docker build --label=ostracon --tag="ostracon/ostracon" -f ./DOCKER/Dockerfile .
.PHONY: build-linux-docker

standalone-linux-docker:
	docker run -it --rm -v "/tmp:/ostracon" -p 26656:26656 -p 26657:26657 -p 26660:26660  ostracon/ostracon
.PHONY: standalone-linux-docker

# XXX Warning: Not test yet
# Runs `make build OSTRACON_BUILD_OPTIONS=cleveldb` from within an Amazon
# Linux (v2)-based Docker build container in order to build an Amazon
# Linux-compatible binary. Produces a compatible binary at ./build/ostracon
build_c-amazonlinux:
	$(MAKE) -C ./DOCKER build_amazonlinux_buildimage
	docker run --rm -it -v `pwd`:/ostracon ostracon/ostracon:build_c-amazonlinux
.PHONY: build_c-amazonlinux

###############################################################################
###                       Local testnet using docker                        ###
###############################################################################

DOCKER_HOME = /go/src/github.com/Finschia/ostracon
DOCKER_CMD = docker run --rm \
                        -v `pwd`:$(DOCKER_HOME) \
                        -w $(DOCKER_HOME)
DOCKER_IMG = golang:1.22-alpine
BUILD_CMD = apk add --update --no-cache git make gcc libc-dev build-base curl jq bash file gmp-dev clang libtool autoconf automake \
	&& cd $(DOCKER_HOME) \
	&& make build-linux

# Login docker-container for confirmation building linux binary
build-shell:
	$(DOCKER_CMD) -it --entrypoint '' ${DOCKER_IMG} /bin/sh
.PHONY: build-shell

build-localnode:
	$(DOCKER_CMD) ${DOCKER_IMG} /bin/sh -c "$(BUILD_CMD)"
.PHONY: build-localnode

build-localnode-docker: build-localnode
	@cd networks/local && make
.PHONY: build-localnode-docker

# Run a 4-node testnet locally
localnet-start: localnet-stop build-localnode-docker
	@if ! [ -f build/node0/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/ostracon:Z ostracon/localnode testnet --config /etc/ostracon/config-template.toml --o . --starting-ip-address 192.167.10.2; fi
	docker-compose up
.PHONY: localnet-start

# Stop testnet
localnet-stop:
	docker-compose down
.PHONY: localnet-stop
