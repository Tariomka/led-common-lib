tests:
	@go test -v ./test/... -count=1

analyse:
	staticcheck ./pkg/...