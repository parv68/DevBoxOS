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
curl -LO https://github.com/parv68/DevBoxOS/releases/latest/download/devboxos_linux_amd64.tar.gz
tar xzf devboxos_linux_amd64.tar.gz
sudo mv devboxos /usr/local/bin/
```

## Quick Start

```bash
# Initialize a new project
cd my-project
devboxos init

# Validate the generated config
devboxos validate

# Start all services (requires engine daemon)
devbox-engine &
devboxos start

# View running services
devboxos status

# View logs
devboxos logs web

# Stop everything
devboxos stop

# Run diagnostics
devboxos doctor
```

## CLI Commands

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devboxos init` | Generate devbox.yml by scanning project | No |
| `devboxos validate` | Validate devbox.yml configuration | No |
| `devboxos start` | Start all services | Yes |
| `devboxos stop [service]` | Stop services | Yes |
| `devboxos status` | Show environment status | Yes |
| `devboxos logs <service>` | View service logs | No (local) / Yes (follow) |
| `devboxos reset` | Tear down and rebuild | Yes |
| `devboxos doctor` | Run diagnostics | Yes (via engine) / No (fallback) |
| `devboxos build [service]` | Build service images | No |
| `devboxos destroy` | Remove all managed containers | No |
| `devboxos exec <service> <cmd>` | Execute command in a service | No |
| `devboxos ps` | List running projects and services | No |
| `devboxos prune` | Remove orphaned containers | No |
| `devboxos shell <service>` | Open interactive shell in a service container | No |
| `devboxos url` | Show accessible URLs for services with port mappings | No |
| `devboxos wait <service> [--timeout]` | Wait for services to become healthy | Yes |
| `devboxos cp <service>:<path> <local-path>` | Copy files to/from service containers | No |
| `devboxos env [service] [--reveal]` | Show environment variables (masked by default) | No |
| `devboxos graph` | Visualize service dependency graph as ASCII tree | No |
| `devboxos top [--interval]` | Real-time CPU/memory dashboard for all services | Yes |
| `devboxos push [service] [--tag --all]` | Push service images to a registry | Yes |
| `devboxos snapshot [save\|load\|list\|delete\|export\|import\|gc]` | Manage environment snapshots | No (fallback) |
| `devboxos secrets [set\|get\|list\|delete\|rotate]` | Manage encrypted secrets | No (fallback) |
| `devboxos init compose-import` | Import docker-compose.yml → devbox config | No |
| `devboxos init compose-export` | Export devbox config → docker-compose.yml | No |
| `devboxos upgrade` | Upgrade to latest version | No |
| `devboxos config` | View or set CLI configuration | No |
| `devboxos version` | Show version | No |
| `devboxos completion [bash\|zsh\|fish\|powershell]` | Generate shell completions | No |

### Notable Flags

| Flag | Description |
|------|-------------|
| `devboxos init --from-git <repo>` | Clone a repo and auto-detect project configuration |
| `devboxos init --template <name>` | Generate a project from a built-in template (react-express-postgres, go-api, python-django, node-express, rust-axum) |
| `devboxos start --watch` | Hot-reload services when files change (fsnotify) |
| `devboxos build --no-cache` | Bypass Docker build cache |
| `devboxos build --pull` | Always pull base image before building |
| `devboxos snapshot export <id> <file>` | Export a snapshot to a tarball |
| `devboxos snapshot import <file>` | Import a snapshot from a tarball |
| `devboxos snapshot gc [--keep <n>] [--older-than <duration>]` | Garbage collect old snapshots |
| `devboxos env --reveal` | Show unmasked secret values |
| `devboxos wait --timeout <seconds>` | Custom health-check timeout |

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
devboxos config           # view all
devboxos config telemetry # view single key
devboxos config telemetry false  # set key=value
```

## Architecture

```
┌──────────────┐     gRPC     ┌──────────────┐     Docker API    ┌─────────┐
│ devboxos CLI  │ ──────────→ │  engine daemon│ ───────────────→ │  Docker  │
│  (cobra CLI)  │ ←────────── │  (daemon.go) │ ←─────────────── │         │
└──────────────┘             └──────────────┘                   └─────────┘
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

# Benchmarks
make test-bench

# Security tests
make test-security

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

## License

MIT
