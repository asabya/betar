#!/bin/sh
# docker/demo-runner.sh — waits for both nodes, discovers seller agent, and executes a task.
set -eu

BUYER_API="${BUYER_API:-http://buyer:8424}"
TASK="${DEMO_TASK:-What is 847 multiplied by 239?}"

echo "=== Betar Demo Runner ==="
echo "Buyer API: $BUYER_API"
echo ""

# ── Wait for buyer API to be ready ───────────────────────────────────────────
echo "→ Waiting for buyer node API..."
for i in $(seq 1 90); do
  if wget -qO- "$BUYER_API/health" >/dev/null 2>&1; then
    echo "  ✓ Buyer API ready"
    break
  fi
  if [ "$i" -eq 90 ]; then
    echo "  ✗ Buyer API not ready after 90s"
    exit 1
  fi
  sleep 1
done

# ── Wait for agent discovery via CRDT ────────────────────────────────────────
echo "→ Waiting for seller agent to appear in CRDT..."
AGENT_ID=""
for i in $(seq 1 120); do
  RESP=$(wget -qO- "$BUYER_API/agents" 2>/dev/null || echo "[]")
  AGENT_ID=$(echo "$RESP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
  if [ -n "$AGENT_ID" ]; then
    AGENT_NAME=$(echo "$RESP" | sed -n 's/.*"name":"\([^"]*\)".*/\1/p' | head -1)
    echo "  ✓ Discovered agent: $AGENT_NAME ($AGENT_ID)"
    break
  fi
  if [ "$i" -eq 120 ]; then
    echo "  ✗ No agents discovered after 120s"
    exit 1
  fi
  sleep 2
done

# ── Execute task on the discovered agent ─────────────────────────────────────
echo ""
echo "→ Executing task: $TASK"
echo "  Agent: $AGENT_ID"
echo ""

RESULT=$(wget -qO- --header="Content-Type: application/json" \
  --post-data="{\"input\":\"$TASK\"}" \
  "$BUYER_API/agents/$AGENT_ID/execute" 2>/dev/null || echo '{"error":"request failed"}')

echo "=== Result ==="
echo "$RESULT"
echo ""
echo "=== Demo Complete ==="
