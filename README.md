# SSH Tunnel Manager

A terminal user interface (TUI) application to manage multiple SSH tunnels from a single configuration file.

## Features

- **Multiple tunnel types**: Local (-L), Remote (-R), and Dynamic SOCKS5 (-D) port forwarding
- **Jump host support**: Connect through bastion/jump hosts with chained jumps
- **Auto-reconnection**: Automatic reconnection with exponential backoff
- **SSH keep-alive**: Prevent connections from timing out
- **Tunnel groups**: Organize tunnels into groups for batch operations
- **Real-time stats**: Monitor connections, bytes transferred, and uptime
- **Interactive TUI**: Navigate and control tunnels with keyboard shortcuts

## Installation

### From source

```bash
go install github.com/labbs/sshtunnel/cmd/sshtunnel@latest
```

### Build from source

```bash
git clone https://github.com/labbs/sshtunnel.git
cd sshtunnel
go build -o sshtunnel ./cmd/sshtunnel
```

## Quick Start

1. Create a default configuration file:

```bash
sshtunnel --init
```

This creates `~/.config/sshtunnel/config.yaml` with example configuration.

2. Edit the configuration file with your hosts and tunnels.

3. Run the application:

```bash
sshtunnel
```

Or specify a custom config path:

```bash
sshtunnel --config /path/to/config.yaml
```

## Configuration

### Global settings

```yaml
global:
  log_level: info  # debug, info, warn, error
  log_file: ""     # Leave empty to disable file logging

  reconnect:
    enabled: true
    max_attempts: 5        # 0 = unlimited
    initial_delay: 1s
    max_delay: 60s
    backoff_multiplier: 2.0

  keep_alive:
    enabled: true
    interval: 30s
    timeout: 15s
```

### Hosts

Define reusable SSH host configurations:

```yaml
hosts:
  bastion:
    hostname: bastion.example.com
    port: 22
    user: deploy
    identity_file: ~/.ssh/id_ed25519
    agent_forwarding: false

  internal-server:
    hostname: internal.private
    port: 22
    user: admin
    identity_file: ~/.ssh/id_ed25519
    jump_host: bastion  # Connect through bastion first
```

### Tunnels

#### Local port forwarding (-L)

Forward a remote service to a local port:

```yaml
tunnels:
  - name: postgres-prod
    description: "PostgreSQL production database"
    host: bastion
    type: local
    local:
      address: 127.0.0.1
      port: 5432
    remote:
      address: db.internal
      port: 5432
    auto_start: true
```

#### Remote port forwarding (-R)

Expose a local service on the remote server:

```yaml
tunnels:
  - name: expose-metrics
    description: "Expose local Prometheus"
    host: bastion
    type: remote
    local:
      address: 127.0.0.1
      port: 9090
    remote:
      address: 0.0.0.0
      port: 9090
```

#### Dynamic SOCKS5 proxy (-D)

Create a SOCKS5 proxy through the SSH connection:

```yaml
tunnels:
  - name: socks-proxy
    description: "SOCKS5 proxy"
    host: bastion
    type: dynamic
    local:
      address: 127.0.0.1
      port: 1080
```

### Groups

Organize tunnels into groups:

```yaml
groups:
  - name: production
    description: "All production tunnels"
    tunnels:
      - postgres-prod
      - redis-prod
    auto_start: false
```

## Keyboard Shortcuts

### Main view

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `Enter/Space` | Toggle selected tunnel |
| `a` | Start all tunnels |
| `s` | Stop all tunnels |
| `g` | View groups |
| `r` | Reload configuration |
| `?` | Show help |
| `q` | Quit |

### Groups view

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `Enter/s` | Start group |
| `x` | Stop group |
| `Esc` | Back to main view |
| `q` | Quit |

## Project Structure

```
sshtunnel/
├── cmd/sshtunnel/         # Application entry point
├── internal/
│   ├── config/            # Configuration loading and validation
│   ├── format/            # Formatting utilities
│   ├── ssh/               # SSH client with jump host support
│   ├── tunnel/            # Tunnel implementations (local, remote, dynamic)
│   └── ui/                # Terminal UI (BubbleTea)
│       ├── components/    # Reusable UI components
│       ├── messages/      # Message types for UI events
│       ├── styles/        # Visual styling (lipgloss)
│       └── views/         # Screen implementations
└── config.example.yaml    # Example configuration
```

## Dependencies

- [BubbleTea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Viper](https://github.com/spf13/viper) - Configuration management
- [x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) - SSH client

## License

MIT
