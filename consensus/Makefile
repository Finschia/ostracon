wal_fuzzy_test:
	go get -u -v github.com/dvyukov/go-fuzz/go-fuzz
	go get -u -v github.com/dvyukov/go-fuzz/go-fuzz-build
	go-fuzz-build github.com/tendermint/tendermint/consensus
	go-fuzz -bin=./consensus-fuzz.zip -workdir=./output

.PHONY: wal_fuzzy_test
