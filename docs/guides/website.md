# Website Guide

The docs site lives at `website/` and deploys to [nightshift.haplab.com](https://nightshift.haplab.com).

## Stack

- **Docusaurus 3.9.2** (classic preset, React 19)
- **Dark-only** theme — no light mode switch
- **Fonts:** Inter (body), JetBrains Mono (headings/code), Crimson Pro (display)
- **Icons:** Lucide icon font via CDN

## Structure

```
website/
├── docusaurus.config.js      # Site config, navbar, footer, fonts
├── sidebars.js               # Docs nav structure
├── package.json
├── src/
│   ├── css/custom.css        # All custom styling (ns-* prefix)
│   └── pages/index.js        # Landing page (React)
├── docs/                     # Markdown documentation
│   ├── intro.md
│   ├── installation.md
│   ├── quick-start.md
│   ├── configuration.md
│   ├── tasks.md
│   ├── budget.md
│   ├── scheduling.md
│   ├── cli-reference.md
│   └── integrations.md
└── static/
    ├── CNAME                 # nightshift.haplab.com
    ├── .nojekyll
    └── img/                  # Logos (nightshift, haplab, sidecar, betamax, td)
```

## Theme

Warm nighttime palette. All CSS variables use the `--ns-` prefix.

| Variable | Value | Usage |
|----------|-------|-------|
| `--ns-amber` | `#e8a849` | Primary accent |
| `--ns-orange` | `#e07b3c` | Secondary accent |
| `--ns-purple` | `#8b6cc1` | Tertiary accent |
| `--ns-rose` | `#c76b7e` | Highlight |
| `--ns-bg-base` | `#0d0b14` | Page background |
| `--ns-bg-surface` | `#13111c` | Card/section background |
| `--ns-bg-elevated` | `#1a1726` | Elevated elements |
| `--ns-text-primary` | `#f0eadd` | Body text |
| `--ns-text-secondary` | `#a89f8f` | Muted text |

CSS classes all use the `ns-` prefix (e.g., `ns-hero`, `ns-featureCard`, `ns-sisterGrid`).

Animated gradient backgrounds use CSS `@property` for smooth `ns-drift` animation (22s loop). Sparkle particle effects fire on copy button clicks.

## Landing Page Sections

`src/pages/index.js` renders these in order:

1. **HeroSection** — logo, animated tagline, subtitle, install command + copy button, CTA buttons
2. **TerminalMockup** — styled terminal showing `nightshift preview` output
3. **FeatureCards** — 6 cards (budget, multi-project, zero-risk, tasks, DX, multi-agent)
4. **WorkflowSection** — 4 steps: Configure → Sleep → Wake up → Review
5. **AgentsSection** — Claude Code + Codex badges
6. **PillGrid** — 18 feature pills (CLI-first, Go binary, etc.)
7. **BottomCTA** — repeated install command + docs/GitHub links
8. **SisterProjects** — Haplab logo + 4 project cards (nightshift, sidecar, betamax, td)

## Docs Sidebar

Defined in `sidebars.js`:

```
intro
Getting Started → installation, quick-start, configuration
Usage → tasks, budget, scheduling
Reference → cli-reference, integrations
```

## Development

```bash
cd website
npm install
npm start          # Dev server at localhost:3000
npm run build      # Production build to build/
npm run serve      # Serve production build locally
```

## Deployment

Automatic via GitHub Actions on push to `main` when `website/**` changes.

**Workflow:** `.github/workflows/deploy-docs.yml`
- Builds with Node 20, `npm ci && npm run build`
- Deploys `website/build` to GitHub Pages via `actions/deploy-pages@v4`

**DNS:** CNAME record `nightshift` → `marcus.github.io` on DigitalOcean (haplab.com domain). GitHub Pages serves from the `CNAME` file in `static/`.

**HTTPS:** Enforced via GitHub Pages settings.

To trigger a deploy, push changes to any file under `website/` on `main`.

## Sister Projects (Haplab Ecosystem)

All haplab sites share a footer with cross-links. When adding a new project:

1. Add its card to the `SisterProjects` component in each site's landing page
2. Add an `ns-sisterCard{Color}` CSS class (or equivalent per site's prefix)
3. Copy its logo to each site's `static/img/`
4. Update the grid column count if needed
5. Add a card to [haplab.com](https://haplab.com) (`~/code/haplab/index.html`) and redeploy with `bash deploy.sh`

Current sites: nightshift (amber), sidecar (green), betamax (blue), td (purple).
