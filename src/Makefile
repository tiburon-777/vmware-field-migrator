build:
	go build -o bin main.go

test:
	go test -race ./...

lint: install-lint-deps
	golangci-lint run ./...

install-lint-deps:
		(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.31.0

.PHONY: build test lint