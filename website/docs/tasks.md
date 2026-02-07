---
sidebar_position: 5
title: Tasks
---

# Tasks

Nightshift includes 20+ built-in tasks organized by category.

## Browse Tasks

```bash
# List all tasks
nightshift task list

# Filter by category
nightshift task list --category pr
nightshift task list --category analysis

# Filter by cost tier
nightshift task list --cost low
nightshift task list --cost medium

# Show task details
nightshift task show lint-fix
nightshift task show lint-fix --prompt-only
```

## Categories

| Category | Description |
|----------|-------------|
| `pr` | Creates PRs with code changes |
| `analysis` | Produces analysis reports without code changes |
| `options` | Suggests improvements for human review |
| `safe` | Low-risk automated fixes |
| `map` | Codebase mapping and documentation |
| `emergency` | Critical issues (security vulnerabilities) |

## Cost Tiers

| Tier | Token Usage | Examples |
|------|-------------|----------|
| `low` | Minimal | lint-fix, dead-imports |
| `medium` | Moderate | docs-backfill, dead-code |
| `high` | Significant | bug-finder, security-audit |
| `veryhigh` | Large | full-refactor, test-generation |

## Run a Single Task

```bash
# Dry run first
nightshift task run lint-fix --provider claude --dry-run

# Execute
nightshift task run lint-fix --provider claude
```

## Skill Grooming Task

Nightshift includes a built-in `skill-groom` task for keeping project-local skills aligned with the current codebase.

It is enabled by default. You can set its interval and priority:

```yaml
tasks:
  priorities:
    skill-groom: 2
  intervals:
    skill-groom: "168h"
```

To opt out:

```yaml
tasks:
  disabled:
    - skill-groom
```

`skill-groom` uses `README.md` as project context, checks `.claude/skills` and `.codex/skills`, and validates `SKILL.md` content against the Agent Skills format (starting docs lookup from `https://agentskills.io/llms.txt`).

## Custom Tasks

Define custom tasks in your config:

```yaml
tasks:
  custom:
    - name: check-migrations
      category: analysis
      cost: low
      prompt: "Review all database migrations for potential issues..."
      interval: "168h"
```

## Task Cooldowns

Each task has a default cooldown per project. After running `lint-fix` on `~/code/sidecar`, it won't run again on that project for 24 hours. Override with `tasks.intervals` in config.
