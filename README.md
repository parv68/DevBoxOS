# DevBoxOS

One command. Any project. Everywhere.

DevBoxOS is a local-first development sandbox: define your stack in `devbox.yml`, run `devbox start`, and get a reproducible multi-service dev environment backed by Docker.

## Status

Local-only v1 is implemented end-to-end.

Supported OS:
1. Windows
2. macOS
3. Linux

## Install

macOS/Linux:

```bash
curl -fsSL https://devbox.sh/install.sh | sh
```

Windows:
1. Download `devbox.exe` and `devbox-engine.exe` from GitHub Releases.
2. Put them on your `PATH`.

## Quickstart

```bash
devbox init
devbox validate
devbox start
devbox status
devbox logs web
devbox exec web sh
devbox stop
```

## Configuration (`devbox.yml`)

Minimal example:

```yaml
name: my-app
version: "1.0"
services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
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

## CLI

Top-level commands:
`build`, `completion`, `config`, `destroy`, `doctor`, `exec`, `init`, `logs`, `prune`, `ps`, `reset`, `secrets`, `snapshot`, `start`, `status`, `stop`, `upgrade`, `validate`, `version`.

Run `devbox <command> --help` for details.

## Docs

See `docs/`:
1. `docs/README.md` (index)
2. `docs/reference/config.md` (full config reference)
3. `docs/reference/cli.md` (command reference)
4. `docs/guides/` (installation, snapshots, secrets, plugins, troubleshooting)

## Architecture (Local)

Components:
1. CLI (`devbox`) talks to Engine (`devbox-engine`) over local IPC
2. Engine orchestrates Docker containers, networks, volumes
3. Shared packages provide config, platform, logging, snapshot, secrets

IPC:
1. Windows: TCP `127.0.0.1:51000`
2. macOS/Linux: Unix socket `~/.devbox/engine.sock`

## Contributing

```bash
go test ./...
```

## License

MIT (see `LICENSE`).
