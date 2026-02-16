# CRDT Listing Test Plan

This test plan verifies that 10 independent agents can register themselves in the CRDT-based agent directory and that all instances converge to see all 10 listings.

## Overview

- **Test Case**: `list-10-agents`
- **Instances**: 10 (fixed)
- **Purpose**: End-to-end validation of CRDT-based agent discovery

## How it works

1. Each instance creates an isolated local stack:
   - libp2p host
   - libp2p pubsub
   - embedded IPFS-lite
   - Marketplace listing service

2. Each instance publishes a unique agent listing to the CRDT.

3. Instances exchange peer addresses via Testground sync topic and connect to each other for CRDT gossip propagation.

4. All instances poll their local CRDT until they observe 10 unique listings.

5. Test passes only when **all 10 instances** see all 10 listings.

## Running the Test

### Prerequisites

- Go 1.25+
- Docker (for docker:go builder)
- Testground daemon running (`testground daemon`)

### Import the plan

```bash
testground plan import --from ./testplans/crdt-listing
```

### Run with local:exec runner

```bash
testground run composition --plan=crdt-listing --case=list-10-agents \
  --builder=exec:go --runner=local:exec --instances=10
```

### Run with local:docker runner

```bash
testground run composition --plan=crdt-listing --case=list-10-agents \
  --builder=docker:go --runner=local:docker --instances=10
```

### Using composition file

```bash
testground run composition --file=./testplans/crdt-listing/compositions/list-10.toml
```

## Expected Output

Each instance should log:
- Address publication and peer discovery
- Connection to other peers
- Listing publication to CRDT
- CRDT convergence verification (10 listings found)

The test passes when all instances successfully verify convergence.

## Troubleshooting

- **CRDT convergence timeout**: Increase the timeout in `waitForConvergence` or ensure network connectivity between instances.
- **Peer connection failures**: Check firewall rules and ensure instances can reach each other's addresses.
