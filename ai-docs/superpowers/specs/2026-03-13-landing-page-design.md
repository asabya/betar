# Betar Landing Page Design Spec

## Overview

A standalone, single-file marketing landing page for Betar — a decentralized P2P agent-to-agent marketplace. Deployed to GitHub Pages. No build step, no framework.

## Goals

- Communicate what Betar is and why it matters
- Drive visitors to the GitHub repo and getting-started docs
- Establish Betar's visual identity: dark, minimal, dramatic

## Non-Goals

- No waitlist or email capture
- No connection to the existing React dashboard
- No dynamic content or API calls

## Tech Stack

- Single `index.html` file with inline CSS and minimal JS
- No build tools, no dependencies
- Deployed via GitHub Pages (separate branch or `/docs` folder)

## Visual Design

- **Background:** Dark (#0a0a1a) with animated floating indigo orbs (CSS keyframe animations, radial gradients)
- **Primary accent:** Indigo #6366f1
- **Typography:** Space Grotesk for headings, Inter for body (loaded via Google Fonts)
- **Style:** Minimal layout with dramatic depth — glowing orbs, subtle gradients, clean whitespace
- **Responsive:** Mobile-first, breakpoints at 768px and 1024px

## Page Sections

### 1. Navigation (fixed)

- Betar logo (text)
- Anchor links: How It Works, Features, On-Chain, Values
- GitHub link (highlighted)

### 2. Hero

- Badge: "Decentralized Agent Marketplace"
- Heading: "Betar"
- Tagline: "x402 Meets libp2p. Money Flows Peer-to-Peer."
- Subtitle: "Autonomous agents discover each other, list services, and transact using EIP-402/x402 payments — no middleman, no central server."
- CTAs: "Get Started" (primary, links to docs/setup guide) + "View on GitHub" (secondary, links to repo)
- Background: Animated floating indigo orbs

### 3. How It Works (3 steps)

1. **Discover** — Agents find each other via libp2p DHT and GossipSub CRDT replication. No central registry needed.
2. **Negotiate** — Direct P2P streams. Buyer requests a service, seller responds with pricing.
3. **Transact** — x402 payment flow — EIP-712 signed USDC transfers on Base Sepolia.

### 4. The Agent Economy (2x2 feature grid)

- **P2P Discovery** — Kademlia DHT + GossipSub CRDT replication. Agents are discoverable without centralized registries.
- **Agent Execution** — Run agents locally via Google ADK or delegate remotely over P2P streams.
- **x402 Payments** — HTTP 402 payment flow — USDC on Base. EIP-712 signatures, facilitator settlement.
- **Multi-Agent Workflows** — Chain agents into persistent workflows. Orchestrate complex tasks across the network.

### 5. On-Chain (3 cards with indigo top border)

- **EIP-8004** — On-chain agent registry. ERC-721 identity for every agent.
- **Reputation** — Aggregated on-chain feedback. Trust scores backed by smart contracts.
- **PaymentVault** — USDC escrow and settlement. Transparent, verifiable transactions.

### 6. Values (3x2 grid)

- **Decentralized** — No central server. Agents connect directly.
- **Permissionless** — Anyone can register and offer agent services.
- **Transparent** — All payments and reputation on-chain.
- **Interoperable** — Standard protocols. EIP-402, libp2p, ERC-721.
- **Autonomous** — Agents act independently. No human in the loop.
- **Open Source** — Built in the open. Verify everything.

### 7. Footer

- Copyright: "© 2026 Betar. Built for the agent economy."
- Links: GitHub, Docs, Base Sepolia

## Animations

- **Floating orbs:** 3-4 large radial-gradient circles with CSS keyframe animations (slow drift, 15-25s cycles). Positioned absolutely behind hero and scattered through the page. Indigo/purple tones with blur.
- **Scroll fade-in:** Sections fade in on scroll using IntersectionObserver (minimal JS).
- **CTA hover:** Subtle translateY(-2px) + box-shadow on hover.

## Deployment

- GitHub Pages from a `gh-pages` branch or `/docs` folder
- Single `index.html` at the root of the deployed branch
- No CI/CD needed — just push the file

## File Structure

```
landing/
  index.html    # The entire landing page
```

Kept in a `landing/` directory in the repo to separate it from the existing `web/` React app.
