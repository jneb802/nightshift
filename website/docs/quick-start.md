---
sidebar_position: 3
title: Quick Start
---

# Quick Start

Get running in 2 minutes.

## 1. Install

```bash
brew install marcus/tap/nightshift
```

## 2. Setup

Run the guided setup to configure providers, projects, and budget:

```bash
nightshift setup
```

## 3. Preview

See what Nightshift will do before it does anything:

```bash
nightshift preview
nightshift budget
```

## 4. Run

Execute tasks manually (or let the scheduler handle it):

```bash
nightshift run
```

Use `--dry-run` to simulate without making changes:

```bash
nightshift run --dry-run
```

## 5. Check Results

Review what happened:

```bash
nightshift status --today
```

Everything lands as a branch or PR. Merge what surprises you, close the rest.
