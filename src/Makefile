build:
	go build -o ./bin ./src/main.go

run:
	go run ./src/main.go

test:
	go test -race ./src/...

lint: install-lint-deps
	golangci-lint run .src/...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.30.0

.PHONY: build run test lint