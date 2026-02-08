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
nightshift run                          # Preflight summary + confirm + execute
nightshift run --dry-run                # Show preflight summary, don't execute
nightshift run --yes                    # Skip confirmation prompt
nightshift run --project ~/code/myproject
nightshift run --task lint-fix
nightshift run --max-projects 3 --max-tasks 2  # Process more projects/tasks
nightshift run --ignore-budget          # Bypass budget limits
```

In interactive terminals, `nightshift run` shows a preflight summary and asks for confirmation before executing. Use `--yes` to skip the prompt (e.g., in scripts). Non-TTY contexts auto-skip confirmation.
