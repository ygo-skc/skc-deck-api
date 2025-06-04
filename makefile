.PHONY: test coverage help

# default
all: test build

build:
	go mod tidy
	env GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath .

test:
	go clean -testcache
	go test ./...

coverage:
	go clean -testcache
	go test -coverprofile coverage.out  ./...
	go tool cover -html=coverage.out

help:
	@echo "Makefile Commands:"
	@echo "  all            - Default target. Installs deps, cleans, and builds the binary."
	@echo "  test           - Runs all tests."
	@echo "  coverage       - Runs all tests w/ coverage."
	@echo "  build          - Compile the Go project for Linux ARM64."
	@echo "  help           - Show this help message."