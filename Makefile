.PHONY: help deps build run test fmt vet clean contracts-build contracts-deploy web-install web-dev web-build dashboard-embed

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
	@printf "  contracts-deploy Deploy contracts to Base Sepolia (requires ETHEREUM_PRIVATE_KEY)\n"
	@printf "  clean           Remove build artifacts\n"

deps:
	$(GO) mod download

build: dashboard-embed
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
	cd contracts && forge build

contracts-deploy:
	cd contracts && forge script script/Deploy.s.sol --rpc-url https://sepolia.base.org --broadcast --verify

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build:
	cd web && VITE_API_URL='' npm run build

dashboard-embed: web-build
	rm -rf cmd/betar/dashboard/dist
	cp -r web/dist cmd/betar/dashboard/dist

clean:
	rm -rf bin
