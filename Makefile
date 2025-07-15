PROJECT_DIR = $(shell pwd)

lint:
	golangci-lint run ./...

lint-fast:
	golangci-lint run ./... --fast

lint-fix:
	golangci-lint run ./... --fix

.PHONY: lint lint-fast lint-fix