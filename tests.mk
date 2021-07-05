#!/usr/bin/make -f

########################################
### Testing

BINDIR ?= $(GOPATH)/bin

## required to be run first by most tests
build_docker_test_image:
	docker build -t tester -f ./test/docker/Dockerfile .
.PHONY: build_docker_test_image

### coverage, app, persistence, and libs tests
test_cover:
	# run the go unit tests with coverage
	bash test/test_cover.sh
.PHONY: test_cover

test_apps:
	# run the app tests using bash
	# requires `abci-cli` and `tendermint` binaries installed
	bash test/app/test.sh
.PHONY: test_apps

test_abci_apps:
	bash abci/tests/test_app/test.sh
.PHONY: test_abci_apps

test_abci_cli:
	# test the cli against the examples in the tutorial at:
	# ./docs/abci-cli.md
	# if test fails, update the docs ^
	@ bash abci/tests/test_cli/test.sh
.PHONY: test_abci_cli

test_integrations:
	make build_docker_test_image
	#make tools # XXX Should remove "make tools": https://github.com/line/ostracon/commit/c6e0d20d4bf062921fcc1eb5b2399447a7d2226e#diff-76ed074a9305c04054cdebb9e9aad2d818052b07091de1f20cad0bbac34ffb52
	make install
	make install_abci
	make test_cover
	make test_apps
	make test_abci_apps
	make test_abci_cli
	#make test_libs # XXX Should remove "make test_libs": https://github.com/line/ostracon/commit/9b9f1beef3cfc5e501fc494afeb4862bd86482f3#diff-76ed074a9305c04054cdebb9e9aad2d818052b07091de1f20cad0bbac34ffb52L114
.PHONY: test_integrations

test_release:
	@go test -tags release $(PACKAGES)
.PHONY: test_release

test100:
	@for i in {1..100}; do make test; done
.PHONY: test100

vagrant_test:
	vagrant up
	vagrant ssh -c 'make test_integrations'
.PHONY: vagrant_test

### go tests
test:
	@echo "--> Running go test"
	@go test -p 1 $(PACKAGES) -tags deadlock
.PHONY: test

test_race:
	@echo "--> Running go test --race"
	@go test -p 1 -v -race $(PACKAGES)
.PHONY: test_race

test_deadlock:
	@echo "--> Running go test --deadlock"
	@go test -p 1 -v  $(PACKAGES) -tags deadlock
.PHONY: test_race

###
#
# WARNING: NOT Support Open API 3. See the CONTRIBUTING.md#RPC Testing
#
###
# Build hooks for dredd, to skip or add information on some steps
build-contract-tests-hooks:
ifeq ($(OS),Windows_NT)
	go build -mod=readonly $(BUILD_FLAGS) -o build/contract_tests.exe ./cmd/contract_tests
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/contract_tests ./cmd/contract_tests
endif
.PHONY: build-contract-tests-hooks

# Run a nodejs tool to test endpoints against a localnet
# The command takes care of starting and stopping the network
# prerequisites: build-contract-tests-hooks
# the build command were not added to let this command run from generic containers or machines.
# The binaries should be built beforehand
contract-tests:
	dredd
.PHONY: contract-tests
