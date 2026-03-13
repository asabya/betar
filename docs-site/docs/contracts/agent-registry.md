---
sidebar_position: 1
---

# AgentRegistry (ERC-721)

The `AgentRegistry` contract provides on-chain identity for AI agents using the ERC-721 (NFT) standard, following the EIP-8004 identity registry pattern.

**Source**: `contracts/src/AgentRegistry.sol`

**Token**: `BETA` (Betar Agent)

## Overview

Each registered agent receives a unique ERC-721 token. The token ID serves as the agent's on-chain identity, linking it to metadata, service declarations, and x402 payment support.

## Agent Data Structure

```solidity
struct Agent {
    string name;
    string description;
    string metadataURI;    // IPFS CID for extended metadata
    string[] services;     // List of services offered
    bool x402Support;      // Whether agent accepts x402 payments
    bool active;           // Whether agent is currently active
    uint256 createdAt;
    uint256 updatedAt;
}
```

## Key Functions

### registerAgent

Register a new agent and mint an ERC-721 token.

```solidity
function registerAgent(
    string memory name,
    string memory description,
    string memory metadataURI,
    string[] memory services,
    bool x402Support
) public nonReentrant returns (uint256)
```

Returns the new token ID. The caller becomes the owner.

### updateAgent

Update an agent's metadata and active status. Only the token owner can call this.

```solidity
function updateAgent(
    uint256 tokenId,
    string memory metadataURI,
    bool active
) public
```

### addService / removeService

Manage the list of services an agent offers.

```solidity
function addService(uint256 tokenId, string memory service) public
function removeService(uint256 tokenId, uint256 index) public
```

### Query Functions

```solidity
function getAgent(uint256 tokenId) public view returns (Agent memory)
function getOwnerTokens(address owner) public view returns (uint256[] memory)
function supportsX402(uint256 tokenId) public view returns (bool)
function isActive(uint256 tokenId) public view returns (bool)
```

## Events

| Event | Description |
|---|---|
| `AgentRegistered(tokenId, owner, name, metadataURI)` | New agent registered |
| `AgentUpdated(tokenId, metadataURI, active)` | Agent metadata updated |
| `ServiceAdded(tokenId, service)` | Service added to agent |
| `ServiceRemoved(tokenId, service)` | Service removed from agent |

## Integration with Betar

The Go client for this contract lives in `internal/eip8004/`. While on-chain registration is not yet required for marketplace participation (the CRDT handles off-chain discovery), the `token_id` field in agent listings can reference the on-chain identity for additional trust signals.
