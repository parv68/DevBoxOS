# DevBoxOS

**One command. Any project. Everywhere.**

DevBoxOS is a local-first development sandbox platform for reproducible multi-service development environments. Define your stack in `devbox.yml`, run `devbox start`, and DevBoxOS builds, starts, networks, secures, observes, snapshots, and tears down the environment on your machine.

DevBoxOS v1 is intentionally **local-only**. It does not require a cloud account, hosted control plane, SaaS backend, or remote runner.

---

## Table Of Contents

1. [Project Status](#project-status)
2. [What DevBoxOS Does](#what-devboxos-does)
3. [Supported Platforms](#supported-platforms)
4. [Architecture](#architecture)
5. [Installation](#installation)
6. [Quickstart](#quickstart)
7. [Configuration](#configuration)
8. [CLI Reference](#cli-reference)
9. [Snapshots](#snapshots)
10. [Secrets](#secrets)
11. [Logs](#logs)
12. [Plugins](#plugins)
13. [Cross-Platform Behavior](#cross-platform-behavior)
14. [Diagnostics And Troubleshooting](#diagnostics-and-troubleshooting)
15. [Error Codes](#error-codes)
16. [Development](#development)
17. [Release Checklist](#release-checklist)
18. [Roadmap](#roadmap)
19. [Contributing](#contributing)
20. [License](#license)

---

## Project Status

**Local v1 status: feature-complete release candidate.**

The local-only version includes the complete core workflow:

- Project detection and `devbox.yml` generation
- Config parsing and validation
- Docker-backed service orchestration
- Multi-service start/stop/status/reset
- Docker image builds
- Local service discovery labels and network setup
- Local mTLS certificate generation
- Encrypted local secrets
- Persistent logs, search, export, and rotation support
- Snapshots with volume export/import
- Diagnostics through `devbox doctor`
- Plugin hooks
- Docker Compose import
- Shell completions
- Cross-platform IPC paths for Windows, macOS, and Linux
- Distribution workflow and install script scaffolding

Before tagging a public `v1.0.0`, the recommended final validation is:

- Run smoke tests on Windows, macOS, and Linux machines.
- Verify Docker Desktop on Windows/macOS and Docker Engine on Linux.
- Run the release workflow and validate produced assets.
- Test install, upgrade, and uninstall behavior from release artifacts.

---

## What DevBoxOS Does

DevBoxOS solves local environment drift and onboarding friction by turning project setup into a reproducible command sequence.

With DevBoxOS, a developer can run:

```bash
git clone https://github.com/parv68/DevBoxOS
cd devbox
devbox start
```

And get:

- Required services running in containers
- Buildable services built from Dockerfiles
- Isolated project networking
- Consistent ports and service names
- Local encrypted secrets
- Persistent searchable logs
- Snapshot save/load for environment state
- Diagnostics for Docker, ports, config, resources, and security
- Plugin hooks for project-specific workflows

DevBoxOS is not a replacement for Docker. It is a developer-experience layer above Docker focused on consistent local environments.

---

## Supported Platforms

DevBoxOS includes platform-level support for:

| OS | CLI | Engine IPC | Docker Runtime | Status |
|---|---:|---|---|---|
| Windows | Yes | TCP `127.0.0.1:51000` | Docker Desktop named pipe path support | Implemented, locally smoke-tested in this workspace |
| macOS | Yes | Unix socket `~/.devbox/engine.sock` | `/var/run/docker.sock` | Implemented, needs final device smoke test before GA |
| Linux | Yes | Unix socket `~/.devbox/engine.sock` | `/var/run/docker.sock` | Implemented, needs final distro smoke test before GA |

Cross-platform behavior lives primarily in:

- `shared/platform/platform.go`
- `engine/cmd/daemon.go`
- `cli/internal/client/grpc_client.go`
- `shared/runtime/docker/client.go`

---

## Architecture

DevBoxOS is a local CLI plus local engine daemon.

```text
+----------------+         local gRPC          +--------------------+
| devbox CLI     |  <------------------------> | devbox-engine      |
| user commands  |                             | orchestration      |
+----------------+                             +---------+----------+
                                                            |
                                                            | Docker API
                                                            v
                                                  +------------------+
                                                  | Docker daemon    |
                                                  | containers, nets |
                                                  +------------------+
```

### Components

| Component | Path | Purpose |
|---|---|---|
| CLI | `cli/` | User-facing commands |
| Engine | `engine/` | Local daemon and gRPC service |
| Shared packages | `shared/` | Config, platform, runtime, logs, snapshots, secrets |
| Install scripts | `scripts/` | Installer and release helpers |
| CI/release workflows | `.github/workflows/` | Test/build/release automation |

### Local IPC

| Platform | Engine Address |
|---|---|
| Windows | `127.0.0.1:51000` |
| macOS | `unix://~/.devbox/engine.sock` |
| Linux | `unix://~/.devbox/engine.sock` |

---

## Installation

### Prerequisites

- Docker installed and running
- Go only if building from source
- Windows: Docker Desktop
- macOS: Docker Desktop or compatible Docker daemon
- Linux: Docker Engine with user access to Docker socket

### Install From Release

Download the matching binaries from GitHub Releases:

- `devbox-<version>-windows-amd64.exe`
- `devbox-engine-<version>-windows-amd64.exe`
- `devbox-<version>-linux-amd64`
- `devbox-engine-<version>-linux-amd64`
- `devbox-<version>-darwin-amd64`
- `devbox-engine-<version>-darwin-amd64`
- `devbox-<version>-darwin-arm64`
- `devbox-engine-<version>-darwin-arm64`

Put both `devbox` and `devbox-engine` on your `PATH`.

### macOS/Linux Installer

```bash
curl -fsSL https://devbox.sh/install.sh | sh
```

Optional environment variables:

```bash
DEVBOX_VERSION=v1.0.0 curl -fsSL https://devbox.sh/install.sh | sh
DEVBOX_INSTALL_DIR=$HOME/.local/bin curl -fsSL https://devbox.sh/install.sh | sh
```

### Windows Manual Install

1. Download `devbox.exe` and `devbox-engine.exe`.
2. Move them to a directory on your `PATH`, for example `C:\Tools\DevBoxOS`.
3. Start Docker Desktop.
4. Open PowerShell and run:

```powershell
devbox version
devbox doctor
```

### Build From Source

```bash
git clone https://github.com/parv68/DevBoxOS
cd devboxos

go test ./shared/...
go test ./engine/...
go test ./cli/...

cd cli && go build -o ../dist/devbox .
cd ../engine && go build -o ../dist/devbox-engine ./cmd/daemon.go
```

On Windows:

```powershell
cd cli
go build -o ..\dist\devbox.exe .
cd ..\engine
go build -o ..\dist\devbox-engine.exe .\cmd\daemon.go
```

---

## Quickstart

Start the engine daemon in one terminal:

```bash
devbox-engine
```

In your project directory:

```bash
devbox init
devbox validate
devbox start
devbox status
devbox logs web
devbox exec web sh
devbox snapshot save --name before-change
devbox stop
```

For a Docker Compose project:

```bash
devbox init compose-import docker-compose.yml --output devbox.yml
devbox validate
devbox start
```

---

## Configuration

DevBoxOS uses `devbox.yml` in the project root.

### Minimal Example

```yaml
name: my-app
version: "1.0"

services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
    port: "3000"
    env:
      NODE_ENV: development
    volumes:
      - .:/app

  db:
    image: postgres:16
    env:
      POSTGRES_PASSWORD: devbox
    volumes:
      - db-data:/var/lib/postgresql/data
```

### Full Example

```yaml
name: full-stack-app
version: "1.0"

runtimes:
  node: "20"
  go: "1.24"

services:
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
      target: dev
      args:
        NODE_ENV: development
      tags:
        - full-stack-web:dev
    command: npm run dev
    working_dir: /app
    port: "3000"
    env:
      API_URL: http://api:8080
    env_file: .env
    volumes:
      - ./web:/app
    depends_on:
      - api
    healthcheck:
      type: http
      path: /health
      interval: 5s
      timeout: 2s
      retries: 10
    resources:
      memory: 512m
      cpu: "0.5"

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    command: go run ./cmd/api
    port: "8080"
    env:
      DATABASE_URL: postgres://postgres:devbox@db:5432/app?sslmode=disable
    depends_on:
      - db

  db:
    image: postgres:16
    port: "5432"
    env:
      POSTGRES_DB: app
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: devbox
    volumes:
      - db-data:/var/lib/postgresql/data

networking:
  discovery: true
  expose:
    - 3000
    - 8080
  egress: default-deny

security:
  tls: mTLS
  capabilities: minimal

plugins:
  - name: notify-ready
    command: echo "Environment ready"
    on:
      - post-start
    timeout: 10
```

### Top-Level Fields

| Field | Type | Required | Description |
|---|---|---:|---|
| `name` | string | Yes | Project/environment name |
| `version` | string | Recommended | Config schema/version marker |
| `runtimes` | map | No | Detected or declared runtime versions |
| `services` | map | Yes | Services to build/run |
| `networking` | object | No | Discovery, egress, exposed ports |
| `security` | object | No | TLS/capability configuration |
| `secrets` | object | No | Secret provider configuration |
| `plugins` | list | No | Lifecycle hooks |

### Service Fields

| Field | Type | Description |
|---|---|---|
| `image` | string | Prebuilt container image |
| `build` | object | Docker build configuration |
| `command` | string | Command to run in the container |
| `working_dir` | string | Working directory inside the container |
| `port` | string | Single service port |
| `ports` | list | Multiple port mappings |
| `env` | map | Environment variables |
| `env_file` | string | Environment file path |
| `volumes` | list | Volume or bind mounts |
| `depends_on` | list | Service dependencies |
| `healthcheck` | object | HTTP or command health checks |
| `resources` | object | CPU, memory, disk hints |
| `restart_policy` | object | Restart behavior |
| `security` | object | Per-service security settings |
| `secrets` | list | Secret references |

---

## CLI Reference

Run `devbox <command> --help` for live command help.

### Core Commands

| Command | Purpose |
|---|---|
| `devbox init` | Generate `devbox.yml` by scanning the project |
| `devbox init compose-import [file]` | Convert Docker Compose into DevBoxOS config |
| `devbox validate` | Validate `devbox.yml` |
| `devbox start` | Start all services |
| `devbox stop [service]` | Stop all services or one service |
| `devbox status` | Show environment status |
| `devbox reset` | Stop and rebuild environment from config |
| `devbox destroy [project]` | Remove DevBoxOS containers for a project |
| `devbox ps` | List active DevBoxOS projects/services |
| `devbox prune` | Remove orphaned DevBoxOS resources |
| `devbox exec <service> <cmd>` | Run a command inside a service container |
| `devbox doctor` | Run diagnostics |
| `devbox version` | Print version info |
| `devbox upgrade` | Upgrade from GitHub Releases |

### Build Commands

```bash
devbox build
devbox build web
devbox build --no-cache
devbox build --pull
```

| Flag | Description |
|---|---|
| `--no-cache` | Disable Docker build cache |
| `--pull` | Pull newer base images |

### Logs Commands

```bash
devbox logs web
devbox logs web --follow
devbox logs web --tail 200
devbox logs web --search "error"
devbox logs web --since 1h
devbox logs web --export web.log
```

| Flag | Description |
|---|---|
| `--follow`, `-f` | Stream live logs from the engine |
| `--tail <n>` | Show last N historical lines |
| `--search <pattern>` | Search persisted logs |
| `--since <duration>` | Filter historical logs by duration such as `1h` or `24h` |
| `--export <file>` | Export logs to a file |

### Snapshot Commands

```bash
devbox snapshot save --name before-refactor
devbox snapshot save --include-logs
devbox snapshot list
devbox snapshot load <snapshot-id>
devbox snapshot load <snapshot-id> --force
devbox snapshot delete <snapshot-id>
devbox snapshot export <snapshot-id> snapshot.tar.gz
devbox snapshot import snapshot.tar.gz
```

Snapshots include configuration, service metadata, volumes, and optionally logs. Volume export/import is implemented locally through Docker-backed helper containers.

### Secrets Commands

```bash
devbox secrets set DATABASE_PASSWORD supersecret
devbox secrets get DATABASE_PASSWORD
devbox secrets list
devbox secrets rotate DATABASE_PASSWORD
devbox secrets rm DATABASE_PASSWORD
```

Secrets are encrypted locally and stored under `.devbox/` by default.

| Flag | Description |
|---|---|
| `--key-path <path>` | Override encryption key path |
| `--store-path <path>` | Override encrypted store path |

### Config Commands

```bash
devbox config
devbox config telemetry false
devbox config telemetry
```

CLI configuration is stored locally in the platform config directory.

### Completion Commands

```bash
devbox completion bash
devbox completion zsh
devbox completion fish
devbox completion powershell
```

---

## Snapshots

Snapshots are stored locally in:

```text
<project>/.devbox/snapshots/
```

Each snapshot contains metadata and exported artifacts needed to restore local state.

Use cases:

- Save a known-good database state
- Reproduce bugs from a local environment
- Roll back after destructive testing
- Package local state for manual sharing

Current snapshot scope:

- Config metadata
- Service state metadata
- Docker volume export/import
- Optional logs
- Secret store copy/restore support

---

## Secrets

DevBoxOS local secrets are encrypted at rest.

Default files:

```text
.devbox/secrets.key
.devbox/secrets.enc
```

Recommended usage:

- Do not commit `.devbox/secrets.key`.
- Treat `.devbox/secrets.enc` as encrypted secret data.
- Use `devbox secrets rotate <name>` to regenerate generated secrets.
- Prefer environment-specific local secrets over committing `.env` files.

---

## Logs

DevBoxOS stores logs locally for search and export.

Features:

- Live streaming via engine
- Historical reads
- Search by pattern
- Export to file
- Rotation support in the logging store

Examples:

```bash
devbox logs api --follow
devbox logs api --search "connection refused"
devbox logs api --since 30m
devbox logs api --export api-debug.log
```

---

## Plugins

Plugins run commands at lifecycle hooks.

Example config:

```yaml
plugins:
  - name: notify-start
    command: echo "DevBoxOS environment started"
    on:
      - post-start
    timeout: 10
```

Supported hook examples:

- `pre-start`
- `post-start`
- `pre-stop`
- `post-stop`

Plugin commands are executed with DevBoxOS environment variables such as:

- `DEVBOX_HOOK`
- `DEVBOX_PROJECT`
- `DEVBOX_PLUGIN`

---

## Cross-Platform Behavior

Platform decisions are centralized in `shared/platform/platform.go`.

### Windows

- Engine listens on TCP: `127.0.0.1:51000`
- Docker socket uses Docker Desktop named pipe form: `npipe:////./pipe/docker_engine`
- Signal handling uses `os.Interrupt`
- Binaries use `.exe` suffix

### macOS

- Engine listens on Unix socket: `~/.devbox/engine.sock`
- Docker socket default: `/var/run/docker.sock`
- Signal handling uses `SIGINT` and `SIGTERM`

### Linux

- Engine listens on Unix socket: `~/.devbox/engine.sock`
- Docker socket default: `/var/run/docker.sock`
- Signal handling uses `SIGINT` and `SIGTERM`

### Important Note

The code paths for Windows, macOS, and Linux are implemented. Before a public `v1.0.0` release, run final smoke tests on actual machines for each platform because Docker Desktop and filesystem behavior differ by OS.

---

## Diagnostics And Troubleshooting

Run:

```bash
devbox doctor
```

`doctor` checks:

- Docker availability
- Config presence and validity
- Port conflicts
- Disk space
- Memory/resource hints
- Network readiness
- Security-related configuration
- Plugin/config problems

### Common Problems

#### Engine is not running

Start it:

```bash
devbox-engine
```

Then retry:

```bash
devbox status
```

#### Docker is not running

Windows/macOS:

```text
Start Docker Desktop.
```

Linux:

```bash
sudo systemctl start docker
```

#### Port already in use

Find and stop the conflicting process, or change the service port in `devbox.yml`.

#### Config not found

```bash
devbox init
devbox validate
```

#### Logs are empty

Make sure the service has started and logs have been collected:

```bash
devbox status
devbox logs <service> --follow
```

#### Snapshot load fails

Try:

```bash
devbox snapshot list
devbox snapshot load <snapshot-id> --force
```

---

## Error Codes

DevBoxOS uses structured local error categories in the CLI error package.

| Code | Meaning | Typical Fix |
|---|---|---|
| `DOCKER_NOT_RUNNING` | Docker daemon is unavailable | Start Docker Desktop or Docker Engine |
| `PORT_IN_USE` | A requested port is already occupied | Stop the conflicting process or change the port |
| `CONFIG_NOT_FOUND` | `devbox.yml` was not found | Run `devbox init` |
| `CONFIG_INVALID` | Config syntax or validation failed | Run `devbox validate` and fix reported issues |
| `ENGINE_NOT_RUNNING` | CLI cannot connect to engine | Start `devbox-engine` |
| `SERVICE_NOT_FOUND` | Named service does not exist | Check service name in `devbox.yml` |
| `NETWORK_ERROR` | Docker/network operation failed | Run `devbox doctor`, inspect Docker networks |
| `PERMISSION_DENIED` | File, socket, or Docker permission issue | Fix file permissions or Docker group access |
| `DISK_SPACE` | Insufficient disk space | Prune Docker resources or free disk space |
| `VERSION_MISMATCH` | CLI/engine versions differ | Upgrade both binaries together |
| `UNKNOWN` | Unclassified error | Re-run with `devbox doctor` and open an issue |

---

## Development

### Repository Layout

```text
cli/                 DevBoxOS CLI
engine/              Local engine daemon and gRPC service
shared/              Shared libraries
scripts/             Install/release scripts
.github/workflows/   CI and release workflows
dist/                Built binaries
```

### Build

```bash
cd cli && go build -o ../dist/devbox .
cd ../engine && go build -o ../dist/devbox-engine ./cmd/daemon.go
```

### Test

Because this repository uses a Go workspace, run tests per module:

```bash
cd shared && go test ./...
cd ../engine && go test ./...
cd ../cli && go test ./...
```

### Current Test Coverage

Implemented tests include:

- Config parser tests
- Platform detection/path tests
- Secrets tests
- Dependency graph/topological sort tests

Additional useful tests before GA:

- Docker runtime integration tests
- Snapshot round-trip tests with real volumes
- Engine gRPC end-to-end tests
- Windows/macOS/Linux smoke tests
- CLI command snapshot tests

---

## Release Checklist

Before publishing a release:

1. Run all tests:

```bash
cd shared && go test ./...
cd ../engine && go test ./...
cd ../cli && go test ./...
```

2. Build local binaries:

```bash
cd cli && go build -ldflags="-s -w" -o ../dist/devbox .
cd ../engine && go build -ldflags="-s -w" -o ../dist/devbox-engine ./cmd/daemon.go
```

3. Smoke test:

```bash
devbox-engine
devbox validate
devbox start
devbox status
devbox logs <service>
devbox exec <service> sh
devbox snapshot save --name smoke-test
devbox stop
```

4. Run platform smoke tests:

- Windows with Docker Desktop
- macOS Intel and/or Apple Silicon with Docker Desktop
- Linux with Docker Engine

5. Publish release artifacts with checksums.

---

## Roadmap

### Local v1

Implemented in this repository:

- Local CLI
- Local engine
- Docker orchestration
- Local config/secrets/logs/snapshots/plugins/diagnostics
- Cross-platform local IPC
- Local distribution scaffolding

### Not In Local v1

These are intentionally not part of the local-only release:

- Hosted cloud backend
- Team workspace SaaS
- Remote compute environments
- Hosted snapshot sharing
- Billing/metering
- Enterprise SSO/RBAC/audit control plane

Those can be future phases, but the local v1 should remain useful without them.

---

## Contributing

Contributions are welcome.

Recommended workflow:

1. Fork the repository.
2. Create a branch.
3. Add tests for behavior changes.
4. Run all module tests.
5. Open a pull request with a clear description.

Useful commands:

```bash
go fmt ./...
cd shared && go test ./...
cd ../engine && go test ./...
cd ../cli && go test ./...
```

When reporting a bug, include:

- OS and version
- Docker version
- DevBoxOS CLI and engine versions
- `devbox doctor` output
- Relevant `devbox.yml`
- Exact command and error output

---

## License

MIT. See `LICENSE`.

---
