.PHONY: help deps build run test fmt vet clean contracts-build anvil-start anvil-deploy anvil-stop

GO ?= go
BINARY ?= betar
CMD ?= ./cmd/betar
FOUNDRY := $(HOME)/.foundry/bin
ANVIL_PID_FILE := /tmp/betar_anvil.pid

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
	@printf "  anvil-start     Start local Anvil node (background)\n"
	@printf "  anvil-deploy    Deploy ERC-8004 contracts to Anvil\n"
	@printf "  anvil-stop      Stop local Anvil node\n"

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
	cd dev/erc-8004-contracts && $(FOUNDRY)/forge build

clean:
	rm -rf bin

anvil-start:
	@echo "Starting Anvil on http://localhost:8545..."
	@$(FOUNDRY)/anvil > /tmp/betar_anvil.log 2>&1 & \
	echo $$! > $(ANVIL_PID_FILE) && \
	echo "Anvil started with PID $$(cat $(ANVIL_PID_FILE))"

anvil-up: anvil-start anvil-deploy anvil-mint-usdc
	@echo ""
	@echo "=== Local Blockchain Ready ==="
	@echo "RPC: http://localhost:8545"
	@echo "Chain ID: 31337"
	@echo ""
	@echo "Contract Addresses:"
	@echo "  IdentityRegistry:    0x5FbDB2315678afecb367f032d93F642f64180aa3"
	@echo "  ReputationRegistry:  0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"
	@echo "  ValidationRegistry:  0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	@echo "  MockUSDC:           0x0165878A594ca255338adfa4d48449f69242Eb8F"

anvil-deploy:
	@cd dev/erc-8004-contracts && \
	PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 $(FOUNDRY)/forge script script/Deploy.s.sol:Deploy --rpc-url http://localhost:8545 --broadcast

anvil-mint-usdc:
	@echo "Minting 1M USDC to each of the first 5 Anvil accounts..."
	@cd dev/erc-8004-contracts && \
	USDC=0x0165878A594ca255338adfa4d48449f69242Eb8F && \
	PK=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 && \
	$(FOUNDRY)/cast send $$USDC "mint(address,uint256)" 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 1000000000000 --private-key $$PK --rpc-url http://localhost:8545 && \
	$(FOUNDRY)/cast send $$USDC "mint(address,uint256)" 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 1000000000000 --private-key $$PK --rpc-url http://localhost:8545 && \
	$(FOUNDRY)/cast send $$USDC "mint(address,uint256)" 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC 1000000000000 --private-key $$PK --rpc-url http://localhost:8545 && \
	$(FOUNDRY)/cast send $$USDC "mint(address,uint256)" 0x90F79bf6EB2c4f870365E785982E1f101E93b906 1000000000000 --private-key $$PK --rpc-url http://localhost:8545 && \
	$(FOUNDRY)/cast send $$USDC "mint(address,uint256)" 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65 1000000000000 --private-key $$PK --rpc-url http://localhost:8545 && \
	echo "Minting complete!"

anvil-stop:
	@if [ -f $(ANVIL_PID_FILE) ]; then \
		kill $$(cat $(ANVIL_PID_FILE)) 2>/dev/null || true; \
		rm -f $(ANVIL_PID_FILE); \
		echo "Anvil stopped"; \
	else \
		echo "Anvil not running"; \
	fi
