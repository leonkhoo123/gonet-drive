.PHONY: test test-cover test-race test-verbose

test:
	CGO_ENABLED=1 go test ./... -count=1

test-cover:
	CGO_ENABLED=1 go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

test-race:
	CGO_ENABLED=1 go test -race ./... -count=1

test-verbose:
	CGO_ENABLED=1 go test -v ./... -count=1
