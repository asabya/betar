# Betar Landing Page Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone single-file landing page for the Betar decentralized agent marketplace, deployable to GitHub Pages.

**Architecture:** Single `index.html` with inline CSS and minimal JS. No build tools, no framework. The file lives in `landing/` to stay separate from the existing `web/` React dashboard.

**Tech Stack:** HTML, CSS (inline), vanilla JS (IntersectionObserver for scroll animations), Google Fonts (Space Grotesk + Inter)

**Spec:** `docs/superpowers/specs/2026-03-13-landing-page-design.md`

---

## Chunk 1: Build the Landing Page

### Task 1: Create the HTML structure with all sections

**Files:**
- Create: `landing/index.html`

- [ ] **Step 1: Create the `landing/` directory**

```bash
mkdir -p landing
```

- [ ] **Step 2: Write `landing/index.html`**

Write the complete single-file landing page. The file contains everything inline — CSS in `<style>`, JS in `<script>`. Structure:

**HTML `<head>`:**
- Meta charset, viewport, title ("Betar — Decentralized Agent Marketplace")
- Google Fonts: Space Grotesk (700) + Inter (400, 500, 600)
- Open Graph meta tags (title, description, type)
- Inline `<style>` block with all CSS

**CSS — Reset & Base:**
```css
* { margin: 0; padding: 0; box-sizing: border-box; }
html { scroll-behavior: smooth; scroll-padding-top: 80px; }
body {
  background: #0a0a1a;
  color: #e2e8f0;
  font-family: 'Inter', system-ui, sans-serif;
  line-height: 1.6;
  overflow-x: hidden;
}
```

**CSS — Floating Orbs:**
- 4 orbs using `position: fixed`, `border-radius: 50%`, `filter: blur(80-120px)`
- Radial gradients with indigo/purple tones: `radial-gradient(circle, rgba(99,102,241,0.3), transparent 70%)`
- CSS keyframe animations: `@keyframes float-1` through `float-4`, each 15-25s, infinite, alternate
- Varying sizes (300px-600px), positioned at different corners/areas
- `pointer-events: none; z-index: 0;` so they don't interfere with content

**CSS — Navigation (fixed):**
```css
nav {
  position: fixed;
  top: 0;
  width: 100%;
  z-index: 100;
  background: rgba(10, 10, 26, 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
}
```
- Flex layout: logo left, links right
- Logo: "Betar" in Space Grotesk 700, 20px, white
- Links: 14px Inter, color #94a3b8, hover → #6366f1
- GitHub link: color #6366f1, font-weight 600
- Mobile: hamburger menu (hidden on desktop, visible below 768px)

**CSS — Hero Section:**
- `min-height: 100vh; display: flex; align-items: center; justify-content: center; text-align: center;`
- `position: relative; z-index: 1;` (above orbs)
- Badge: inline-block, `background: rgba(99,102,241,0.12)`, `border: 1px solid rgba(99,102,241,0.25)`, rounded-full, small text, indigo color
- Heading "Betar": Space Grotesk, `font-size: clamp(3rem, 8vw, 6rem)`, white, slight text-shadow with indigo glow
- Tagline: `font-size: clamp(1.1rem, 3vw, 1.5rem)`, color #c7d2fe (light indigo)
- Subtitle: max-width 600px, color #94a3b8, font-size 1rem
- Primary CTA: `background: #6366f1`, white text, padding 14px 32px, rounded 10px, `transition: transform 0.2s, box-shadow 0.2s`, hover → `translateY(-2px)` + `box-shadow: 0 8px 25px rgba(99,102,241,0.4)`
- Secondary CTA: `background: transparent`, `border: 1px solid #334155`, color #94a3b8, same size/shape, hover → `border-color: #6366f1`

**CSS — Section Base:**
```css
section {
  position: relative;
  z-index: 1;
  max-width: 1100px;
  margin: 0 auto;
  padding: 100px 24px;
}
.section-label {
  text-transform: uppercase;
  letter-spacing: 2px;
  font-size: 13px;
  color: #6366f1;
  font-weight: 600;
  margin-bottom: 12px;
}
.section-title {
  font-family: 'Space Grotesk', sans-serif;
  font-size: clamp(1.8rem, 4vw, 2.5rem);
  color: #f1f5f9;
  margin-bottom: 48px;
}
```

**CSS — How It Works (3-column grid):**
- `display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px;`
- Each step card: `background: rgba(30, 41, 59, 0.5)`, `border: 1px solid #1e293b`, rounded 12px, padding 32px
- Step number: 36px circle, `background: rgba(99,102,241,0.15)`, color #818cf8, centered
- Step title: Space Grotesk, 18px, white
- Step description: 14px, #94a3b8
- Responsive: `grid-template-columns: 1fr` below 768px

**CSS — Agent Economy (2x2 grid):**
- `display: grid; grid-template-columns: repeat(2, 1fr); gap: 24px;`
- Feature cards: same card style as How It Works
- Icon: 32px, margin-bottom 12px (use SVG inline or emoji)
- Responsive: `grid-template-columns: 1fr` below 768px

**CSS — On-Chain (3-column grid):**
- Same grid as How It Works
- Cards have `border-top: 3px solid #6366f1` for visual distinction
- Slight indigo glow on hover: `box-shadow: 0 4px 20px rgba(99,102,241,0.15)`

**CSS — Values (3x2 grid):**
- `display: grid; grid-template-columns: repeat(3, 1fr); gap: 32px;`
- Text-centered, no card background — just title + description
- Title: 16px, white, Space Grotesk
- Description: 14px, #64748b
- Responsive: `grid-template-columns: repeat(2, 1fr)` below 768px, `1fr` below 480px

**CSS — Footer:**
```css
footer {
  border-top: 1px solid #1e293b;
  padding: 32px 24px;
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: #64748b;
}
```
- Links: flex gap 24px, color #94a3b8, hover → #6366f1

**CSS — Scroll Animations:**
```css
.fade-in {
  opacity: 0;
  transform: translateY(24px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.fade-in.visible {
  opacity: 1;
  transform: translateY(0);
}
```

**HTML `<body>` structure:**

```
<div class="orb orb-1"></div>
<div class="orb orb-2"></div>
<div class="orb orb-3"></div>
<div class="orb orb-4"></div>

<nav>...</nav>

<section id="hero">
  <div class="badge">Decentralized Agent Marketplace</div>
  <h1>Betar</h1>
  <p class="tagline">x402 Meets libp2p. Money Flows Peer-to-Peer.</p>
  <p class="subtitle">Autonomous agents discover each other...</p>
  <div class="ctas">
    <a href="#" class="btn-primary">Get Started</a>
    <a href="https://github.com/asabya/betar" class="btn-secondary">View on GitHub</a>
  </div>
</section>

<section id="how" class="fade-in">
  <p class="section-label">How It Works</p>
  <h2 class="section-title">Three steps to the agent economy</h2>
  <div class="steps-grid">
    <!-- 3 step cards -->
  </div>
</section>

<section id="features" class="fade-in">
  <p class="section-label">The Agent Economy</p>
  <h2 class="section-title">Everything agents need to thrive</h2>
  <div class="features-grid">
    <!-- 4 feature cards -->
  </div>
</section>

<section id="onchain" class="fade-in">
  <p class="section-label">On-Chain</p>
  <h2 class="section-title">Trust, verified on-chain</h2>
  <div class="onchain-grid">
    <!-- 3 cards -->
  </div>
</section>

<section id="values" class="fade-in">
  <p class="section-label">Values</p>
  <h2 class="section-title">What we believe</h2>
  <div class="values-grid">
    <!-- 6 value items -->
  </div>
</section>

<footer>
  <span>&copy; 2026 Betar. Built for the agent economy.</span>
  <div class="footer-links">
    <a href="https://github.com/asabya/betar">GitHub</a>
    <a href="#">Docs</a>
    <a href="https://sepolia.basescan.org">Base Sepolia</a>
  </div>
</footer>
```

**Inline `<script>` (scroll animation):**
```javascript
document.addEventListener('DOMContentLoaded', () => {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible');
      }
    });
  }, { threshold: 0.1 });

  document.querySelectorAll('.fade-in').forEach(el => observer.observe(el));

  // Mobile menu toggle
  const toggle = document.querySelector('.menu-toggle');
  const links = document.querySelector('.nav-links');
  if (toggle) {
    toggle.addEventListener('click', () => links.classList.toggle('open'));
    links.querySelectorAll('a').forEach(a =>
      a.addEventListener('click', () => links.classList.remove('open'))
    );
  }
});
```

- [ ] **Step 3: Open in browser and verify**

```bash
open landing/index.html
```

Verify: all 7 sections render, orbs animate, scroll fade-in works, mobile responsive, CTAs link correctly.

- [ ] **Step 4: Commit**

```bash
git add landing/index.html
git commit -m "feat: add standalone landing page for GitHub Pages"
```

### Task 2: Add .gitignore entry for brainstorm files

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Add `.superpowers/` to .gitignore**

Append `.superpowers/` to the project `.gitignore` so brainstorm mockup files aren't committed.

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add .superpowers/ to gitignore"
```
