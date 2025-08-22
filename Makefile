tests:
	@go test -v ./test/... -count=1 -vet=all -timeout 60s

benchmark:
	@go test -v ./test/... -bench=. -run=Benchmark* -benchmem

analyse:
	staticcheck ./pkg/...