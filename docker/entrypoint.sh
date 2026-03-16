#!/bin/sh
# docker/entrypoint.sh — wrapper that starts betar and shares peer info via /shared volume.
set -eu

SHARED_DIR="${SHARED_DIR:-/shared}"
NODE_ROLE="${NODE_ROLE:-}"

# If buyer role, wait for seller's multiaddr before starting.
if [ "$NODE_ROLE" = "buyer" ]; then
  echo "[entrypoint] Waiting for seller peer info..."
  while [ ! -f "$SHARED_DIR/seller-multiaddr" ]; do
    sleep 1
  done
  export BOOTSTRAP_PEERS
  BOOTSTRAP_PEERS="$(cat "$SHARED_DIR/seller-multiaddr")"
  echo "[entrypoint] Seller multiaddr: $BOOTSTRAP_PEERS"
fi

# Start betar, tee output so we can extract peer ID.
betar "$@" 2>&1 | tee /tmp/betar.log &
BETAR_PID=$!

# If seller, extract and publish peer ID + multiaddr.
if [ "$NODE_ROLE" = "seller" ]; then
  echo "[entrypoint] Waiting for Peer ID..."
  PEER_ID=""
  for _ in $(seq 1 60); do
    if grep -q "Peer ID:" /tmp/betar.log 2>/dev/null; then
      PEER_ID="$(grep "Peer ID:" /tmp/betar.log | head -1 | awk '{print $NF}')"
      break
    fi
    sleep 1
  done

  if [ -z "$PEER_ID" ]; then
    echo "[entrypoint] ERROR: Could not extract Peer ID within 60s"
    kill $BETAR_PID 2>/dev/null || true
    exit 1
  fi

  PORT="${BETAR_PORT:-4001}"
  MULTIADDR="/dns4/seller/tcp/${PORT}/p2p/${PEER_ID}"
  mkdir -p "$SHARED_DIR"
  echo "$MULTIADDR" > "$SHARED_DIR/seller-multiaddr"
  echo "[entrypoint] Published multiaddr: $MULTIADDR"
fi

wait $BETAR_PID
