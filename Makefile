VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test lint golden

build:
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)" -o bin/env-checker ./cmd/env-checker

test:
	go test -race ./...

lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	go vet ./...
	go tool staticcheck ./...

golden:
	go test ./cmd/env-checker -update
