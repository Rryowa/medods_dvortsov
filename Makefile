PROJECT_DIR = $(shell pwd)

run:
	go run ./cmd/main.go

lint:
	golangci-lint run ./...

lint-fast:
	golangci-lint run ./... --fast

lint-fix:
	golangci-lint run ./... --fix

gen:
	go generate ./...

.PHONY: run lint lint-fast lint-fix gen