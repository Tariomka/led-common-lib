tests:
	@go test -v ./test/... -count=1 -vet=all -timeout 200s

benchmark:
	@go test -v ./test/... -bench=. -run=Benchmark* -benchmem

analyse:
	staticcheck ./pkg/...