# Nightshift User Guide

Nightshift is a Go CLI tool that runs AI-powered maintenance tasks on your codebase overnight, using your remaining daily token budget from Claude Code or Codex subscriptions. Wake up to a cleaner codebase without unexpected costs.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Monitoring & Visibility](#monitoring--visibility)
- [Integrations](#integrations)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Prerequisites

- Go 1.23 or later
- Claude Code CLI (`claude`) and/or Codex CLI (`codex`) installed
- API keys configured via environment variables:
  - `ANTHROPIC_API_KEY` for Claude
  - `OPENAI_API_KEY` for Codex

### Build from Source

```bash
git clone https://github.com/marcusvorwaller/nightshift.git
cd nightshift
go build -o nightshift ./cmd/nightshift
```

Move the binary to your PATH:

```bash
sudo mv nightshift /usr/local/bin/
```

### Verify Installation

```bash
nightshift --version
nightshift --help
```

---

## Quick Start

### 1. Initialize Configuration

Create a project config in your repository:

```bash
cd ~/code/myproject
nightshift init
```

Or create a global config for all projects:

```bash
nightshift init --global
```

### 2. Run a Dry-Run

See what Nightshift would do without making changes:

```bash
nightshift run --dry-run
```

### 3. Run Tasks

Execute tasks manually:

```bash
nightshift run
```

### 4. Install as System Service

Set up automatic overnight runs:

```bash
# macOS
nightshift install launchd

# Linux
nightshift install systemd

# Universal (cron)
nightshift install cron
```

---

## Configuration

### Config File Locations

| Type | Location |
|------|----------|
| Global | `~/.config/nightshift/config.yaml` |
| Project | `nightshift.yaml` or `.nightshift.yaml` in project root |

Project configs override global settings.

### Basic Configuration

```yaml
# Schedule when Nightshift runs
schedule:
  cron: "0 2 * * *"           # Run at 2 AM daily
  # OR
  interval: 4h                 # Run every 4 hours

  window:                      # Optional: only run during these hours
    start: "22:00"
    end: "06:00"
    timezone: "America/Denver"

# Budget controls
budget:
  mode: daily                  # daily | weekly
  max_percent: 10              # Max % of budget per run
  reserve_percent: 5           # Always keep this % in reserve

# Enable/disable task types
tasks:
  enabled:
    - lint
    - docs
    - security
    - dead-code
  disabled:
    - idea-generator
```

### Budget Modes

**Daily Mode** (recommended for most users):
- Each night uses up to `max_percent` of your daily budget (weekly / 7)
- Consistent, predictable usage

**Weekly Mode**:
- Uses `max_percent` of *remaining* weekly budget
- With `aggressive_end_of_week: true`, spends more near week's end to avoid waste

### Provider Configuration

```yaml
providers:
  claude:
    enabled: true
    data_path: "~/.claude"     # Where Claude Code stores usage data
  codex:
    enabled: true
    data_path: "~/.codex"

budget:
  weekly_tokens: 700000        # Fallback if provider doesn't expose limits
  per_provider:
    claude: 700000
    codex: 500000
```

### Multi-Project Setup

```yaml
# In global config (~/.config/nightshift/config.yaml)
projects:
  - path: ~/code/project1
    priority: 1                # Higher priority = processed first
    tasks:
      - lint
      - docs
  - path: ~/code/project2
    priority: 2

  # Or use glob patterns
  - pattern: ~/code/oss/*
    exclude:
      - ~/code/oss/archived
```

---

## Usage

### Manual Execution

```bash
# Run all enabled tasks
nightshift run

# Dry-run (show what would happen)
nightshift run --dry-run

# Run for specific project
nightshift run --project ~/code/myproject

# Run specific task type
nightshift run --task lint
```

### Daemon Mode

Run Nightshift as a background daemon that executes on schedule:

```bash
# Start the daemon
nightshift daemon start

# Check status
nightshift daemon status

# Stop the daemon
nightshift daemon stop

# Run in foreground (for debugging)
nightshift daemon start --foreground
```

### System Service

For automatic overnight runs, install as a system service:

```bash
# macOS (launchd)
nightshift install launchd

# Linux (systemd)
nightshift install systemd

# Any platform (cron)
nightshift install cron

# Remove the service
nightshift uninstall
```

---

## Monitoring & Visibility

### Check Run History

```bash
# Show last 5 runs
nightshift status

# Show last 10 runs
nightshift status --last 10

# Today's summary
nightshift status --today
```

Example output:

```
Last 5 Runs
===========

2024-01-15 02:30  myproject     3 tasks  45.2K tokens  completed
2024-01-14 02:30  myproject     2 tasks  23.1K tokens  completed
2024-01-14 02:35  library       1 task   12.4K tokens  completed
```

### View Logs

```bash
# Show recent logs
nightshift logs

# Show last 100 lines
nightshift logs --tail 100

# Stream logs in real-time
nightshift logs --follow

# Export logs to file
nightshift logs --export nightshift-logs.txt
```

### Check Budget

```bash
# Show all providers
nightshift budget

# Show specific provider
nightshift budget --provider claude
```

Example output:

```
Budget Status (mode: daily)
================================

[claude]
  Weekly:       700.0K tokens
  Daily:        100.0K tokens
  Used today:   45.2K (45.2%)
  Remaining:    54.8K tokens
  Reserve:      5.0K tokens
  Nightshift:   4.5K tokens available
  Progress:     [##############----------------] 45.2%
```

### Morning Summary

After each run, Nightshift generates a summary at:
`~/.local/share/nightshift/summaries/nightshift-YYYY-MM-DD.md`

```markdown
# Nightshift Summary - 2024-01-15

## Budget
- Started with: 100,000 tokens
- Used: 45,234 tokens (45%)
- Remaining: 54,766 tokens

## Tasks Completed
- [PR #123] Fixed 12 linting issues in myproject
- [Report] Found 3 dead code blocks in library

## What's Next?
- Review PR #123 in myproject
- Consider removing dead code in library
```

### Interactive TUI

Launch the terminal UI for real-time monitoring:

```go
// From your code, import and use:
import "github.com/marcusvorwaller/nightshift/internal/ui"

model := ui.New()
model.Run()
```

The TUI shows three panels:
- **Status**: Daemon state, current task, budget usage
- **Tasks**: Recent/queued tasks with status indicators
- **Logs**: Scrollable log viewer

Navigation:
- `Tab` / `Shift+Tab`: Switch panels
- `j`/`k` or arrows: Scroll
- `q`: Quit

---

## Integrations

### claude.md / agents.md

Nightshift reads these files from your project root to understand context:

- **claude.md**: Project description, coding conventions, task hints
- **AGENTS.md**: Agent behavior preferences, tool restrictions

Tasks mentioned in these files get a priority bonus (+2).

### GitHub Issues

Label issues with `nightshift` to have them picked up:

```yaml
integrations:
  task_sources:
    - github_issues
```

Nightshift will:
1. Fetch issues with the `nightshift` label
2. Process them as tasks (priority bonus +3)
3. Comment progress on the issue

### td Task Management

Integrate with [td](https://github.com/anthropics/td) for task tracking:

```yaml
integrations:
  task_sources:
    - td:
        enabled: true
        teach_agent: true   # Include td usage in agent prompts
```

---

## Troubleshooting

### Common Issues

**"No config file found"**
```bash
nightshift init           # Create project config
nightshift init --global  # Create global config
```

**"Insufficient budget"**
- Check current budget: `nightshift budget`
- Increase `max_percent` in config
- Wait for budget reset (check reset time in output)

**"Provider not available"**
- Ensure Claude/Codex CLI is installed and in PATH
- Check API key environment variables are set

### Debug Mode

Enable verbose logging:

```bash
nightshift run --verbose
```

Or set log level in config:

```yaml
logging:
  level: debug    # debug | info | warn | error
```

### Safe Defaults

Nightshift includes safety features:

| Feature | Default | Override |
|---------|---------|----------|
| Read-only first run | Yes | `--enable-writes` |
| Max budget per run | 10% | `budget.max_percent` |
| Auto-push to remote | No | Manual only |
| Reserve budget | 5% | `budget.reserve_percent` |

### File Locations

| Type | Location |
|------|----------|
| Run logs | `~/.local/share/nightshift/logs/nightshift-YYYY-MM-DD.log` |
| Audit logs | `~/.local/share/nightshift/audit/audit-YYYY-MM-DD.jsonl` |
| Summaries | `~/.local/share/nightshift/summaries/` |
| State | `~/.local/share/nightshift/state/state.json` |
| PID file | `~/.local/share/nightshift/nightshift.pid` |

### Getting Help

```bash
nightshift --help
nightshift <command> --help
```

Report issues: https://github.com/marcusvorwaller/nightshift/issues
