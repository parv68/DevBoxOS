# DevBoxOS

**Universal Development Sandbox** — spin up fully configured, reproducible development environments with a single command.

[![CI](https://github.com/parv68/DevBoxOS/actions/workflows/ci.yml/badge.svg)](https://github.com/parv68/DevBoxOS/actions/workflows/ci.yml)
[![Release](https://github.com/parv68/DevBoxOS/actions/workflows/release.yml/badge.svg)](https://github.com/parv68/DevBoxOS/actions/workflows/release.yml)
![Go Version](https://img.shields.io/badge/go-1.25%2B-blue)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)

---

## What is DevBoxOS?

DevBoxOS is a local-first development environment manager. It lets you define your project's services (databases, caches, web servers, etc.) in a single `devbox.yml` file, then spin them all up with one command — no Docker Compose, no Kubernetes, no cloud dependencies.

```bash
# One command to start your entire dev environment
devbox start
```

### Why DevBoxOS?

| Problem | DevBoxOS Solution |
|---------|-------------------|
| "It works on my machine" | Reproducible environments from a single config file |
| "I need 5 terminals to run my stack" | One command starts everything |
| "Docker Compose is verbose" | Smart defaults, auto-detection, simpler syntax |
| "My services depend on each other" | Automatic dependency resolution with health checks |
| "I need to snapshot my DB state" | Built-in snapshot save/load/export/import |
| "Secrets in env files leak" | Encrypted secrets with age (X25519 + ChaCha20-Poly1305) |

---

## Features

- ✅ **Zero cloud** — Everything runs locally via Docker
- ✅ **Single config** — `devbox.yml` defines your entire stack
- ✅ **Dependency resolution** — Services start in the right order
- ✅ **Health checks** — Waits for services to be ready
- ✅ **Hot reload** — Watch files and auto-restart services
- ✅ **Snapshots** — Save, export, import, and restore environment state
- ✅ **Encrypted secrets** — age-encrypted secrets stored in your project
- ✅ **Shell access** — Interactive shell into any service container
- ✅ **File copy** — Copy files to/from containers (`devbox cp`)
- ✅ **Resource monitoring** — Real-time CPU/memory dashboard
- ✅ **Log management** — Persistent logs with search, tail, and rotation
- ✅ **Diagnostics** — Comprehensive health checks (`devbox doctor`)
- ✅ **Plugin system** — Hook-based lifecycle plugins
- ✅ **Cross-platform** — Windows, macOS, Linux
- ✅ **Auto-detection** — Scan a project and generate config automatically
- ✅ **Docker Compose import/export** — Migrate between formats
- ✅ **No daemon dependency** — Most commands fall back to direct Docker SDK calls

---

## Architecture

```
┌────────────────┐     gRPC (TCP)     ┌────────────────┐     Docker API     ┌──────────┐
│  devbox CLI  │ ─────────────────→ │  Engine Daemon │ ────────────────→ │  Docker  │
│  (cobra CLI)   │ ←──────────────── │  (daemon.go)   │ ←──────────────── │  Engine  │
└────────────────┘                   └────────────────┘                   └──────────┘
       │                                      │
       │  (direct Docker SDK fallback)         │
       └──────────────────────────────────────┘
```

The CLI talks to the engine daemon via gRPC over TCP (`127.0.0.1:51000`). If the engine isn't running, most commands fall back to calling the Docker SDK directly.

### Project Structure

```
devbox/
├── cli/               # CLI (cobra-based)
│   ├── cmd/           #   Command implementations
│   └── internal/      #   gRPC client, output, autodetect
├── engine/            # Engine daemon (gRPC server)
│   ├── cmd/           #   Entry point + gRPC handlers
│   ├── internal/      #   Networking, orchestrator, state
│   └── proto/         #   gRPC proto definitions
├── shared/            # Cross-cutting packages
│   ├── config/        #   Config parsing, validation, auto-detection
│   ├── diagnostics/   #   Health checks
│   ├── logging/       #   Persistent log storage
│   ├── platform/      #   OS detection
│   ├── plugins/       #   Hook system
│   ├── runtime/       #   Docker SDK wrapper
│   ├── secrets/       #   age encryption
│   └── snapshot/      #   Environment snapshots
└── tests/             # E2E, benchmark, and security tests
```

---

## Installation

### Linux

**Option 1 — Download the release archive**

```bash
# Download the latest release
curl -LO https://github.com/parv68/DevBoxOS/releases/latest/download/devbox_linux_amd64.tar.gz

# Extract
tar xzf devbox_linux_amd64.tar.gz

# Install
sudo mv devbox devbox-engine /usr/local/bin/
```

**Option 2 — Install script**

```bash
curl -fsSL https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.sh | sh
```

**Option 3 — Build from source**

```bash
# Prerequisites: Go 1.25+, Docker
git clone https://github.com/parv68/DevBoxOS.git
cd DevBoxOS

# Build CLI and engine
go build -o devbox ./cli
go build -o devbox-engine ./engine/cmd

# Install
sudo mv devbox devbox-engine /usr/local/bin/
```

### macOS

**Option 1 — Download the release archive (Intel)**

```bash
curl -LO https://github.com/parv68/DevBoxOS/releases/latest/download/devbox_darwin_amd64.tar.gz
tar xzf devbox_darwin_amd64.tar.gz
sudo mv devbox devbox-engine /usr/local/bin/
```

**Option 1 — Download the release archive (Apple Silicon)**

```bash
curl -LO https://github.com/parv68/DevBoxOS/releases/latest/download/devbox_darwin_arm64.tar.gz
tar xzf devbox_darwin_arm64.tar.gz
sudo mv devbox devbox-engine /usr/local/bin/
```

**Option 2 — Install script**

```bash
curl -fsSL https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.sh | sh
```

**Option 3 — Build from source**

```bash
git clone https://github.com/parv68/DevBoxOS.git
cd DevBoxOS
go build -o devbox ./cli
go build -o devbox-engine ./engine/cmd
sudo mv devbox devbox-engine /usr/local/bin/
```

### Windows

**Option 1 — Install script (recommended)**

One command — downloads, extracts, adds to PATH permanently.

PowerShell:
```powershell
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.ps1'))
```

cmd.exe:
```cmd
powershell -c "iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.ps1'))"
```

After installation, close and reopen your terminal — `devbox` works from anywhere.

**Option 2 — Download from GitHub Releases (manual)**

PowerShell:
```powershell
# Download the latest release
Invoke-WebRequest -Uri "https://github.com/parv68/DevBoxOS/releases/latest/download/devbox_windows_amd64.zip" -OutFile "devbox.zip"

# Extract to a permanent location
$installDir = "$env:LOCALAPPDATA\DevBoxOS"
Expand-Archive -Path "devbox.zip" -DestinationPath $installDir -Force

# Add to PATH permanently
[Environment]::SetEnvironmentVariable("Path", "$installDir;$env:Path", [EnvironmentVariableTarget]::User)

# Verify
devbox version
```

cmd.exe:
```cmd
REM Download
powershell -c "Invoke-WebRequest -Uri 'https://github.com/parv68/DevBoxOS/releases/latest/download/devbox_windows_amd64.zip' -OutFile 'devbox.zip'"

REM Extract
powershell -c "Expand-Archive -Path 'devbox.zip' -DestinationPath '%LOCALAPPDATA%\DevBoxOS' -Force"

REM Add to PATH permanently
setx PATH "%PATH%;%LOCALAPPDATA%\DevBoxOS"

REM Close and reopen your terminal, then verify
devbox version
```

**Option 3 — Build from source**

```powershell
git clone https://github.com/parv68/DevBoxOS.git
cd DevBoxOS
go build -o devbox.exe ./cli
go build -o devbox-engine.exe ./engine/cmd
```

All methods install two binaries: `devbox.exe` (CLI) and `devbox-engine.exe` (daemon). After installation, start the engine and verify:

```powershell
devbox-engine
devbox version
```

### Prerequisites

- **Docker** — DevBoxOS manages Docker containers. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) for your platform.
- **Go 1.25+** — Only needed if building from source.

---

## Quick Start

### 1. Start the engine daemon

```bash
# Start the engine daemon (auto-launched on first `devbox start`)
devbox engine start

# Check status
devbox engine status   # (coming soon)
```

The engine runs as a background process and is automatically started when you run `devbox start` if it's not already running. You can manage it explicitly with `devbox engine {start|stop|restart}`.

### 2. Initialize a project

```bash
cd my-project

# Auto-detect and generate devbox.yml
devbox init

# Or use a template
devbox init --template react-express-postgres

# Or clone a repo and auto-detect
devbox init --from-git https://github.com/user/repo.git
```

### 3. Validate your config

```bash
devbox validate
```

### 4. Start all services

```bash
devbox start
```

### 5. Check status

```bash
devbox status
```

### 6. View logs

```bash
devbox logs web
devbox logs web --tail 50
devbox logs web --follow
```

### 7. Stop everything

```bash
devbox stop
```

---

## CLI Commands

### Command Reference

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox init` | Generate devbox.yml by scanning project | No |
| `devbox init --from-git <repo>` | Clone a repo and auto-detect configuration | No |
| `devbox init --template <name>` | Generate a project from a built-in template | No |
| `devbox init compose-import` | Import docker-compose.yml → devbox.yml | No |
| `devbox init compose-export` | Export devbox.yml → docker-compose.yml | No |
| `devbox validate` | Validate devbox.yml configuration | No |
| `devbox start` | Start all services | Yes |
| `devbox start --watch` | Start with hot-reload on file changes | Yes |
| `devbox stop [service]` | Stop all services or a specific service | Yes |
| `devbox status` | Show environment status and service health | Yes |
| `devbox ps` | List running projects and services | No |
| `devbox logs <service>` | View service logs | No (local) / Yes (follow) |
| `devbox logs <service> --follow` | Stream logs in real-time | Yes |
| `devbox logs <service> --tail <n>` | Show last N log lines | No |
| `devbox logs <service> --search <pattern>` | Search logs with regex | No |
| `devbox logs <service> --export <file>` | Export logs to file | No |
| `devbox build [service]` | Build service images | No |
| `devbox build --no-cache` | Bypass Docker build cache | No |
| `devbox build --pull` | Always pull base images | No |
| `devbox exec <service> <cmd>` | Execute a command in a service container | No |
| `devbox shell <service>` | Open an interactive shell in a container | No |
| `devbox cp <service>:<path> <local-path>` | Copy files from a container to local | No |
| `devbox cp <local-path> <service>:<path>` | Copy files from local to a container | No |
| `devbox env [service]` | Show environment variables (masked) | No |
| `devbox env [service] --reveal` | Show environment variables (unmasked) | No |
| `devbox url` | Show accessible URLs for services with ports | No |
| `devbox graph` | Visualize service dependency graph | No |
| `devbox doctor` | Run system diagnostics | Yes (engine) / No (fallback) |
| `devbox reset` | Tear down and rebuild all services | Yes |
| `devbox destroy` | Remove all managed containers and networks | No |
| `devbox prune` | Remove orphaned containers | No |
| `devbox push [service]` | Push service images to a registry | Yes |
| `devbox push --tag <tag>` | Tag image before pushing | Yes |
| `devbox push --all` | Push all service images | Yes |
| `devbox top` | Real-time CPU/memory dashboard | Yes |
| `devbox top --interval <sec>` | Custom refresh interval | Yes |
| `devbox wait <service>` | Wait for a service to become healthy | Yes |
| `devbox wait --timeout <sec>` | Custom health-check timeout | Yes |
| `devbox engine start` | Start the engine daemon (if not running) | No |
| `devbox engine stop` | Gracefully stop the engine daemon | Yes |
| `devbox engine restart` | Restart the engine daemon | Yes |

### Snapshot Commands

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox snapshot save <name>` | Create a snapshot of the current environment | No (fallback) |
| `devbox snapshot list` | List all snapshots | No (fallback) |
| `devbox snapshot load <id>` | Restore environment from a snapshot | No (fallback) |
| `devbox snapshot delete <id>` | Delete a snapshot | No (fallback) |
| `devbox snapshot export <id> <file>` | Export a snapshot to a tarball | No (fallback) |
| `devbox snapshot import <file>` | Import a snapshot from a tarball | No (fallback) |
| `devbox snapshot gc` | Garbage collect old snapshots | No (fallback) |
| `devbox snapshot gc --keep <n>` | Keep only the N most recent snapshots | No (fallback) |
| `devbox snapshot gc --older-than <duration>` | Remove snapshots older than duration | No (fallback) |

### Secrets Commands

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox secrets set <key> <value>` | Set an encrypted secret | No (fallback) |
| `devbox secrets get <key>` | Retrieve a secret value | No (fallback) |
| `devbox secrets list` | List all secret keys | No (fallback) |
| `devbox secrets delete <key>` | Delete a secret | No (fallback) |
| `devbox secrets rotate` | Re-encrypt all secrets with a new key | No (fallback) |

### Utility Commands

| Command | Description | Engine Required |
|---------|-------------|----------------|
| `devbox config` | View all configuration | No |
| `devbox config <key>` | View a single config value | No |
| `devbox config <key> <value>` | Set a config value | No |
| `devbox version` | Show version and build info | No |
| `devbox upgrade` | Upgrade to the latest version | No |
| `devbox completion bash` | Generate bash completion script | No |
| `devbox completion zsh` | Generate zsh completion script | No |
| `devbox completion fish` | Generate fish completion script | No |
| `devbox completion powershell` | Generate PowerShell completion script | No |

---

## Detailed Command Guide

### `devbox init` — Project Initialization

```bash
# Auto-detect project type and generate devbox.yml
cd my-project
devbox init

# Use a built-in template
devbox init --template react-express-postgres
devbox init --template go-api
devbox init --template python-django
devbox init --template node-express
devbox init --template rust-axum

# Clone a repo and auto-detect
devbox init --from-git https://github.com/user/project.git

# Import from Docker Compose
devbox init compose-import

# Export to Docker Compose
devbox init compose-export
```

### `devbox start` — Starting Services

```bash
# Start all services with dependency resolution
devbox start

# Start with file watching (hot reload)
devbox start --watch
```

When you run `devbox start`, the engine:
1. Resolves the dependency graph
2. Builds any images with `build` config
3. Creates the isolated Docker network
4. Starts services in dependency order
5. Waits for health checks to pass
6. Attaches log collectors

### `devbox snapshot` — Environment Snapshots

Snapshots let you save and restore your entire development environment state, including container images, volumes, networks, and secrets.

```bash
# Save the current state
devbox snapshot save my-db-state

# List all snapshots
devbox snapshot list

# Export a snapshot to share with your team
devbox snapshot export abc12345 ./snapshot.tar

# Import a snapshot from a teammate
devbox snapshot import ./snapshot.tar

# Restore environment to a snapshot
devbox snapshot load abc12345

# Delete old snapshots
devbox snapshot gc --keep 5
devbox snapshot gc --older-than 7d

# Delete a specific snapshot
devbox snapshot delete abc12345
```

### `devbox secrets` — Encrypted Secrets

Secrets are encrypted with [age](https://age-encryption.org/) (X25519 + ChaCha20-Poly1305) and stored in `.devbox/secrets.enc` in your project directory.

```bash
# Set a secret
devbox secrets set DATABASE_URL "postgres://user:pass@db:5432/myapp"

# Get a secret (masked by default)
devbox secrets get DATABASE_URL

# Get a secret (unmasked)
devbox secrets get DATABASE_URL --reveal

# List all secrets
devbox secrets list

# Delete a secret
devbox secrets delete DATABASE_URL

# Re-encrypt all secrets with a new key
devbox secrets rotate
```

### `devbox shell` — Interactive Shell Access

Open a shell inside any running service container:

```bash
# Open a bash shell
devbox shell web

# Open a specific shell
devbox shell redis -- /bin/sh
```

### `devbox cp` — File Copy

Copy files between your local machine and service containers:

```bash
# Copy from container to local
devbox cp web:/etc/nginx/nginx.conf ./nginx.conf

# Copy from local to container
devbox cp ./config.json web:/app/config.json

# Copy directories
devbox cp web:/app/logs/ ./logs/
```

### `devbox top` — Resource Monitor

Real-time CPU and memory dashboard for all running services:

```bash
# Start the dashboard
devbox top

# Custom refresh interval (default: 2s)
devbox top --interval 5
```

### `devbox logs` — Log Management

```bash
# View logs for a service
devbox logs web

# Follow logs in real-time
devbox logs web --follow

# Show last 100 lines
devbox logs web --tail 100

# Search logs with regex
devbox logs web --search "error|panic"

# Export logs to a file
devbox logs web --export ./web-logs.txt

# Filter by time range
devbox logs web --since 2024-01-01T00:00:00Z
```

### `devbox doctor` — Diagnostics

Run comprehensive system checks:

```bash
devbox doctor
```

Checks performed:
- Docker daemon accessibility
- Disk space
- Memory availability
- Config file validity
- Secrets encryption integrity
- Network configuration
- Container health
- Orphaned resource detection
- Circular dependency detection
- Port conflict detection

### `devbox env` — Environment Variables

```bash
# Show all variables (values masked)
devbox env

# Show variables for a specific service
devbox env web

# Show unmasked values
devbox env --reveal
```

### `devbox graph` — Dependency Graph

Visualize your service dependencies as an ASCII tree:

```bash
devbox graph
```

Example output:
```
web
├── api
│   ├── redis
│   └── postgres
└── nginx
```

### `devbox upgrade` — Self-Upgrade

```bash
# Check for and install the latest version
devbox upgrade
```

---

## Configuration

### `devbox.yml` Reference

A minimal `devbox.yml`:

```yaml
name: my-project
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  redis:
    image: redis:7-alpine
```

A complete `devbox.yml`:

```yaml
name: my-full-project
version: "1.0"

services:
  web:
    image: nginx:alpine
    build:
      context: ./web
      dockerfile: Dockerfile
      args:
        NODE_ENV: development
    ports:
      - "8080:80"
    env:
      - NODE_ENV=development
      - DATABASE_URL=${secrets.DATABASE_URL}
    volumes:
      - ./web:/app
      - web_data:/data
    depends_on:
      - api
      - redis
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 10s
      timeout: 5s
      retries: 3
    restart: unless-stopped
    resources:
      cpus: 0.5
      memory: 256m

  api:
    image: myapp-api:latest
    build:
      context: ./api
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    env:
      - REDIS_URL=redis://redis:6379
      - DB_URL=postgres://user:pass@postgres:5432/myapp
    depends_on:
      - redis
      - postgres
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s

  postgres:
    image: postgres:16-alpine
    env:
      - POSTGRES_DB=myapp
      - POSTGRES_PASSWORD=${secrets.DB_PASSWORD}
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 10s

volumes:
  web_data:
  redis_data:
  pg_data:
```

### CLI Configuration

CLI configuration is stored in `~/.devbox/config.json`:

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

### Engine Configuration

The engine accepts these flags:

```bash
devbox-engine --help

Options:
  --port 51000       # gRPC port (default: 51000)
  --host 127.0.0.1   # Listen address (default: 127.0.0.1)
```

---

## Templates

Built-in project templates:

| Template | Services |
|----------|----------|
| `react-express-postgres` | React frontend, Express API, PostgreSQL |
| `go-api` | Go API server |
| `python-django` | Django web app + PostgreSQL |
| `node-express` | Node.js Express app |
| `rust-axum` | Rust Axum web server |

```bash
devbox init --template go-api
```

---

## Development

### Prerequisites

- Go 1.25+
- Docker
- Protocol Buffers compiler (for proto changes)

### Build

```bash
# Build all modules
go build ./shared/...
go build ./engine/...
go build ./cli/...

# Build binaries
go build -o devbox ./cli
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

# E2E smoke tests (local-only, fast)
make test-e2e-short

# Full E2E suite (requires Docker + engine daemon)
make test-e2e

# Benchmarks
make test-bench

# Security tests
make test-security

# Code coverage
make coverage
```

### Proto Changes

If you modify the gRPC protocol:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  engine/proto/engine.proto
```

### Linting

```bash
make vet
make lint
```

---

## Comparison: DevBoxOS vs Alternatives

| Feature | DevBoxOS | Docker Compose | Tilt | DevContainer |
|---------|----------|---------------|------|-------------|
| Local-only | ✅ | ✅ | ✅ | ✅ |
| Single binary | ✅ | Requires Docker CLI | Requires Go/Node | VS Code extension |
| Dependency resolution | ✅ Automatic | ✅ Manual depends_on | ✅ Automatic | ❌ |
| Health checks | ✅ Built-in | ✅ Supported | ✅ Built-in | ❌ |
| Snapshots | ✅ Native | ❌ | ❌ | ❌ |
| Encrypted secrets | ✅ age-encrypted | ❌ Plain env files | ❌ | ❌ |
| Hot reload | ✅ Built-in | ❌ | ✅ Built-in | ❌ |
| Log management | ✅ Persistent + search | ❌ Raw docker logs | ✅ | ❌ |
| Diagnostics | ✅ Comprehensive | ❌ | ❌ | ❌ |
| Resource monitoring | ✅ Built-in | ❌ | ❌ | ❌ |
| Plugin system | ✅ Hook-based | ❌ | ❌ | ❌ |
| Cloud dependency | ❌ None | ❌ None | ❌ None | ❌ None |
| Config complexity | Low | Medium | High | Medium |

---

## FAQ

**Q: Do I need the engine daemon running all the time?**

A: No. Commands like `init`, `validate`, `build`, `exec`, `shell`, `cp`, `logs`, `graph`, `ps`, `prune`, `destroy`, `snapshot`, `secrets`, and `config` work without the engine by falling back to the Docker SDK directly. Only `start`, `stop`, `status`, `reset`, `wait`, `top`, and `engine stop`/`engine restart` require the engine. Use `devbox engine start` to launch it when needed.

**Q: Where is data stored?**

A: Project-level data is stored in `.devbox/` inside your project directory. This includes logs, snapshots, encrypted secrets, and SQLite state. CLI config is stored in `~/.devbox/config.json`.

**Q: Can I use DevBoxOS with existing Docker Compose projects?**

A: Yes. Use `devbox init compose-import` to convert your `docker-compose.yml` to `devbox.yml`, or use `devbox init compose-export` to go the other direction.

**Q: Is DevBoxOS production-ready?**

A: DevBoxOS is designed for local development environments, not production deployments. Use it to replace `docker-compose up` and manual container management during development.

**Q: Does DevBoxOS support Kubernetes?**

A: No. DevBoxOS is a local-only tool. It manages Docker containers directly, not Kubernetes pods.

**Q: How do I share my environment with teammates?**

A: Commit your `devbox.yml` and `.devbox/secrets.enc` (the encrypted secrets store). Your teammates clone the repo, install DevBoxOS, and run `devbox start`.

---

## Troubleshooting

**Engine won't start**

```bash
# Start the engine
devbox engine start

# If it's already running but unresponsive, restart it
devbox engine restart

# If gRPC is unresponsive, force-kill and restart
killall devbox-engine   # macOS/Linux
taskkill /f /im devbox-engine.exe   # Windows
devbox engine start

# Check if port 51000 is in use
netstat -an | findstr 51000   # Windows
lsof -i :51000                # macOS/Linux

# Check Docker is running
docker info
```

**Container fails to start**

```bash
# View logs
devbox logs <service>

# Run diagnostics
devbox doctor

# Rebuild the service
devbox build <service> --no-cache
devbox start
```

**Secret not working**

```bash
# Verify the secret exists
devbox secrets list

# Check the value (with --reveal)
devbox secrets get <key> --reveal

# Re-encrypt if needed
devbox secrets rotate
```

---

## Contributing

Contributions are welcome! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`make test && make vet`)
5. Commit with a descriptive message
6. Push and open a Pull Request

### Code Style

- Follow Go standard formatting (`gofmt`)
- Write tests for new features
- Update documentation for API changes
- Keep imports sorted (standard, third-party, internal)

### Reporting Issues

Report bugs and feature requests at [github.com/parv68/DevBoxOS/issues](https://github.com/parv68/DevBoxOS/issues).

---

## License

MIT — See [LICENSE](LICENSE) for details.

---

Built with ❤️ for developers who want their dev environment to just work.
