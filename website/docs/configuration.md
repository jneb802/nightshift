---
sidebar_position: 4
title: Configuration
---

# Configuration

Nightshift uses YAML config files. Run `nightshift setup` for an interactive setup, or edit directly.

## Config Location

- **Global:** `~/.config/nightshift/config.yaml`
- **Per-project:** `nightshift.yaml` or `.nightshift.yaml` in the repo root

## Minimal Config

```yaml
schedule:
  cron: "0 2 * * *"

budget:
  mode: daily
  max_percent: 75
  reserve_percent: 5
  billing_mode: subscription
  calibrate_enabled: true
  snapshot_interval: 30m

providers:
  preference:
    - claude
    - codex
  claude:
    enabled: true
    data_path: "~/.claude"
    dangerously_skip_permissions: true
  codex:
    enabled: true
    data_path: "~/.codex"
    dangerously_bypass_approvals_and_sandbox: true

projects:
  - path: ~/code/sidecar
  - path: ~/code/td
```

## Schedule

Use cron syntax or interval-based scheduling:

```yaml
schedule:
  cron: "0 2 * * *"        # Every night at 2am
  # interval: "8h"         # Or run every 8 hours
```

## Budget

Control how much of your token budget Nightshift uses:

| Field | Default | Description |
|-------|---------|-------------|
| `mode` | `daily` | `daily` or `weekly` |
| `max_percent` | `75` | Max budget % to use per run |
| `reserve_percent` | `5` | Always keep this % available |
| `billing_mode` | `subscription` | `subscription` or `api` |
| `calibrate_enabled` | `true` | Auto-calibrate from local CLI data |

## Task Selection

Enable/disable tasks and set priorities:

```yaml
tasks:
  enabled:
    - lint-fix
    - docs-backfill
    - bug-finder
  priorities:
    lint-fix: 1
    bug-finder: 2
  intervals:
    lint-fix: "24h"
    docs-backfill: "168h"
```

Each task has a default cooldown interval to prevent the same task from running too frequently on a project.

## Providers

Nightshift supports Claude Code and Codex as execution providers. It will use whichever has budget remaining, in the order specified by `preference`.
