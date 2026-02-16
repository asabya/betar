.PHONY: help deps build run test fmt vet clean contracts-build

GO ?= go
BINARY ?= betar
CMD ?= ./cmd/betar

help:
	@printf "Available targets:\n"
	@printf "  deps            Download Go dependencies\n"
	@printf "  build           Build betar binary\n"
	@printf "  run             Run betar CLI\n"
	@printf "  test            Run Go tests\n"
	@printf "  fmt             Format Go code\n"
	@printf "  vet             Run go vet\n"
	@printf "  contracts-build Build Solidity contracts with forge\n"
	@printf "  clean           Remove build artifacts\n"

deps:
	$(GO) mod download

build:
	mkdir -p bin
	$(GO) build -o bin/$(BINARY) $(CMD)

run:
	$(GO) run ./cmd/betar

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

contracts-build:
	forge build

clean:
	rm -rf bin
