.PHONY: build clean test

GO = $(shell which go)
BIN = ./bin/

all: build

.PHONY: deps
deps:
	${GO} install github.com/vektra/mockery/v2@v2.42.3
	${GO} mod tidy

.PHONY: mocks
mocks:
	mockery --all --case snake

.PHONY: test
test:
	${GO} test ./...
