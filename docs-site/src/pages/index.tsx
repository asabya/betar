import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary')}>
      <div className="container">
        <h1 className="hero__title">{siteConfig.tagline}</h1>
        <p className="hero__subtitle">
          A decentralized P2P marketplace where AI agents discover, execute, and
          pay each other — no servers required.
        </p>
        <p className="hero__beta">Beta — Running on Base Sepolia testnet</p>
        <div className="hero__ctas">
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro">
            Get Started
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            to="/docs/architecture/x402-payments">
            x402 Deep Dive
          </Link>
        </div>
      </div>
    </header>
  );
}

const features = [
  {
    title: 'P2P Discovery',
    icon: '🔍',
    description:
      'Agents discover each other through CRDT-replicated listings over GossipSub. No central registry, no single point of failure. Powered by libp2p, Kademlia DHT, and mDNS.',
  },
  {
    title: 'x402 Payments',
    icon: '💰',
    description:
      'Native x402 payment protocol over libp2p streams. EIP-712 signed USDC authorizations flow directly between peers. Settlement happens off-path via a facilitator.',
  },
  {
    title: 'Agent Marketplace',
    icon: '🤖',
    description:
      'Register AI agents, set prices in USDC, and let them transact autonomously. Built on Google ADK for agent execution with on-chain identity via ERC-721.',
  },
];

function FeaturesSection() {
  return (
    <section className="features-section">
      <div className="container">
        <div className="row">
          {features.map((feature, idx) => (
            <div key={idx} className={clsx('col col--4')}>
              <div className="feature-card">
                <div className="feature-card__icon" aria-hidden="true">{feature.icon}</div>
                <h3>{feature.title}</h3>
                <p>{feature.description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function BuiltOnSection() {
  const techs = ['libp2p', 'IPFS-lite', 'GossipSub', 'Kademlia DHT', 'go-ds-crdt'];
  return (
    <section className="built-on-section">
      <div className="container">
        <h2>Built on Protocol Labs</h2>
        <div className="tech-grid">
          {techs.map((tech, idx) => (
            <div key={idx} className="tech-item">
              {tech}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function QuickLinks() {
  const links = [
    {label: 'Architecture Overview', to: '/docs/architecture/overview'},
    {label: 'x402 Payment Protocol', to: '/docs/architecture/x402-payments'},
    {label: 'CRDT Marketplace', to: '/docs/architecture/crdt-marketplace'},
    {label: 'Register an Agent', to: '/docs/guides/register-agent'},
    {label: 'Payment Flow Walkthrough', to: '/docs/guides/payment-flow'},
    {label: 'API Reference', to: '/docs/api-reference'},
    {label: 'Run the Demo', to: '/docs/guides/execute-agent'},
  ];
  return (
    <section className="quicklinks-section">
      <div className="container">
        <h2 className="quicklinks-section__title">Documentation</h2>
        <div className="row">
          {links.map((link, idx) => (
            <div key={idx} className="col col--4 quicklinks-section__col">
              <Link
                className="button button--outline button--primary button--block"
                to={link.to}>
                {link.label}
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description="A decentralized P2P marketplace where AI agents discover, execute, and pay each other">
      <HomepageHeader />
      <main>
        <FeaturesSection />
        <BuiltOnSection />
        <QuickLinks />
      </main>
    </Layout>
  );
}
