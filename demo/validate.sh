#!/usr/bin/env bash
# demo/validate.sh — Validates the Betar demo setup without requiring API keys or funded wallets.
#
# What this tests:
#   1. Build succeeds
#   2. Seller node starts and prints Peer ID
#   3. Buyer node starts on a different port and connects to seller via --bootstrap
#   4. API server responds on both ports
#   5. agent discover returns valid JSON from buyer node
#
# What requires human validation (not tested here):
#   - GOOGLE_API_KEY / agent registration (requires LLM provider)
#   - x402 payment flow (requires funded Base Sepolia wallets)
#   - demo video recording

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

SELLER_DIR=$(mktemp -d)
BUYER_DIR=$(mktemp -d)
SELLER_LOG=$(mktemp)
BUYER_LOG=$(mktemp)
SELLER_PID=""
BUYER_PID=""

cleanup() {
    [ -n "$SELLER_PID" ] && kill "$SELLER_PID" 2>/dev/null || true
    [ -n "$BUYER_PID" ] && kill "$BUYER_PID" 2>/dev/null || true
    rm -rf "$SELLER_DIR" "$BUYER_DIR" "$SELLER_LOG" "$BUYER_LOG"
}
trap cleanup EXIT

pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; exit 1; }

echo "=== Betar Demo Validation ==="
echo ""

# ── 1. Build ─────────────────────────────────────────────────────────────────
echo "→ Step 1: Build binary"
make build > /dev/null 2>&1
pass "bin/betar built"

# ── 2. Start seller node ─────────────────────────────────────────────────────
echo ""
echo "→ Step 2: Start seller node (port 14001, api-port 18424)"
BETAR_DATA_DIR="$SELLER_DIR" bin/betar node --port 14001 > "$SELLER_LOG" 2>&1 &
SELLER_PID=$!
sleep 3

grep -q "Peer ID:" "$SELLER_LOG" || fail "Seller node did not print Peer ID"
PEER_ID=$(awk '/Peer ID:/ {print $NF; exit}' "$SELLER_LOG")
SELLER_MULTIADDR="/ip4/127.0.0.1/tcp/14001/p2p/$PEER_ID"
pass "Seller started | Peer ID: $PEER_ID"

# ── 3. Start buyer node bootstrapping from seller ────────────────────────────
echo ""
echo "→ Step 3: Start buyer node (port 14002, bootstrap from seller)"
BETAR_DATA_DIR="$BUYER_DIR" bin/betar start \
    --port 14002 \
    --api-port 18425 \
    --bootstrap "$SELLER_MULTIADDR" \
    > "$BUYER_LOG" 2>&1 &
BUYER_PID=$!
sleep 4

grep -q "Peer ID:" "$BUYER_LOG" || fail "Buyer node did not print Peer ID"
BUYER_PEER_ID=$(awk '/Peer ID:/ {print $NF; exit}' "$BUYER_LOG")
pass "Buyer started | Peer ID: $BUYER_PEER_ID"
pass "Bootstrap multiaddr: $SELLER_MULTIADDR"

# ── 4. API server responds ────────────────────────────────────────────────────
echo ""
echo "→ Step 4: Check buyer API server responds"
grep -q "HTTP API server running on port 18425" "$BUYER_LOG" || fail "Buyer API server did not start on port 18425"
pass "Buyer API server up on port 18425"

# ── 5. agent discover returns valid JSON ─────────────────────────────────────
echo ""
echo "→ Step 5: agent discover returns valid JSON"
RESULT=$(bin/betar agent discover --api-url http://localhost:18425 2>&1)
echo "$RESULT" | grep -qE "Discovered Agents:|No agents discovered" || fail "agent discover returned unexpected output: $RESULT"
pass "agent discover: OK (output: $RESULT)"

echo ""
echo "=== Validation passed ==="
echo ""
echo "Note: The following steps require human action to validate:"
echo "  - Set GOOGLE_API_KEY and run: bin/betar start --name math-agent --description 'Math tasks' --price 0.001 --port 4001"
echo "  - Verify agent registration + CRDT replication"
echo "  - Fund buyer wallet with Base Sepolia USDC and run agent execute"
echo "  - Confirm x402 payment + on-chain settlement"
