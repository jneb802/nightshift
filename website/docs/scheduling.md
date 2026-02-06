---
sidebar_position: 7
title: Scheduling
---

# Scheduling

Nightshift can run automatically on a schedule.

## Cron-Based

```yaml
schedule:
  cron: "0 2 * * *"  # Every night at 2am
```

## Daemon Mode

Run as a persistent background process:

```bash
nightshift daemon start
nightshift daemon start --foreground  # For debugging
nightshift daemon stop
```

## System Service

Install as a system service for automatic startup:

```bash
# macOS (launchd)
nightshift install launchd

# Linux (systemd)
nightshift install systemd

# Universal (cron)
nightshift install cron
```

## Manual Runs

Skip the scheduler and run immediately:

```bash
nightshift run
nightshift run --dry-run
nightshift run --project ~/code/myproject
nightshift run --task lint-fix
```
