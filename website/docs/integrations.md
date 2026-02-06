---
sidebar_position: 9
title: Integrations
---

# Integrations

Nightshift integrates with your existing development workflow.

## Claude Code

Nightshift uses the Claude Code CLI to execute tasks. Authenticate via subscription or API key:

```bash
claude
/login
```

## Codex

Nightshift supports OpenAI's Codex CLI as an alternative provider:

```bash
codex --login
```

## GitHub

All output is PR-based. Nightshift creates branches and pull requests for its findings.

## td (Task Management)

Nightshift can source tasks from [td](https://td.haplab.com) — task management for AI-assisted development. Tasks tagged with `nightshift` in td will be picked up automatically.

## CLAUDE.md / AGENTS.md

Nightshift reads project-level instruction files to understand context when executing tasks. Place a `CLAUDE.md` or `AGENTS.md` in your repo root to give Nightshift project-specific guidance.

## GitHub Issues

Source tasks from GitHub issues labeled with `nightshift`:

```yaml
integrations:
  github_issues:
    enabled: true
    label: "nightshift"
```
