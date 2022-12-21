all: docker generator runner

docker:
	# See the default version on test/e2e/pkg/testnet.go:LoadTestnet
	# Relate the "docker-compose.yml" on test/e2e/pkg/infra/docker/docker.go:dockerComposeBytes
	# Relate the "cleanup" on test/e2e/runner/cleanup.go:cleanupDir
	docker build --progress=plain \
		--tag ostracon/e2e-node:local-version \
		-f docker/Dockerfile ../..

# We need to build support for database backends into the app in
# order to build a binary with an Ostracon node in it (for built-in
# ABCI testing).
node:
	go build -o build/node -tags libsodium,badgerdb,boltdb,cleveldb,rocksdb ./node

generator:
	go build -o build/generator -tags libsodium ./generator

runner:
	go build -o build/runner -tags libsodium ./runner

.PHONY: all node docker generator maverick runner
