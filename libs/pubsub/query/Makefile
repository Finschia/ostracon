gen_query_parser: peg
	peg -inline -switch query.peg

peg:
	@go install github.com/pointlander/peg@v1.0.1

fuzzy_test:
	go get -u -v github.com/dvyukov/go-fuzz/go-fuzz
	go get -u -v github.com/dvyukov/go-fuzz/go-fuzz-build
	go-fuzz-build github.com/Finschia/ostracon/libs/pubsub/query/fuzz_test
	go-fuzz -bin=./fuzz_test-fuzz.zip -workdir=./fuzz_test/output

.PHONY: gen_query_parser fuzzy_test
