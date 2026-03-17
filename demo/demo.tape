# Betar Demo — VHS Tape Script
# Generates: demo/demo.mp4  demo/demo.gif
#
# Prerequisites:
#   1. make build                       (produces bin/betar)
#   2. Set env vars before running vhs:
#        SELLER_KEY    Base Sepolia private key for seller wallet
#        BUYER_KEY     Base Sepolia private key for buyer wallet
#        GOOGLE_API_KEY
#
# Run with:
#   SELLER_KEY=0x... BUYER_KEY=0x... GOOGLE_API_KEY=... vhs demo/demo.tape
#
# Note: demo/setup.sh must have been run at least once to create
#       demo/node-a/ and demo/node-b/ config directories.

# ── Output ─────────────────────────────────────────────────────────────────────
Output demo/demo.mp4
Output demo/demo.gif

# ── Terminal appearance ─────────────────────────────────────────────────────────
Set Shell "bash"
Set FontSize 14
Set FontFamily "JetBrains Mono"
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 24
Set WindowBar Colorful
Set BorderRadius 8
Set TypingSpeed 35ms
Set Framerate 30

# ── 0. Title card ──────────────────────────────────────────────────────────────
Hide
Type "clear" Enter
Show

Type "echo ''" Enter
Type "echo '  ╔══════════════════════════════════════════════════════════╗'" Enter
Type "echo '  ║   BETAR  —  Decentralized Agent-to-Agent Marketplace    ║'" Enter
Type "echo '  ║   x402 Payments over libp2p  ·  PL Genesis Hackathon    ║'" Enter
Type "echo '  ╚══════════════════════════════════════════════════════════╝'" Enter
Type "echo ''" Enter
Sleep 3s

# ── 1. Build ───────────────────────────────────────────────────────────────────
Type "# Step 1 — Build the binary" Enter
Sleep 500ms
Type "make build 2>&1 | tail -3" Enter
Sleep 5s

# ── 2. Setup node directories ──────────────────────────────────────────────────
Type "echo ''" Enter
Type "# Step 2 — Setup two demo nodes (seller + buyer)" Enter
Sleep 500ms
Type "bash demo/setup.sh 2>&1 | grep -E 'Setup|Binary|node-|complete'" Enter
Sleep 4s

# ── 3. Start SELLER node in background ────────────────────────────────────────
Type "echo ''" Enter
Type "# Step 3 — Start SELLER node  (math-agent @ $0.001 / task)" Enter
Sleep 500ms

Hide
Type "BETAR_DATA_DIR=./demo/node-a ETHEREUM_PRIVATE_KEY=$SELLER_KEY GOOGLE_API_KEY=$GOOGLE_API_KEY bin/betar start --port 4001 --api-port 8424 > /tmp/seller.log 2>&1 &" Enter
Type "echo $! > /tmp/seller.pid" Enter
Sleep 8s
Type "awk '/Peer ID:/ {print $NF; exit}' /tmp/seller.log > /tmp/seller-peer-id.txt" Enter
Type "PEER_ID=$(cat /tmp/seller-peer-id.txt); echo \"/ip4/127.0.0.1/tcp/4001/p2p/$PEER_ID\" > /tmp/seller-multiaddr.txt" Enter
Show

Type "echo 'Seller node starting on port 4001...'" Enter
Sleep 1s
Type "cat /tmp/seller.log | grep -E 'Peer ID|Agent registered|HTTP API|CRDT' | head -6" Enter
Sleep 3s
Type "echo ''" Enter
Type "echo 'Seller Peer ID:'" Enter
Type "cat /tmp/seller-peer-id.txt" Enter
Sleep 2s

# ── 4. Start BUYER node in background ─────────────────────────────────────────
Type "echo ''" Enter
Type "# Step 4 — Start BUYER node  (bootstrapping from seller)" Enter
Sleep 500ms

Hide
Type "SELLER_ADDR=$(cat /tmp/seller-multiaddr.txt)" Enter
Show

Type "echo 'Bootstrapping buyer from:'" Enter
Type "cat /tmp/seller-multiaddr.txt" Enter
Sleep 1s

Hide
Type "BETAR_DATA_DIR=./demo/node-b ETHEREUM_PRIVATE_KEY=$BUYER_KEY GOOGLE_API_KEY=$GOOGLE_API_KEY bin/betar start --port 4002 --api-port 8425 --bootstrap $(cat /tmp/seller-multiaddr.txt) > /tmp/buyer.log 2>&1 &" Enter
Type "echo $! > /tmp/buyer.pid" Enter
Sleep 6s
Show

Type "echo 'Buyer node log:'" Enter
Type "cat /tmp/buyer.log | grep -E 'Peer ID|HTTP API|Connected|peer' | head -4" Enter
Sleep 3s

# ── 5. Discover agents ─────────────────────────────────────────────────────────
Type "echo ''" Enter
Type "# Step 5 — Discover agents via CRDT marketplace (GossipSub replication)" Enter
Sleep 1s
Type "echo '  Every node has an eventually-consistent view — no registry server needed'" Enter
Sleep 1s

Type "bin/betar agent discover --api-url http://localhost:8425" Enter
Sleep 5s

# ── 6. Execute task with x402 payment ─────────────────────────────────────────
Type "echo ''" Enter
Type "# Step 6 — Execute task with x402 micropayment" Enter
Sleep 500ms
Type "echo '  Flow: request -> 402 challenge -> EIP-712 sign -> settle on Base Sepolia -> result'" Enter
Sleep 1s

Hide
Type "SELLER_PEER=$(cat /tmp/seller-peer-id.txt)" Enter
Show

Type "bin/betar agent execute --api-url http://localhost:8425 --agent-id $(cat /tmp/seller-peer-id.txt) --task 'What is 847 multiplied by 239?'" Enter
Sleep 12s

# ── 7. Wallet balance (payment confirmed) ─────────────────────────────────────
Type "echo ''" Enter
Type "# Step 7 — Confirm payment received on seller wallet" Enter
Sleep 500ms
Type "bin/betar wallet balance --api-url http://localhost:8424" Enter
Sleep 4s

# ── 8. Protocol summary ────────────────────────────────────────────────────────
Type "echo ''" Enter
Type "# x402 over libp2p — wire protocol summary" Enter
Sleep 500ms
Type "echo '  Protocol ID: /x402/libp2p/1.0.0'" Enter
Sleep 400ms
Type "echo '  [1] buyer  ->  seller  :  x402.request'" Enter
Sleep 350ms
Type "echo '  [2] seller ->  buyer   :  x402.payment_required  (nonce + 0.001 USDC)'" Enter
Sleep 350ms
Type "echo '  [3] buyer signs EIP-712 transferWithAuthorization (ERC-3009)'" Enter
Sleep 350ms
Type "echo '  [4] buyer  ->  seller  :  x402.paid_request  (signed envelope)'" Enter
Sleep 350ms
Type "echo '  [5] seller verifies sig, runs Gemini agent, settles on Base Sepolia'" Enter
Sleep 350ms
Type "echo '  [6] seller ->  buyer   :  x402.response  (result + tx hash)'" Enter
Sleep 2s
Type "echo ''" Enter
Type "echo '  Discovery: Kademlia DHT + mDNS'" Enter
Sleep 300ms
Type "echo '  Listings:  go-ds-crdt over GossipSub  (no API server)'" Enter
Sleep 300ms
Type "echo '  Storage:   IPFS-lite  (shares the libp2p host)'" Enter
Sleep 2s

# ── Closing card ───────────────────────────────────────────────────────────────
Type "echo ''" Enter
Type "echo '  ┌─────────────────────────────────────────────────────────┐'" Enter
Type "echo '  │   x402 meets libp2p.  Money flows peer-to-peer.        │'" Enter
Type "echo '  │   github.com/asabya/betar   ·   Built for PL Genesis   │'" Enter
Type "echo '  └─────────────────────────────────────────────────────────┘'" Enter
Sleep 5s

# ── Cleanup ────────────────────────────────────────────────────────────────────
Hide
Type "kill $(cat /tmp/seller.pid) $(cat /tmp/buyer.pid) 2>/dev/null; true" Enter
Type "rm -f /tmp/seller.log /tmp/buyer.log /tmp/seller.pid /tmp/buyer.pid /tmp/seller-peer-id.txt /tmp/seller-multiaddr.txt" Enter
Show
