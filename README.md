# DevBoxOS

Universal Development Sandbox — spin up fully configured, reproducible development environments with a single command.

## Installation

### From source

```bash
# Build the CLI
go build -o devboxos ./cli

# (Optional) Build the engine daemon
go build -o devbox-engine ./engine/cmd

# Move to PATH
mv devboxos /usr/local/bin/
```

### Using GoReleaser (tagged releases)

```bash
curl -LO https://github.com/devboxos/devboxos/releases/latest/download/devboxos_linux_amd64.tar.gz
tar xzf devboxos_linux_amd64.tar.gz
sudo mv devboxos /usr/local/bin/
```

## Quick Start

```bash
# Initialize a new project
cd my-project
devbox init

# Validate the generated config
devbox validate

# Start all services (requires engine daemon)
devbox-engine --daemon &
devbox start

# View running services
devbox status

# View logs
devbox logs web

# Stop everything
devbox stop

# Run diagnostics
devbox doctor
```

## CLI Commands

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox init` | Generate devbox.yml by scanning project | No |
| `devbox validate` | Validate devbox.yml configuration | No |
| `devbox start` | Start all services | Yes |
| `devbox stop [service]` | Stop services | Yes |
| `devbox status` | Show environment status | Yes |
| `devbox logs <service>` | View service logs | No (local) / Yes (follow) |
| `devbox reset` | Tear down and rebuild | Yes |
| `devbox doctor` | Run diagnostics | Yes (via engine) / No (fallback) |
| `devbox build [service]` | Build service images | No |
| `devbox destroy` | Remove all managed containers | No |
| `devbox exec <service> <cmd>` | Execute command in a service | No |
| `devbox ps` | List running projects and services | No |
| `devbox prune` | Remove orphaned containers | No |
| `devbox snapshot [save\|load\|list\|delete]` | Manage environment snapshots | No (fallback) |
| `devbox secrets [set\|get\|list\|delete\|rotate]` | Manage encrypted secrets | No (fallback) |
| `devbox init compose-import` | Import docker-compose.yml → devbox config | No |
| `devbox init compose-export` | Export devbox config → docker-compose.yml | No |
| `devbox upgrade` | Upgrade to latest version | No |
| `devbox config` | View or set CLI configuration | No |
| `devbox version` | Show version | No |
| `devbox completion [bash\|zsh\|fish\|powershell]` | Generate shell completions | No |

## Configuration

CLI configuration is stored in `~/.devboxos/config.json`:

```json
{
  "telemetry": "true",
  "engine": "auto"
}
```

View or modify with:

```bash
devbox config           # view all
devbox config telemetry # view single key
devbox config telemetry false  # set key=value
```

## Architecture

```
┌─────────────┐     gRPC     ┌──────────────┐     Docker API    ┌─────────┐
│  devbox CLI  │ ──────────→ │  engine daemon│ ───────────────→ │  Docker  │
│  (cobra CLI) │ ←────────── │  (daemon.go) │ ←─────────────── │         │
└─────────────┘             └──────────────┘                   └─────────┘
       │                            │
       │  (direct fallback)         │
       └────────────────────────────┘
```

Most commands prefer the engine daemon but fall back to direct Docker SDK calls when the engine isn't running.

## Development

### Prerequisites

- Go 1.22+
- Docker
- Protocol Buffers compiler (for proto changes)

### Build

```bash
# Build all modules
go build ./shared/...
go build ./engine/...
go build ./cli/...

# Build binaries
go build -o devboxos ./cli
go build -o devbox-engine ./engine/cmd
```

### Test

```bash
# Unit tests
make test

# With race detector
make test-race

# Integration tests (requires Docker)
make test-integration

# E2E smoke tests (requires CLI binary)
make test-e2e-short

# Full E2E (requires Docker)
make test-e2e

# Code coverage
make coverage
```

### Project Structure

```
shared/          — Cross-cutting packages (config, secrets, platform, runtime, etc.)
engine/          — Engine daemon (gRPC server)
  cmd/           —   Entry point + gRPC handlers
  internal/      —   Private engine packages (networking, orchestrator, state)
  proto/         —   gRPC proto definitions + generated code
cli/             — CLI (cobra-based)
  cmd/           —   Command implementations
  internal/      —   gRPC client, output, autodetect
tests/           — E2E smoke tests (tagged e2e)
```

### Proto changes

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  engine/proto/engine.proto
```

## Additional CLI Commands (v0.2+)

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox shell <service>` | Open interactive shell in a service container | No |
| `devbox url` | Show accessible URLs for services with port mappings | No |
| `devbox wait <service> [--timeout]` | Wait for services to become healthy | Yes |
| `devbox cp <service>:<path> <local-path>` | Copy files to/from service containers | No |
| `devbox env [service] [--reveal]` | Show environment variables (masked by default) | No |
| `devbox graph` | Visualize service dependency graph as ASCII tree | No |
| `devbox snapshot gc` | Garbage collect old snapshots by age or count | No |
| `devbox push [service] [--tag --all]` | Push service images to a registry | Yes |
| `devbox top [--interval]` | Real-time CPU/memory dashboard for all services | Yes |

### Notable New Flags

| Flag | Description |
|------|-------------|
| `devbox init --from-git <repo>` | Clone a repo and auto-detect project configuration |
| `devbox init --template <name>` | Generate a project from a built-in template (react-express-postgres, go-api, python-django, node-express, rust-axum) |
| `devbox start --watch` | Hot-reload services when files change (fsnotify) |
| `devbox build --no-cache` | Bypass Docker build cache |
| `devbox build --pull` | Always pull base image before building |
| `devbox snapshot export` | Export a snapshot to a tarball |
| `devbox snapshot import` | Import a snapshot from a tarball |
| `devbox env --reveal` | Show unmasked secret values |
| `devbox wait --timeout <seconds>` | Custom health-check timeout |

## License

MIT
