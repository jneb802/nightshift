---
sidebar_position: 6
title: Budget
---

# Budget Management

Nightshift is designed to use tokens you'd otherwise waste. It tracks your remaining budget and never exceeds your configured limits.

## Check Budget

```bash
nightshift budget
nightshift budget --provider claude
nightshift budget --provider codex
```

## Budget Modes

### Daily Mode

Uses a percentage of your remaining daily token budget. Resets each day.

### Weekly Mode

Tracks cumulative usage across the week. Better for subscription plans with weekly reset cycles.

## Calibration

Nightshift can auto-calibrate by reading local CLI data:

```bash
nightshift budget calibrate
```

This reads usage data from `~/.claude` and `~/.codex` to determine remaining budget. Enable auto-calibration in config:

```yaml
budget:
  calibrate_enabled: true
  snapshot_interval: 30m
```

## Budget History

View past budget snapshots:

```bash
nightshift budget history -n 10
nightshift budget snapshot --local-only
```

## Safety

- `max_percent` (default 75%) caps how much budget a single run can use
- `reserve_percent` (default 5%) always keeps some budget available for your daytime work
- If budget is exhausted, Nightshift skips remaining tasks gracefully
