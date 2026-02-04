# Night Shift

> Wake up to a cleaner codebase

A Go CLI that runs overnight to perform AI-powered maintenance tasks on your codebase, using your remaining daily token budget from Claude Code/Codex subscriptions.

## Features

- **Budget-aware**: Uses remaining daily allotment, never exceeds configurable max (default 10%)
- **Multi-project support**: Works across multiple repos
- **Configurable tasks**: From auto-PRs to analysis reports
- **Great DX**: Built with bubbletea/lipgloss for a delightful CLI experience

## Installation

```bash
go install github.com/marcusvorwaller/nightshift@latest
```

## Quick Start

```bash
# Initialize config in current directory
nightshift init

# Run maintenance tasks
nightshift run

# Check status of last run
nightshift status
```

## Configuration

Night Shift uses a YAML config file (`nightshift.yaml`) to define:

- Token budget limits
- Target repositories
- Task priorities
- Schedule preferences

See [SPEC.md](docs/SPEC.md) for detailed configuration options.

## License

MIT - see [LICENSE](LICENSE) for details.
