tests:
	@go test -v ./test/... -count=1

benchmark:
	@go test -v ./test/... -bench=. -run=Benchmark* -benchmem

analyse:
	staticcheck ./pkg/...