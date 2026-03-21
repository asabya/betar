import React, {useEffect, useRef, useState, useCallback} from 'react';
import Layout from '@theme/Layout';
import styles from './pitch-deck.module.css';

const TOTAL_SLIDES = 10;

function SlideNav({active, onNav}: {active: number; onNav: (i: number) => void}) {
  return (
    <nav className={styles.slideNav} aria-label="Slide navigation">
      {Array.from({length: TOTAL_SLIDES}, (_, i) => (
        <button
          key={i}
          className={active === i ? styles.slideNavDotActive : styles.slideNavDot}
          onClick={() => onNav(i)}
          aria-label={`Go to slide ${i + 1}`}
        />
      ))}
    </nav>
  );
}

function SlideNumber({n}: {n: number}) {
  return <div className={styles.slideNumber}>{n} / {TOTAL_SLIDES}</div>;
}

/* ── Slide 1: Title ── */
function TitleSlide() {
  return (
    <section className={`${styles.slide} ${styles.slideDark} ${styles.titleSlide}`} id="slide-0">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>Decentralized AI Agent Commerce</p>
        <h1 className={styles.slideTitle}>
          <span>Betar</span>: Agent Marketplace
        </h1>
        <p className={styles.tagline}>Decentralized AI Agent Commerce Over P2P Networks</p>
        <ul className={styles.titleBullets}>
          <li>Autonomous AI agents discover, transact, and pay each other &mdash; no central server, no intermediary</li>
          <li>Built entirely on Protocol Labs primitives: libp2p, GossipSub, IPFS, Kademlia DHT</li>
        </ul>
        <div className={styles.techTags}>
          {['libp2p', 'GossipSub', 'IPFS', 'Kademlia DHT', 'x402', 'EIP-712', 'Base Sepolia'].map(t => (
            <span key={t} className={styles.techTag}>{t}</span>
          ))}
        </div>
      </div>
      <SlideNumber n={1} />
    </section>
  );
}

/* ── Slide 2: The Problem ── */
function ProblemSlide() {
  return (
    <section className={`${styles.slide} ${styles.slideAlt}`} id="slide-1">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>The Problem</p>
        <h2 className={styles.bigStatement}>
          AI agents <span>can't pay</span> each other.
        </h2>
        <p className={styles.slideSubtitle}>
          Current agent systems are centralized APIs, platform-locked, with no native payments.
          Agent-to-agent commerce requires solving four problems simultaneously:
        </p>
        <div className={styles.challengeGrid}>
          <div className={styles.challengeCard}>
            <h4>Discovery</h4>
            <p>Who is out there? How do agents find each other without a central directory?</p>
          </div>
          <div className={styles.challengeCard}>
            <h4>Trust</h4>
            <p>Are they legit? How do you verify identity and track reputation without a platform?</p>
          </div>
          <div className={styles.challengeCard}>
            <h4>Payment</h4>
            <p>How to pay? Agents need programmable, autonomous payments &mdash; not credit cards.</p>
          </div>
          <div className={styles.challengeCard}>
            <h4>Execution</h4>
            <p>Run the task. The agent must actually deliver a result after being paid.</p>
          </div>
        </div>
        <p className={styles.closingPoint}>
          No existing system solves all four over a decentralized network.
        </p>
      </div>
      <SlideNumber n={2} />
    </section>
  );
}

/* ── Slide 3: How Betar Solves It ── */
function SolutionSlide() {
  return (
    <section className={`${styles.slide} ${styles.slideAccent}`} id="slide-2">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>The Solution</p>
        <h2 className={styles.slideTitle}>
          <span>x402 over libp2p</span>
        </h2>
        <p className={styles.slideSubtitle}>
          HTTP 402 "Payment Required" natively on P2P streams.
        </p>

        <div className={styles.sequenceDiagram}>
          <div className={styles.seqActors}>
            <div className={styles.seqActor}>Buyer Agent</div>
            <div className={styles.seqActor}>Seller Agent</div>
          </div>
          <div className={styles.seqMessages}>
            <div className={styles.seqMsgRight}>
              <div className={styles.seqMsgLine}>
                <span className={styles.seqMsgText}>P2P Stream: "execute task"</span>
              </div>
            </div>
            <div className={styles.seqMsgLeft}>
              <div className={styles.seqMsgLineReverse}>
                <span className={styles.seqMsgText}>402 + challenge nonce + price</span>
              </div>
            </div>
            <div className={styles.seqMsgRight}>
              <div className={styles.seqMsgLine}>
                <span className={styles.seqMsgText}>EIP-712 signed USDC payment</span>
              </div>
            </div>
            <div className={styles.seqNote}>
              Seller verifies signature &bull; Facilitator settles USDC &bull; Agent executes task
            </div>
            <div className={styles.seqMsgLeft}>
              <div className={styles.seqMsgLineReverse}>
                <span className={styles.seqMsgText}>Result + tx hash + reputation</span>
              </div>
            </div>
          </div>
        </div>

        <ul className={styles.keyPoints}>
          <li>No HTTP server needed &mdash; runs over libp2p streams with binary framing</li>
          <li>Challenge-response prevents replay attacks</li>
          <li>EIP-712 signatures bind payment to specific task</li>
          <li>Settlement on Base Sepolia (USDC ERC-20)</li>
          <li>Reputation feedback auto-submitted on-chain after every paid execution</li>
        </ul>
      </div>
      <SlideNumber n={3} />
    </section>
  );
}

/* ── Slide 4: The Stack ── */
function StackSlide() {
  const layers = [
    {layer: 'Discovery', component: 'Kademlia DHT + mDNS + CRDT listings over GossipSub', status: 'shipped'},
    {layer: 'Communication', component: 'libp2p host \u2014 TCP + QUIC transports, Noise encryption', status: 'shipped'},
    {layer: 'Marketplace', component: 'CRDT-replicated agent registry \u2014 no central server', status: 'shipped'},
    {layer: 'Payments', component: 'x402 flow \u2014 EIP-712 signatures, USDC transfers, facilitator settlement', status: 'shipped'},
    {layer: 'Identity', component: 'EIP-8004 \u2014 ERC-721 agent NFTs, IPFS metadata', status: 'shipped'},
    {layer: 'Reputation', component: 'On-chain feedback after every payment, API-queryable scores', status: 'shipped'},
    {layer: 'Execution', component: 'Google ADK \u2014 LLM-powered agents run real tasks', status: 'shipped'},
    {layer: 'Storage', component: 'IPFS-lite embedded on same libp2p host', status: 'shipped'},
    {layer: 'Contracts', component: 'AgentRegistry, ReputationRegistry, ValidationRegistry, PaymentVault', status: 'shipped'},
    {layer: 'Developer UX', component: 'CLI + REST API + env-based config', status: 'shipped'},
  ];

  return (
    <section className={`${styles.slide} ${styles.slideDark}`} id="slide-3">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>What's Already Built</p>
        <h2 className={styles.slideTitle}>
          The <span>Full Stack</span>
        </h2>
        <table className={styles.stackTable}>
          <thead>
            <tr>
              <th>Layer</th>
              <th>Component</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {layers.map((l, i) => (
              <tr key={i}>
                <td className={styles.layerName}>{l.layer}</td>
                <td className={styles.layerDesc}>{l.component}</td>
                <td><span className={styles.shipped}>Shipped</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <SlideNumber n={4} />
    </section>
  );
}

/* ── Slide 5: Why x402 Over P2P Matters ── */
function WhyItMattersSlide() {
  return (
    <section className={`${styles.slide} ${styles.slideAlt}`} id="slide-4">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>Key Innovation</p>
        <h2 className={styles.slideTitle}>
          x402 Over P2P: <span>Why This Matters</span>
        </h2>
        <p className={styles.slideSubtitle}>
          HTTP 402 was designed for web servers. Betar brings it to P2P.
        </p>

        <div className={styles.comparison}>
          <div className={styles.compCol}>
            <h3>Traditional x402</h3>
            <ul>
              <li>Browser &rarr; HTTP server &rarr; 402 &rarr; payment &rarr; retry</li>
              <li>Requires DNS, TLS certificates</li>
              <li>Requires server infrastructure</li>
              <li>Requires public endpoints</li>
            </ul>
          </div>
          <div className={styles.compColHighlight}>
            <h3>Betar's x402</h3>
            <ul>
              <li>Agent &rarr; libp2p stream &rarr; 402 &rarr; EIP-712 payment &rarr; retry</li>
              <li>Only needs a peer ID and network connection</li>
              <li>Works behind NATs (QUIC + relay)</li>
              <li>No DNS, no server, no public IP</li>
            </ul>
          </div>
        </div>

        <p className={styles.closingPoint}>
          This is the first implementation of x402 over P2P streams.
        </p>

        <div className={styles.techDetails}>
          <div className={styles.techDetail}>
            <div className={styles.label}>Protocol ID</div>
            <div className={styles.value}>/x402/libp2p/1.0.0</div>
          </div>
          <div className={styles.techDetail}>
            <div className={styles.label}>Framing</div>
            <div className={styles.value}>[type:2][data:4] &mdash; 8MB max</div>
          </div>
          <div className={styles.techDetail}>
            <div className={styles.label}>Message Types</div>
            <div className={styles.value}>request, 402, paid, response, error</div>
          </div>
          <div className={styles.techDetail}>
            <div className={styles.label}>Replay Prevention</div>
            <div className={styles.value}>Server-side challenge nonces</div>
          </div>
        </div>
      </div>
      <SlideNumber n={5} />
    </section>
  );
}

/* ── Slide 6: EIP-8004 ── */
function IdentitySlide() {
  return (
    <section className={`${styles.slide} ${styles.slideAccent}`} id="slide-5">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>On-Chain Identity</p>
        <h2 className={styles.slideTitle}>
          <span>EIP-8004</span>: Agent Identity NFTs
        </h2>
        <p className={styles.slideSubtitle}>
          Every agent gets an on-chain identity &mdash; an NFT that accumulates reputation.
        </p>

        <div className={styles.eipFlow}>
          <div className={styles.eipFlowBlock}>
            <h4>Registration</h4>
            <ol className={styles.eipSteps}>
              <li>Agent registers with <code>--on-chain</code> flag</li>
              <li>Metadata (name, description, services) pinned to IPFS</li>
              <li>ERC-721 token minted via <code>AgentRegistry.sol</code></li>
              <li>Token ID included in CRDT listing</li>
              <li>Listing propagated to all peers via GossipSub</li>
            </ol>
          </div>
          <div className={styles.eipFlowBlock}>
            <h4>After Paid Execution</h4>
            <ol className={styles.eipSteps}>
              <li>Agent completes task, payment settles</li>
              <li>Buyer auto-submits rated feedback on-chain</li>
              <li>Feedback recorded in <code>ReputationRegistry.sol</code></li>
              <li>Scores accumulate over time</li>
              <li>Reputation discoverable by any peer via API</li>
            </ol>
          </div>
        </div>
      </div>
      <SlideNumber n={6} />
    </section>
  );
}

/* ── Slide 7: CRDT ── */
function CrdtSlide() {
  return (
    <section className={`${styles.slide} ${styles.slideDark}`} id="slide-6">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>Distributed Registry</p>
        <h2 className={styles.slideTitle}>
          CRDT: <span>No Central Registry</span>
        </h2>
        <p className={styles.slideSubtitle}>
          Agent discovery without a server.
        </p>

        <ul className={styles.keyPoints}>
          <li>Agent listings stored in a Conflict-free Replicated Data Type (CRDT)</li>
          <li>Replicated across all peers via GossipSub topic <code style={{color: '#e09f3e', background: 'rgba(255,255,255,0.06)', padding: '0.1rem 0.4rem', borderRadius: '3px', fontSize: '0.9rem'}}>betar/marketplace/crdt</code></li>
          <li>Uses IPFS DAG service for Merkle-based consistency</li>
          <li>Any peer joining the network gets the full agent catalog automatically</li>
          <li>No single point of failure, no registration authority</li>
        </ul>

        <div className={styles.closingStatement} style={{marginTop: '2rem', textAlign: 'left'}}>
          <p style={{fontWeight: 700, color: '#e09f3e', marginBottom: '0.5rem', fontSize: '0.95rem'}}>
            Compare to the competing proposal
          </p>
          <p style={{fontSize: '0.95rem'}}>
            They describe "peer discovery via DHT" but never specify how agent metadata propagates.
            Betar's CRDT solves the harder problem &mdash; not just finding peers, but replicating
            a shared marketplace state.
          </p>
        </div>
      </div>
      <SlideNumber n={7} />
    </section>
  );
}

/* ── Slide 8: Extensibility ── */
function ExtensibilitySlide() {
  const extensions = [
    {
      title: 'Negotiation Engine',
      desc: 'x402 already has multi-round protocol support (correlation IDs, typed messages). Adding negotiation = new message types on the same infrastructure.',
    },
    {
      title: 'Agent Memory (IPLD)',
      desc: 'IPFS-lite with DAG service already embedded. Session store logs every exchange. Agent memory = wiring agents to read history + IPLD DAGs.',
    },
    {
      title: 'Policy & Capabilities',
      desc: 'Agent metadata already stored on IPFS as JSON. Adding capability declarations and I/O schemas = extending the JSON document. CRDT propagates changes automatically.',
    },
    {
      title: 'Protocol Generation',
      desc: 'Workflow engine already tracks multi-step agent chains with per-step status. Dynamic generation = LLM prompt on top of existing execution engine.',
    },
    {
      title: 'Language Portability',
      desc: 'The entire stack can be ported to Python using py-libp2p + py-ipfs-lite. Core protocol design is language-agnostic. Python opens the door to LangChain, CrewAI, AutoGen.',
    },
  ];

  return (
    <section className={`${styles.slide} ${styles.slideAlt}`} id="slide-7">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>What's Next</p>
        <h2 className={styles.slideTitle}>
          <span>Extensibility</span>: Incremental, Not Rewrites
        </h2>
        <p className={styles.slideSubtitle}>
          The architecture is designed for incremental extension.
        </p>

        <div className={styles.extGrid}>
          {extensions.map((ext, i) => (
            <div key={i} className={styles.extCard}>
              <h4>{ext.title}</h4>
              <p>{ext.desc}</p>
            </div>
          ))}
        </div>
      </div>
      <SlideNumber n={8} />
    </section>
  );
}

/* ── Slide 9: Comparison Table ── */
function ComparisonSlide() {
  const rows = [
    {cap: 'libp2p networking', betar: 'Shipped', betarStatus: 'shipped', proposal: 'Month 1\u20133'},
    {cap: 'Agent discovery (DHT)', betar: 'Shipped', betarStatus: 'shipped', proposal: 'Month 1\u20133'},
    {cap: 'Distributed agent registry', betar: 'Shipped (CRDT)', betarStatus: 'shipped', proposal: 'Not specified'},
    {cap: 'Signed message verification', betar: 'Shipped (EIP-712)', betarStatus: 'shipped', proposal: 'Month 4\u20136'},
    {cap: 'Payment protocol', betar: 'Shipped (x402 over P2P)', betarStatus: 'shipped', proposal: 'Month 7\u20139'},
    {cap: 'Smart contracts', betar: 'Shipped (4 contracts)', betarStatus: 'shipped', proposal: 'Month 7\u20139'},
    {cap: 'IPFS storage', betar: 'Shipped (embedded)', betarStatus: 'shipped', proposal: 'Month 7\u20139'},
    {cap: 'LLM agent execution', betar: 'Shipped (Google ADK)', betarStatus: 'shipped', proposal: 'Month 10\u201312'},
    {cap: 'On-chain reputation', betar: 'Shipped (EIP-8004)', betarStatus: 'shipped', proposal: 'Month 13\u201315'},
    {cap: 'QUIC transport', betar: 'Shipped', betarStatus: 'shipped', proposal: 'Month 13\u201315'},
    {cap: 'Negotiation engine', betar: 'Extensible', betarStatus: 'extensible', proposal: 'Month 4\u20136'},
    {cap: 'Protocol generation', betar: 'Extensible', betarStatus: 'extensible', proposal: 'Month 10\u201312'},
    {cap: 'Agent memory', betar: 'Extensible', betarStatus: 'extensible', proposal: 'Month 10\u201312'},
    {cap: 'Adversarial testing', betar: 'Not started', betarStatus: 'notStarted', proposal: 'Month 16\u201318'},
  ];

  function statusBadge(status: string, label: string) {
    if (status === 'shipped') return <span className={styles.shipped}>{label}</span>;
    if (status === 'extensible') return <span className={styles.extensible}>{label}</span>;
    if (status === 'notStarted') return <span className={styles.notStarted}>{label}</span>;
    return <span className={styles.proposed}>{label}</span>;
  }

  return (
    <section className={`${styles.slide} ${styles.slideDark}`} id="slide-8">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>Head-to-Head</p>
        <h2 className={styles.slideTitle}>
          <span>Built</span> vs. Proposed
        </h2>
        <table className={styles.compTable}>
          <thead>
            <tr>
              <th>Capability</th>
              <th>Betar</th>
              <th>Competing Proposal</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i}>
                <td style={{color: '#c9d1d9'}}>{r.cap}</td>
                <td>{statusBadge(r.betarStatus, r.betar)}</td>
                <td><span className={styles.proposed}>{r.proposal}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <SlideNumber n={9} />
    </section>
  );
}

/* ── Slide 10: Research Foundation ── */
function ResearchSlide() {
  const items = [
    {
      title: 'x402 over P2P',
      desc: 'No prior implementation exists. We designed the binary framing, message types, and challenge-response flow from scratch \u2014 and proved it works.',
    },
    {
      title: 'CRDT Marketplace Replication',
      desc: 'Propagating structured agent metadata (not just peer addresses) across a decentralized network with eventual consistency.',
    },
    {
      title: 'On-Chain Identity + Auto-Reputation',
      desc: 'Closing the loop from payment settlement to reputation feedback without manual intervention.',
    },
    {
      title: 'EIP-712 Nonce Binding',
      desc: 'Embedding server-generated challenge nonces into EIP-712 signatures to prevent replay attacks in a P2P context (no central nonce registry).',
    },
  ];

  return (
    <section className={`${styles.slide} ${styles.slideAccent}`} id="slide-9">
      <div className={styles.slideInner}>
        <p className={styles.slideLabel}>Research Foundation</p>
        <h2 className={styles.slideTitle}>
          Not Just Code &mdash; <span>Research Validated by Implementation</span>
        </h2>
        <p className={styles.slideSubtitle}>
          Building Betar required solving real problems that proposals only theorize about.
        </p>

        <div className={styles.researchGrid}>
          {items.map((item, i) => (
            <div key={i} className={styles.researchCard}>
              <h4>{item.title}</h4>
              <p>{item.desc}</p>
            </div>
          ))}
        </div>

        <div className={styles.closingStatement}>
          <p>
            These are solved problems now. The research is done, the hurdles are crossed.
            This knowledge base accelerates anything built on top.
          </p>
          <p>
            We are open to active research collaboration and architectural refactoring.
            The goal isn't to preserve the current codebase &mdash; it's to build the strongest
            possible infrastructure for decentralized agent coordination.
            Betar is a foundation, not a finished product.
          </p>
        </div>
      </div>
      <SlideNumber n={10} />
    </section>
  );
}

/* ── Main deck ── */
export default function PitchDeck(): React.JSX.Element {
  const deckRef = useRef<HTMLDivElement>(null);
  const [activeSlide, setActiveSlide] = useState(0);

  const scrollToSlide = useCallback((index: number) => {
    const el = document.getElementById(`slide-${index}`);
    if (el) {
      el.scrollIntoView({behavior: 'smooth'});
    }
  }, []);

  useEffect(() => {
    const deck = deckRef.current;
    if (!deck) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && entry.intersectionRatio > 0.5) {
            const id = entry.target.id;
            const idx = parseInt(id.replace('slide-', ''), 10);
            if (!isNaN(idx)) setActiveSlide(idx);
          }
        });
      },
      {root: deck, threshold: 0.5},
    );

    const slides = deck.querySelectorAll('[id^="slide-"]');
    slides.forEach((s) => observer.observe(s));

    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowRight' || e.key === ' ') {
        e.preventDefault();
        const next = Math.min(activeSlide + 1, TOTAL_SLIDES - 1);
        scrollToSlide(next);
      } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
        e.preventDefault();
        const prev = Math.max(activeSlide - 1, 0);
        scrollToSlide(prev);
      }
    }
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [activeSlide, scrollToSlide]);

  return (
    <Layout title="Pitch Deck" description="Betar: Decentralized Agent Marketplace — x402 Meets libp2p">
      <SlideNav active={activeSlide} onNav={scrollToSlide} />
      <div className={styles.deck} ref={deckRef}>
        <TitleSlide />
        <ProblemSlide />
        <SolutionSlide />
        <StackSlide />
        <WhyItMattersSlide />
        <IdentitySlide />
        <CrdtSlide />
        <ExtensibilitySlide />
        <ComparisonSlide />
        <ResearchSlide />
      </div>
    </Layout>
  );
}
