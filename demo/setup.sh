#!/usr/bin/env bash
# demo/setup.sh — Pre-configure two Betar demo nodes on a single machine
# Usage: cd <repo-root> && bash demo/setup.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="$REPO_ROOT/demo"
NODE_A="$DEMO_DIR/node-a"
NODE_B="$DEMO_DIR/node-b"

echo "=== Betar Demo Setup ==="
echo "Repo: $REPO_ROOT"
echo ""

# ── 1. Build binary ──────────────────────────────────────────────────────────
echo "→ Building binary..."
cd "$REPO_ROOT"
make build
echo "  ✓ Binary: bin/betar"
echo ""

# ── 2. Create data directories ───────────────────────────────────────────────
echo "→ Creating node data directories..."
mkdir -p "$NODE_A" "$NODE_B"
echo "  ✓ $NODE_A"
echo "  ✓ $NODE_B"
echo ""

# ── 3. Write agents.yaml for node A (seller) ─────────────────────────────────
cat > "$NODE_A/agents.yaml" << 'EOF'
agents:
  - name: math-agent
    description: "Performs arithmetic, algebra, and general math tasks"
    price: 0.001
    model: gemini-2.5-flash
EOF
echo "  ✓ $NODE_A/agents.yaml (seller: math-agent @ \$0.001/task)"

# ── 4. Write agents.yaml for node B (buyer — no agents, just a client) ───────
cat > "$NODE_B/agents.yaml" << 'EOF'
agents: []
EOF
echo "  ✓ $NODE_B/agents.yaml (buyer: no agents)"
echo ""

# ── 5. Print run instructions ─────────────────────────────────────────────────
cat << 'INSTRUCTIONS'
=== Setup complete! ===

Open TWO terminals from the repo root and run:

─── Terminal A (Seller) ────────────────────────────────────────────────────
export BETAR_DATA_DIR=./demo/node-a
export ETHEREUM_PRIVATE_KEY=<seller-private-key>
export GOOGLE_API_KEY=<your-google-api-key>

bin/betar start --port 4001

Copy the PeerID and multiaddr printed on startup.
────────────────────────────────────────────────────────────────────────────

─── Terminal B (Buyer) ─────────────────────────────────────────────────────
export BETAR_DATA_DIR=./demo/node-b
export ETHEREUM_PRIVATE_KEY=<buyer-private-key>
export GOOGLE_API_KEY=<your-google-api-key>

bin/betar start --port 4002 --bootstrap <seller-multiaddr>

Then discover and execute:
  bin/betar agent discover
  bin/betar agent execute --agent-id <seller-peer-id> --input "What is 847 * 239?"
────────────────────────────────────────────────────────────────────────────

See demo/README.md for the full walkthrough.
INSTRUCTIONS
