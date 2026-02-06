import { useState, useCallback } from 'react';
import Link from '@docusaurus/Link';
import useBaseUrl from '@docusaurus/useBaseUrl';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';

const BREW_COMMAND = 'brew install marcus/tap/nightshift';

function CopyButton({ text }) {
  const [copied, setCopied] = useState(false);
  const [sparkles, setSparkles] = useState([]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);

      const colors = [
        'var(--ns-amber)',
        'var(--ns-orange)',
        'var(--ns-purple)',
        'var(--ns-rose)',
        'var(--ns-warmgold)',
      ];
      const newSparkles = Array.from({ length: 24 }, (_, i) => ({
        id: Date.now() + i,
        color: colors[Math.floor(Math.random() * colors.length)],
        angle: (i * 15) + (Math.random() * 10 - 5),
        distance: 40 + Math.random() * 60,
        size: 3 + Math.random() * 5,
        delay: Math.random() * 80,
        duration: 500 + Math.random() * 300,
      }));
      setSparkles(newSparkles);

      setTimeout(() => setCopied(false), 2000);
      setTimeout(() => setSparkles([]), 900);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  }, [text]);

  return (
    <div className="ns-copyBtnWrap">
      {sparkles.map(s => (
        <span
          key={s.id}
          className="ns-sparkle"
          style={{
            '--sparkle-angle': `${s.angle}deg`,
            '--sparkle-distance': `${s.distance}px`,
            '--sparkle-size': `${s.size}px`,
            '--sparkle-color': s.color,
            '--sparkle-delay': `${s.delay}ms`,
            '--sparkle-duration': `${s.duration}ms`,
          }}
        />
      ))}
      <button
        type="button"
        className="ns-copyBtn"
        onClick={handleCopy}
        aria-label={copied ? 'Copied' : 'Copy to clipboard'}
      >
        <i className={copied ? 'icon-check' : 'icon-copy'} />
      </button>
    </div>
  );
}

function HeroSection() {
  return (
    <section className="ns-hero">
      <div className="container">
        <img
          src={useBaseUrl('/img/nightshift-logo.png')}
          alt="Nightshift"
          className="ns-heroLogo"
        />
        <p className="ns-tagline">It finds what you forgot to look for.</p>
        <p className="ns-subtitle">
          AI-powered overnight maintenance for your codebase. Uses your remaining
          token budget to find dead code, doc drift, test gaps, security issues, and
          20+ other things silently accumulating while you ship features.
        </p>
        <div className="ns-installWrapper">
          <div className="ns-installBlock">
            <span className="ns-prompt">$ </span>
            <span>{BREW_COMMAND}</span>
          </div>
          <CopyButton text={BREW_COMMAND} />
        </div>
        <div className="ns-ctaLinks">
          <Link className="ns-btnPrimary" to="/docs/intro">
            <i className="icon-book-open" /> Docs
          </Link>
          <a className="ns-btnSecondary" href="https://github.com/marcus/nightshift">
            <i className="icon-github" /> GitHub
          </a>
        </div>
      </div>
    </section>
  );
}

function TerminalMockup() {
  return (
    <section className="ns-terminalSection">
      <div className="ns-terminal">
        <div className="ns-terminalBar">
          <span className="ns-terminalDot" />
          <span className="ns-terminalDot" />
          <span className="ns-terminalDot" />
          <span className="ns-terminalTitle">nightshift preview</span>
        </div>
        <div className="ns-terminalBody">
          <div><span className="ns-prompt">$ </span><span className="ns-command">nightshift preview</span></div>
          <br />
          <div><span className="ns-output">Nightshift v0.9.0</span></div>
          <div><span className="ns-dim">Budget: 68% remaining (daily mode)</span></div>
          <div><span className="ns-dim">Provider: claude (subscription)</span></div>
          <br />
          <div><span className="ns-warn">Next run: 3 tasks across 2 projects</span></div>
          <br />
          <div><span className="ns-output">  ~/code/sidecar</span></div>
          <div><span className="ns-success">    ✓ lint-fix</span> <span className="ns-dim">(low cost, 24h cooldown)</span></div>
          <div><span className="ns-success">    ✓ dead-code</span> <span className="ns-dim">(medium cost, 72h cooldown)</span></div>
          <br />
          <div><span className="ns-output">  ~/code/td</span></div>
          <div><span className="ns-success">    ✓ docs-backfill</span> <span className="ns-dim">(medium cost, 168h cooldown)</span></div>
          <br />
          <div><span className="ns-dim">Estimated budget usage: ~32%</span></div>
        </div>
      </div>
    </section>
  );
}

const FEATURES = [
  {
    icon: '🔋',
    title: 'Budget-Aware',
    description: 'Uses remaining daily allotment. Never exceeds your configured max — default 75%.',
  },
  {
    icon: '📁',
    title: 'Multi-Project',
    description: 'Point it at your repos. It already knows what to look for in each one.',
  },
  {
    icon: '🛡️',
    title: 'Zero Risk',
    description: 'Everything lands as a PR. Merge what surprises you, close the rest.',
  },
  {
    icon: '⚡',
    title: '20+ Built-in Tasks',
    description: 'Linting, dead code, doc drift, security, test gaps, and more — ready to go.',
  },
  {
    icon: '🔧',
    title: 'Great DX',
    description: 'Thoughtful CLI defaults with clear output, reports, and morning summaries.',
  },
  {
    icon: '🤖',
    title: 'Multi-Agent',
    description: 'Works with Claude Code and Codex. Uses whichever has budget remaining.',
  },
];

function FeatureCards() {
  return (
    <section className="ns-features">
      <h2 className="ns-featuresTitle">Like a Roomba for your codebase</h2>
      <div className="ns-featureGrid">
        {FEATURES.map((f, i) => (
          <div key={i} className="ns-featureCard">
            <span className="ns-featureIcon">{f.icon}</span>
            <h3>{f.title}</h3>
            <p>{f.description}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

const WORKFLOW_STEPS = [
  { icon: 'icon-settings', label: 'Configure', desc: 'Set budget, repos, and schedule', color: 'ns-stepAmber' },
  { icon: 'icon-moon', label: 'Sleep', desc: 'Nightshift runs at 2am with your remaining tokens', color: 'ns-stepPurple' },
  { icon: 'icon-sun', label: 'Wake up', desc: 'Morning summary of branches and PRs', color: 'ns-stepOrange' },
  { icon: 'icon-git-pull-request', label: 'Review', desc: 'Merge what surprised you, close the rest', color: 'ns-stepRose' },
];

function WorkflowSection() {
  return (
    <section className="ns-workflow">
      <h2 className="ns-workflowTitle">How it works</h2>
      <div className="ns-workflowGrid">
        {WORKFLOW_STEPS.map((step, i) => (
          <div key={i} className="ns-workflowStep">
            <div className={`ns-stepIcon ${step.color}`}>
              <i className={step.icon} />
            </div>
            <h3>{step.label}</h3>
            <p>{step.desc}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function AgentsSection() {
  return (
    <section className="ns-agents">
      <h2 className="ns-agentsTitle">Works with your agents</h2>
      <p className="ns-agentsSubtitle">Uses the CLI tools you already have installed</p>
      <div className="ns-agentsGrid">
        <div className="ns-agentBadge">
          <div className="ns-agentDot" style={{ background: 'rgba(232, 168, 73, 0.15)', color: 'var(--ns-amber)' }}>C</div>
          <span>Claude Code</span>
        </div>
        <div className="ns-agentBadge">
          <div className="ns-agentDot" style={{ background: 'rgba(0, 200, 150, 0.15)', color: '#7ec97e' }}>X</div>
          <span>Codex</span>
        </div>
      </div>
    </section>
  );
}

const PILLS = [
  'CLI-first', 'Go binary', 'Zero config defaults', 'Homebrew',
  'YAML config', 'Cron scheduling', 'SQLite state', 'PR-based output',
  'Budget calibration', 'Subscription billing', 'API billing',
  'Custom tasks', 'Task cooldowns', 'Multi-provider', 'Dry-run mode',
  'Morning summaries', 'Structured logs', 'macOS & Linux',
];

function PillGrid() {
  return (
    <div className="ns-pillGrid">
      {PILLS.map((pill, i) => (
        <span key={i} className="ns-pill">{pill}</span>
      ))}
    </div>
  );
}

function BottomCTA() {
  return (
    <section className="ns-bottomCta">
      <div className="container">
        <h2 className="ns-bottomCtaTitle">Get started in seconds</h2>
        <div className="ns-installWrapper">
          <div className="ns-installBlock">
            <span className="ns-prompt">$ </span>
            <span>{BREW_COMMAND}</span>
          </div>
          <CopyButton text={BREW_COMMAND} />
        </div>
        <div className="ns-bottomCtaLinks">
          <Link className="ns-btnPrimary" to="/docs/intro">
            <i className="icon-book-open" /> Read the docs
          </Link>
          <a className="ns-btnSecondary" href="https://github.com/marcus/nightshift">
            <i className="icon-github" /> GitHub
          </a>
        </div>
      </div>
    </section>
  );
}

function SisterProjects() {
  return (
    <section className="ns-sisterProjects">
      <div className="container">
        <h2 className="ns-sisterTitle">Sister Projects</h2>
        <a href="https://haplab.com" className="ns-sisterHaplab">
          <img src={useBaseUrl('/img/haplab-logo.png')} alt="Haplab" />
        </a>
        <div className="ns-sisterGrid">
          <a href="https://nightshift.haplab.com/" className="ns-sisterCard ns-sisterCardAmber ns-sisterCardCurrent">
            <div className="ns-sisterLogoWrapper">
              <img src={useBaseUrl('/img/nightshift-logo.png')} alt="Nightshift" className="ns-sisterLogo" />
            </div>
            <p>It finds what you forgot to look for.</p>
          </a>
          <a href="https://sidecar.haplab.com/" className="ns-sisterCard ns-sisterCardGreen">
            <div className="ns-sisterLogoWrapper">
              <img src={useBaseUrl('/img/sidecar-logo.png')} alt="Sidecar" className="ns-sisterLogo" />
            </div>
            <p>You might never open your editor again.</p>
          </a>
          <a href="https://betamax.haplab.com/" className="ns-sisterCard ns-sisterCardBlue">
            <div className="ns-sisterLogoWrapper">
              <img src={useBaseUrl('/img/betamax-logo-fuzzy.png')} alt="Betamax" className="ns-sisterLogo" />
            </div>
            <p>Record anything you see in your terminal.</p>
          </a>
          <a href="https://td.haplab.com/" className="ns-sisterCard ns-sisterCardPurple">
            <div className="ns-sisterLogoWrapper">
              <img src={useBaseUrl('/img/td-logo.png')} alt="td" className="ns-sisterLogo" />
            </div>
            <p>Task management for AI-assisted development.</p>
          </a>
        </div>
      </div>
    </section>
  );
}

export default function Home() {
  return (
    <Layout title="Nightshift" description="AI-powered overnight maintenance for your codebase">
      <div className="ns-pageBg" />
      <main>
        <HeroSection />
        <TerminalMockup />
        <FeatureCards />
        <WorkflowSection />
        <AgentsSection />
        <PillGrid />
        <BottomCTA />
        <SisterProjects />
      </main>
    </Layout>
  );
}
