# DevBoxOS — Complete End-to-End Implementation Plan

> **Status:** Complete Implementation Blueprint
> **Scope:** All phases (Local MVP → Cloud → Enterprise)
> **Current Build Target:** Phase 1-2 (Local Only)
> **Future Phases:** Documented for planning, not yet in scope

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Phase 1-2: Local MVP — Detailed Plan](#2-phase-1-2-local-mvp--detailed-plan)
3. [Phase 3: Cloud Backend — Detailed Plan](#3-phase-3-cloud-backend--detailed-plan)
4. [Phase 4: Cloud Compute — Detailed Plan](#4-phase-4-cloud-compute--detailed-plan)
5. [Phase 5: Enterprise — Detailed Plan](#5-phase-5-enterprise--detailed-plan)
6. [Data Models](#6-data-models)
7. [API Contracts](#7-api-contracts)
8. [Configuration Schema](#8-configuration-schema)
9. [Testing Strategy](#9-testing-strategy)
10. [CI/CD Pipeline](#10-cicd-pipeline)
11. [Release & Versioning](#11-release--versioning)
12. [Dependency Graph](#12-dependency-graph)
13. [Migration Strategy](#13-migration-strategy)
14. [Security Threat Model](#14-security-threat-model)
15. [Architecture Decision Records](#15-architecture-decision-records)
16. [Cost Estimation](#16-cost-estimation)
17. [Error Taxonomy](#17-error-taxonomy)
18. [Performance Targets](#18-performance-targets)
19. [Cloud Observability](#19-cloud-observability)
20. [Disaster Recovery](#20-disaster-recovery)
21. [Rate Limiting & Abuse Prevention](#21-rate-limiting--abuse-prevention)

---

## 1. Project Structure

### 1.1 Monorepo Layout

```
devboxos/
├── cli/                          # DevBoxOS CLI (Go)
│   ├── cmd/                      # CLI commands
│   │   ├── root.go               # Root command
│   │   ├── start.go              # devbox start
│   │   ├── stop.go               # devbox stop
│   │   ├── logs.go               # devbox logs
│   │   ├── status.go             # devbox status
│   │   ├── reset.go              # devbox reset
│   │   ├── doctor.go             # devbox doctor
│   │   ├── snapshot/             # Snapshot commands
│   │   │   ├── save.go
│   │   │   ├── load.go
│   │   │   ├── list.go
│   │   │   └── delete.go
│   │   ├── share.go              # devbox share (Phase 3)
│   │   ├── join.go               # devbox join (Phase 3)
│   │   ├── config.go             # devbox config
│   │   ├── init.go               # devbox init
│   │   ├── plugin/               # Plugin commands
│   │   │   ├── install.go
│   │   │   ├── list.go
│   │   │   └── remove.go
│   │   └── version.go            # devbox version
│   ├── internal/
│   │   ├── client/               # Engine daemon client
│   │   │   ├── grpc_client.go    # gRPC client to engine
│   │   │   └── fallback.go       # Direct invocation fallback
│   │   ├── output/               # Terminal output formatting
│   │   │   ├── table.go
│   │   │   ├── json.go
│   │   │   └── spinner.go
│   │   └── telemetry/            # Anonymous usage telemetry
│   │       └── telemetry.go
│   └── main.go
│
├── engine/                       # Environment Engine daemon (Go)
│   ├── cmd/
│   │   └── daemon.go             # Daemon entrypoint
│   ├── internal/
│   │   ├── config/               # YAML config parser
│   │   │   ├── parser.go
│   │   │   ├── validator.go
│   │   │   ├── schema.go         # JSON Schema for devbox.yml
│   │   │   └── autodetect.go     # Auto-detect runtimes/services
│   │   ├── orchestrator/         # Service orchestration
│   │   │   ├── orchestrator.go   # Main orchestrator
│   │   │   ├── graph.go          # Dependency graph resolution
│   │   │   ├── lifecycle.go      # Start/stop/restart logic
│   │   │   ├── healthcheck.go    # Health check engine
│   │   │   └── recovery.go       # Error recovery & resilience
│   │   ├── runtime/              # Container runtime abstraction
│   │   │   ├── docker/           # Docker implementation
│   │   │   │   ├── client.go
│   │   │   │   ├── container.go
│   │   │   │   ├── network.go
│   │   │   │   └── volume.go
│   │   │   ├── containerd/       # containerd implementation (future)
│   │   │   └── runtime.go        # Runtime interface
│   │   ├── networking/           # Networking layer
│   │   │   ├── dns.go            # Local DNS resolver
│   │   │   ├── network.go        # Virtual network management
│   │   │   ├── mtls.go           # mTLS certificate management
│   │   │   └── egress.go         # Egress policy enforcement
│   │   ├── secrets/              # Secrets management
│   │   │   ├── age.go            # age encryption/decryption
│   │   │   ├── vault.go          # HashiCorp Vault integration
│   │   │   ├── onepassword.go    # 1Password integration
│   │   │   ├── aws.go            # AWS Secrets Manager
│   │   │   └── injector.go       # Secret injection into containers
│   │   ├── snapshot/             # Snapshot engine
│   │   │   ├── format.go         # .devbox archive format
│   │   │   ├── capture.go        # State capture
│   │   │   ├── restore.go        # State restore
│   │   │   ├── integrity.go      # SHA-256 + Ed25519 verification
│   │   │   └── sbom.go           # SBOM generation (SPDX)
│   │   ├── logging/              # Log management
│   │   │   ├── collector.go      # Log collection from containers
│   │   │   ├── storage.go        # Local log storage
│   │   │   ├── rotation.go       # Log rotation
│   │   │   └── stream.go         # Log streaming
│   │   ├── state/                # State management
│   │   │   ├── sqlite.go         # SQLite state database
│   │   │   ├── lock.go           # File-based locking
│   │   │   └── reconciliation.go # State reconciliation
│   │   ├── diagnostics/          # Intelligent diagnostics
│   │   │   ├── doctor.go         # Diagnostic engine
│   │   │   ├── suggestions.go    # Fix suggestion engine
│   │   │   └── reporter.go       # Error reporting
│   │   ├── plugin/               # Plugin system
│   │   │   ├── registry.go       # Plugin registry
│   │   │   ├── loader.go         # Plugin loader
│   │   │   ├── sandbox.go        # Plugin sandboxing
│   │   │   └── api.go            # Plugin API definitions
│   │   └── security/             # Security layer
│   │       ├── capabilities.go   # Linux capability management
│   │       ├── seccomp.go        # Seccomp profiles
│   │       └── image_verify.go   # Cosign/Sigstore verification
│   ├── proto/                    # gRPC protocol definitions
│   │   └── engine.proto
│   └── main.go
│
├── cloud/                        # Cloud Backend (Go) — Phase 3+
│   ├── api/                      # REST API server
│   │   ├── cmd/
│   │   │   └── server.go
│   │   ├── internal/
│   │   │   ├── auth/             # Authentication
│   │   │   │   ├── oauth.go      # OAuth 2.0 device flow
│   │   │   │   ├── apikey.go     # API key auth
│   │   │   │   └── middleware.go # Auth middleware
│   │   │   ├── snapshots/        # Snapshot storage API
│   │   │   │   ├── handler.go
│   │   │   │   ├── s3.go         # S3 storage backend
│   │   │   │   └── encryption.go # Snapshot encryption
│   │   │   ├── sharing/          # Environment sharing
│   │   │   │   ├── handler.go
│   │   │   │   ├── tokens.go     # Share token generation
│   │   │   │   └── webrtc.go     # P2P sharing (optional)
│   │   │   ├── teams/            # Team workspace
│   │   │   │   ├── handler.go
│   │   │   │   ├── members.go
│   │   │   │   └── rbac.go       # Role-based access (Phase 5)
│   │   │   ├── compute/          # Remote compute (Phase 4)
│   │   │   │   ├── handler.go
│   │   │   │   ├── provision.go  # Environment provisioning
│   │   │   │   └── k8s.go        # Kubernetes integration
│   │   │   ├── billing/          # Billing & metering (Phase 4)
│   │   │   │   ├── handler.go
│   │   │   │   ├── usage.go      # Usage tracking
│   │   │   │   └── stripe.go     # Stripe integration
│   │   │   └── enterprise/       # Enterprise features (Phase 5)
│   │   │       ├── sso.go        # SAML/OIDC SSO
│   │   │       ├── audit.go      # Audit logging
│   │   │       └── compliance.go # Compliance exports
│   │   ├── openapi/              # OpenAPI 3.0 specification
│   │   │   └── spec.yaml
│   │   └── main.go
│   ├── web/                      # Web Dashboard (React/Next.js)
│   │   ├── src/
│   │   │   ├── app/              # Next.js App Router
│   │   │   ├── components/       # React components
│   │   │   ├── lib/              # API client, utilities
│   │   │   └── styles/           # Tailwind CSS
│   │   └── package.json
│   └── deploy/                   # Deployment configurations
│       ├── docker-compose.yml    # Self-hosted deployment
│       └── helm/                 # Kubernetes Helm chart
│           ├── Chart.yaml
│           ├── values.yaml
│           └── templates/
│
├── shared/                       # Shared types and utilities
│   ├── proto/                    # Shared protobuf definitions
│   ├── types/                    # Shared Go types
│   │   ├── config.go             # devbox.yml types
│   │   ├── snapshot.go           # Snapshot types
│   │   └── api.go                # API request/response types
│   └── schemas/                  # JSON Schema definitions
│       └── devbox.schema.json
│
├── plugins/                      # Official plugins
│   ├── runtime-deno/             # Deno runtime plugin
│   ├── runtime-bun/              # Bun runtime plugin
│   └── secret-doppler/           # Doppler secret provider
│
├── docs/                         # Documentation
│   ├── getting-started/
│   ├── reference/
│   ├── guides/
│   └── contributing/
│
├── tests/                        # Test suites
│   ├── unit/                     # Unit tests
│   ├── integration/              # Integration tests
│   ├── e2e/                      # End-to-end tests
│   ├── smoke/                    # Smoke tests
│   └── fixtures/                 # Test fixtures
│       ├── sample-projects/      # Sample devbox.yml projects
│       └── snapshots/            # Pre-built snapshot archives
│
├── scripts/                      # Build and utility scripts
│   ├── build.sh
│   ├── test.sh
│   ├── release.sh
│   └── dev.sh
│
├── .github/                      # GitHub configuration
│   ├── workflows/                # CI/CD workflows
│   ├── ISSUE_TEMPLATE/
│   └── SECURITY.md
│
├── go.work                        # Go workspace (multi-module)
├── go.mod                         # Root module (if needed)
├── Makefile                       # Build targets
├── LICENSE                        # Apache 2.0
├── README.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
└── SECURITY.md
```

---

## 2. Phase 1-2: Local MVP — Detailed Plan

**Target:** Months 1-5
**Scope:** 100% local, no cloud dependency
**Deliverable:** `devbox start` works on macOS, Linux, Windows

### 2.1 Sprint Breakdown

#### Sprint 1-2: Foundation (Weeks 1-4)

**Goal:** CLI skeleton + Engine daemon + basic config parsing

| Task | Files | Details |
|------|-------|---------|
| Initialize Go workspace | `go.work`, `cli/go.mod`, `engine/go.mod` | Multi-module Go workspace |
| CLI framework | `cli/cmd/root.go`, `cli/main.go` | Cobra CLI, version command, help |
| Daemon architecture | `engine/cmd/daemon.go`, `engine/internal/state/` | Background daemon with auto-restart, SQLite state DB |
| gRPC protocol | `engine/proto/engine.proto`, `shared/proto/` | Define CLI ↔ Engine communication protocol |
| CLI client | `cli/internal/client/grpc_client.go`, `cli/internal/client/fallback.go` | gRPC client with direct invocation fallback |
| YAML config parser | `engine/internal/config/parser.go`, `engine/internal/config/schema.go` | Parse `devbox.yml`, validate against JSON Schema |
| Output formatting | `cli/internal/output/table.go`, `cli/internal/output/spinner.go` | Table output, spinners, colored output |

**Milestone:** `devbox version` works, daemon starts/stops, config file parses

---

#### Sprint 3-4: Service Orchestration (Weeks 5-8)

**Goal:** Start/stop services from `devbox.yml`

| Task | Files | Details |
|------|-------|---------|
| Docker runtime | `engine/internal/runtime/docker/client.go`, `engine/internal/runtime/docker/container.go` | Docker SDK integration, container lifecycle |
| Dependency graph | `engine/internal/orchestrator/graph.go` | Topological sort of service dependencies |
| Service lifecycle | `engine/internal/orchestrator/lifecycle.go` | Start in dependency order, stop in reverse |
| Volume management | `engine/internal/runtime/docker/volume.go` | Bind mounts, named volumes, data persistence |
| `devbox start` | `cli/cmd/start.go` | Command implementation, daemon communication |
| `devbox stop` | `cli/cmd/stop.go` | Graceful shutdown, SIGTERM → wait → SIGKILL |
| `devbox status` | `cli/cmd/status.go`, `engine/internal/state/sqlite.go` | Show running services, health status |

**Milestone:** `devbox start` launches multi-service project from `devbox.yml`

---

#### Sprint 5-6: Networking & DNS (Weeks 9-12)

**Goal:** Local service discovery with `.local` hostnames

| Task | Files | Details |
|------|-------|---------|
| Virtual networks | `engine/internal/networking/network.go` | Per-project Docker networks |
| DNS resolver | `engine/internal/networking/dns.go` | Local DNS for `service.local` resolution |
| Port management | `engine/internal/runtime/docker/network.go` | Port allocation, conflict detection, host exposure |
| mTLS setup | `engine/internal/networking/mtls.go` | Per-environment CA, certificate generation |
| Egress policies | `engine/internal/networking/egress.go` | Default-deny egress, explicit allow rules |
| `devbox init` | `cli/cmd/init.go`, `engine/internal/config/autodetect.go` | Auto-generate `devbox.yml` from project files |

**Milestone:** Services communicate via `api.local:3000`, `db.local:5432`

---

#### Sprint 7-8: Secrets & Security (Weeks 13-16)

**Goal:** Encrypted secrets, container security defaults

| Task | Files | Details |
|------|-------|---------|
| age encryption | `engine/internal/secrets/age.go` | Encrypt/decrypt `.env.devbox.age` files |
| Secret injection | `engine/internal/secrets/injector.go` | Inject secrets into containers at startup |
| Vault integration | `engine/internal/secrets/vault.go` | HashiCorp Vault provider |
| Capability management | `engine/internal/security/capabilities.go` | Drop all capabilities by default, allowlist |
| Seccomp profiles | `engine/internal/security/seccomp.go` | Default seccomp profile, custom profiles |
| Image verification | `engine/internal/security/image_verify.go` | Cosign/Sigstore signature verification |
| `devbox secrets` | `cli/cmd/secrets/` (new) | Secret management CLI commands |

**Milestone:** Secrets encrypted at rest, containers run with minimal privileges

---

#### Sprint 9-10: Snapshots (Weeks 17-20)

**Goal:** Local snapshot save/load

| Task | Files | Details |
|------|-------|---------|
| Archive format | `engine/internal/snapshot/format.go` | `.devbox` bundle: tar.gz with manifest.json |
| State capture | `engine/internal/snapshot/capture.go` | DB dumps, volume snapshots, config, metadata |
| State restore | `engine/internal/snapshot/restore.go` | Restore from snapshot archive |
| Integrity verification | `engine/internal/snapshot/integrity.go` | SHA-256 hashes, Ed25519 signatures |
| SBOM generation | `engine/internal/snapshot/sbom.go` | SPDX-format Software Bill of Materials |
| `devbox snapshot save` | `cli/cmd/snapshot/save.go` | Save command |
| `devbox snapshot load` | `cli/cmd/snapshot/load.go` | Load command |
| `devbox snapshot list` | `cli/cmd/snapshot/list.go` | List local snapshots |
| `devbox snapshot delete` | `cli/cmd/snapshot/delete.go` | Delete local snapshots |

**Milestone:** Full environment state can be saved and restored

---

#### Sprint 11-12: Health, Logging, Diagnostics (Weeks 21-24)

**Goal:** Production-ready observability

| Task | Files | Details |
|------|-------|---------|
| Health check engine | `engine/internal/orchestrator/healthcheck.go` | HTTP, TCP, custom health checks |
| Recovery engine | `engine/internal/orchestrator/recovery.go` | Restart policies, circuit breakers, backoff |
| Log collector | `engine/internal/logging/collector.go` | Collect logs from containers |
| Log storage | `engine/internal/logging/storage.go` | Local log storage with 100MB cap |
| Log rotation | `engine/internal/logging/rotation.go` | Automatic rotation, retention |
| Log streaming | `engine/internal/logging/stream.go` | Real-time log streaming |
| `devbox logs` | `cli/cmd/logs.go` | Log command with --follow, --tail, --since |
| Diagnostic engine | `engine/internal/diagnostics/doctor.go` | Analyze environment issues |
| Fix suggestions | `engine/internal/diagnostics/suggestions.go` | Human-readable fix suggestions |
| `devbox doctor` | `cli/cmd/doctor.go` | Diagnose and repair command |

**Milestone:** Full observability — logs, health checks, diagnostics

---

#### Sprint 13-14: Cross-Platform & Polish (Weeks 25-28)

**Goal:** Works on macOS, Linux, Windows

| Task | Files | Details |
|------|-------|---------|
| Windows support | All platform-specific code | Named pipes, path handling, Docker Desktop |
| macOS support | All platform-specific code | Unix sockets, launchd integration |
| Linux support | All platform-specific code | systemd integration, Unix sockets |
| Telemetry | `cli/internal/telemetry/telemetry.go` | Anonymous usage data, opt-out |
| `devbox config` | `cli/cmd/config.go` | Configuration management |
| `devbox reset` | `cli/cmd/reset.go` | Tear down and rebuild |
| Plugin system | `engine/internal/plugin/` | Plugin loader, registry, sandboxing |
| Documentation | `docs/` | Getting started, reference, guides |

**Milestone:** Production-ready, cross-platform, documented

---

### 2.2 Phase 1-2 Deliverables Checklist

- [ ] `devbox start` — starts all services from `devbox.yml`
- [ ] `devbox stop` — graceful shutdown
- [ ] `devbox status` — shows running services
- [ ] `devbox logs [service]` — stream logs
- [ ] `devbox reset` — tear down and rebuild
- [ ] `devbox snapshot save/load/list/delete` — local snapshots
- [ ] `devbox doctor` — diagnose issues
- [ ] `devbox init` — auto-generate config
- [ ] `devbox config` — manage settings
- [ ] `devbox version` — show version
- [ ] Cross-platform: macOS, Linux, Windows
- [ ] mTLS between services
- [ ] Encrypted secrets (age)
- [ ] Container security defaults
- [ ] Health checks and recovery
- [ ] Log management
- [ ] Plugin system
- [ ] Telemetry (opt-out)
- [ ] Full documentation

---

## 3. Phase 3: Cloud Backend — Detailed Plan

**Target:** Months 6-9
**Scope:** Cloud storage, sharing, team workspace — fully open source, self-hostable
**Deliverable:** `devbox share`, cloud snapshot storage, team workspace

### 3.1 Architecture

```
┌──────────────┐         ┌──────────────────────┐         ┌──────────────┐
│  DevBox CLI  │◄───────►│   Cloud API (Go)     │◄───────►│  PostgreSQL  │
│  (local)     │  gRPC/  │                      │  SQL    │              │
│              │  REST   │                      │         └──────────────┘
└──────────────┘         │                      │         ┌──────────────┐
                         │                      │◄───────►│  Redis       │
                         │                      │         │  (cache)     │
                         │                      │         └──────────────┘
                         │                      │         ┌──────────────┐
                         │                      │◄───────►│  S3          │
                         │                      │         │  (snapshots) │
                         └──────────────────────┘         └──────────────┘
                                          │
                                          ▼
                              ┌──────────────────────┐
                              │  Web Dashboard       │
                              │  (Next.js)           │
                              └──────────────────────┘
```

### 3.2 Sprint Breakdown

#### Sprint 15-16: Cloud API Foundation (Weeks 29-32)

| Task | Files | Details |
|------|-------|---------|
| API server skeleton | `cloud/api/cmd/server.go` | Go HTTP server, routing |
| OpenAPI spec | `cloud/api/openapi/spec.yaml` | Full API contract |
| OAuth 2.0 device flow | `cloud/api/internal/auth/oauth.go` | CLI authentication |
| API key auth | `cloud/api/internal/auth/apikey.go` | Machine-to-machine auth |
| Auth middleware | `cloud/api/internal/auth/middleware.go` | JWT validation, rate limiting |
| PostgreSQL setup | `cloud/api/internal/db/` | Migrations, connection pooling |
| Redis cache | `cloud/api/internal/cache/` | Session management |

**Milestone:** Cloud API running, CLI can authenticate

---

#### Sprint 17-18: Snapshot Storage (Weeks 33-36)

| Task | Files | Details |
|------|-------|---------|
| S3 storage backend | `cloud/api/internal/snapshots/s3.go` | Upload/download to S3 |
| Snapshot encryption | `cloud/api/internal/snapshots/encryption.go` | AES-256-GCM at rest |
| Snapshot API | `cloud/api/internal/snapshots/handler.go` | CRUD endpoints |
| `devbox snapshot push` | `cli/cmd/snapshot/push.go` (new) | Upload to cloud |
| `devbox snapshot pull` | `cli/cmd/snapshot/pull.go` (new) | Download from cloud |

**Milestone:** Snapshots can be stored and retrieved from cloud

---

#### Sprint 19-20: Environment Sharing (Weeks 37-40)

| Task | Files | Details |
|------|-------|---------|
| Share token system | `cloud/api/internal/sharing/tokens.go` | Token generation, expiry, access control |
| Share API | `cloud/api/internal/sharing/handler.go` | Create/manage shares |
| `devbox share` | `cli/cmd/share.go` | Create shareable link |
| `devbox join` | `cli/cmd/join.go` | Join shared environment |
| P2P sharing (optional) | `cloud/api/internal/sharing/webrtc.go` | Direct peer-to-peer fallback |

**Milestone:** Environments can be shared via token

---

#### Sprint 21-22: Team Workspace (Weeks 41-44)

| Task | Files | Details |
|------|-------|---------|
| Team management | `cloud/api/internal/teams/handler.go` | Create/join teams |
| Member management | `cloud/api/internal/teams/members.go` | Invite/remove members |
| Shared snapshot library | `cloud/api/internal/snapshots/shared.go` | Team-accessible snapshots |
| Web dashboard | `cloud/web/src/` | Next.js dashboard |
| Self-hosted deploy | `cloud/deploy/docker-compose.yml`, `cloud/deploy/helm/` | Docker Compose + Helm chart |

**Milestone:** Teams can collaborate with shared snapshots

---

### 3.3 Phase 3 Deliverables Checklist

- [ ] Cloud API (Go) running with OpenAPI spec
- [ ] OAuth 2.0 device flow + API key auth
- [ ] Snapshot cloud storage (S3)
- [ ] `devbox snapshot push/pull`
- [ ] `devbox share/join`
- [ ] Team workspace
- [ ] Web dashboard (Next.js)
- [ ] Self-hosted deployment (Docker Compose + Helm)
- [ ] Fully open source (Apache 2.0)

---

## 4. Phase 4: Cloud Compute — Detailed Plan

**Target:** Months 9-12
**Scope:** Remote environments, CI integration, billing
**Deliverable:** `devbox up --cloud`, `devbox ci run`, billing system

### 4.1 Sprint Breakdown

#### Sprint 23-24: Remote Compute (Weeks 45-48)

| Task | Files | Details |
|------|-------|---------|
| Provisioning API | `cloud/api/internal/compute/handler.go` | Create/destroy remote environments |
| Kubernetes integration | `cloud/api/internal/compute/k8s.go` | Provision on K8s cluster |
| `devbox up --cloud` | `cli/cmd/up.go` (new) | Start cloud environment |
| Hybrid mode | `engine/internal/runtime/` | Local frontend + cloud backend |

**Milestone:** Remote dev environments provisioned on demand

---

#### Sprint 25-26: CI Integration (Weeks 49-52)

| Task | Files | Details |
|------|-------|---------|
| CI environment contract | `shared/types/ci.go` | Lightweight snapshot format for CI |
| `devbox ci run` | `cli/cmd/ci/run.go` (new) | Run commands in CI environment |
| GitHub Action | `.github/actions/devbox/` | Official GitHub Action |
| GitLab CI template | `cloud/deploy/gitlab/` | Official GitLab CI template |

**Milestone:** Reproducible CI pipelines with DevBoxOS

---

#### Sprint 27-28: Billing & Metering (Weeks 53-56)

| Task | Files | Details |
|------|-------|---------|
| Usage tracking | `cloud/api/internal/billing/usage.go` | Track compute hours, storage |
| Stripe integration | `cloud/api/internal/billing/stripe.go` | Payment processing |
| Billing API | `cloud/api/internal/billing/handler.go` | Usage, invoices, subscriptions |
| Web billing UI | `cloud/web/src/app/billing/` | Billing dashboard |

**Milestone:** Paid tiers live with usage-based billing

---

### 4.2 Phase 4 Deliverables Checklist

- [ ] `devbox up --cloud` — remote environments
- [ ] Kubernetes-based provisioning
- [ ] `devbox ci run` — CI integration
- [ ] GitHub Action + GitLab CI template
- [ ] Billing and metering (Stripe)
- [ ] Paid tiers (Pro, Team) live

---

## 5. Phase 5: Enterprise — Detailed Plan

**Target:** Months 12-18
**Scope:** RBAC, SSO, audit logging, Firecracker runtime, compliance
**Deliverable:** Enterprise-grade platform with self-hosted option

### 5.1 Sprint Breakdown

#### Sprint 29-30: RBAC & SSO (Weeks 57-60)

| Task | Files | Details |
|------|-------|---------|
| RBAC system | `cloud/api/internal/teams/rbac.go` | Role-based permissions |
| SAML SSO | `cloud/api/internal/enterprise/sso.go` | SAML integration |
| OIDC SSO | `cloud/api/internal/enterprise/sso.go` | OIDC integration |
| Audit logging | `cloud/api/internal/enterprise/audit.go` | Operation audit trail |
| Compliance exports | `cloud/api/internal/enterprise/compliance.go` | SOC 2, GDPR exports |

**Milestone:** Enterprise auth, permissions, and audit

---

#### Sprint 31-32: Firecracker Runtime (Weeks 61-66)

| Task | Files | Details |
|------|-------|---------|
| Firecracker integration | `engine/internal/runtime/firecracker/` | Micro-VM runtime |
| VM image management | `engine/internal/runtime/firecracker/images.go` | VM image lifecycle |
| Networking | `engine/internal/runtime/firecracker/network.go` | TAP device networking |
| Enterprise runtime flag | `engine/internal/config/` | Switch between container/micro-VM |

**Milestone:** Hardware-level isolation for enterprise

---

#### Sprint 33-34: Compliance & Scale (Weeks 67-72)

| Task | Files | Details |
|------|-------|---------|
| SOC 2 Type II | Security team | Certification process |
| GDPR compliance | Legal/Security | Data processing agreements |
| Global edge distribution | Infrastructure | Multi-region deployment |
| Enterprise SLA | Product team | Support tooling, monitoring |
| Self-hosted enterprise | `cloud/deploy/helm/` | Enterprise Helm chart |

**Milestone:** Enterprise GA with compliance certifications

---

### 5.2 Phase 5 Deliverables Checklist

- [ ] RBAC and team permissions
- [ ] SSO (SAML + OIDC)
- [ ] Audit logging
- [ ] Firecracker micro-VM runtime
- [ ] SOC 2 Type II certification
- [ ] GDPR compliance
- [ ] Global edge distribution
- [ ] Enterprise self-hosted deployment
- [ ] Enterprise SLA

---

## 6. Data Models

### 6.1 Local State (SQLite)

```sql
-- environments table
CREATE TABLE environments (
    id          TEXT PRIMARY KEY,    -- UUID
    name        TEXT NOT NULL,       -- Project name from devbox.yml
    path        TEXT NOT NULL,       -- Absolute path to project
    version     TEXT NOT NULL,       -- devbox.yml version
    status      TEXT NOT NULL,       -- running, stopped, failed
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

-- services table
CREATE TABLE services (
    id              TEXT PRIMARY KEY,
    environment_id  TEXT NOT NULL REFERENCES environments(id),
    name            TEXT NOT NULL,
    container_id    TEXT,            -- Docker container ID
    status          TEXT NOT NULL,   -- starting, running, stopped, failed, healthy
    port            INTEGER,
    health_status   TEXT,            -- healthy, unhealthy, starting
    last_check      DATETIME,
    restart_count   INTEGER DEFAULT 0,
    created_at      DATETIME NOT NULL
);

-- snapshots table
CREATE TABLE snapshots (
    id              TEXT PRIMARY KEY,
    environment_id  TEXT NOT NULL REFERENCES environments(id),
    name            TEXT NOT NULL,
    path            TEXT NOT NULL,   -- Local file path
    size_bytes      INTEGER NOT NULL,
    hash_sha256     TEXT NOT NULL,   -- Content hash
    signature       TEXT,            -- Ed25519 signature (optional)
    metadata        JSON,            -- Service states, versions, etc.
    created_at      DATETIME NOT NULL
);

-- locks table
CREATE TABLE locks (
    id          TEXT PRIMARY KEY,
    environment_id TEXT NOT NULL REFERENCES environments(id),
    operation   TEXT NOT NULL,       -- start, stop, snapshot, etc.
    acquired_at DATETIME NOT NULL,
    expires_at  DATETIME NOT NULL
);

-- telemetry table
CREATE TABLE telemetry (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type  TEXT NOT NULL,       -- command_start, command_end, error
    command     TEXT,                -- start, stop, etc.
    duration_ms INTEGER,
    os          TEXT,
    arch        TEXT,
    version     TEXT,
    timestamp   DATETIME NOT NULL
);
```

### 6.2 Cloud Database (PostgreSQL)

```sql
-- users table
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    name            TEXT,
    auth_provider   TEXT NOT NULL,   -- oauth, apikey
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- api_keys table
CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    key_hash        TEXT NOT NULL,   -- bcrypt hash
    name            TEXT,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- teams table
CREATE TABLE teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT UNIQUE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- team_members table
CREATE TABLE team_members (
    team_id         UUID NOT NULL REFERENCES teams(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    role            TEXT NOT NULL,   -- owner, admin, member
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

-- snapshots table
CREATE TABLE snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID REFERENCES teams(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    s3_key          TEXT NOT NULL,   -- S3 object key
    s3_bucket       TEXT NOT NULL,
    size_bytes      BIGINT NOT NULL,
    hash_sha256     TEXT NOT NULL,
    encryption_key  TEXT NOT NULL,   -- Encrypted AES key
    metadata        JSONB,
    is_shared       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- share_tokens table
CREATE TABLE share_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id     UUID NOT NULL REFERENCES snapshots(id),
    token           TEXT UNIQUE NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    max_uses        INTEGER,
    use_count       INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- audit_logs table
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    team_id         UUID REFERENCES teams(id),
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT,
    details         JSONB,
    ip_address      TEXT,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- usage_records table
CREATE TABLE usage_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    team_id         UUID REFERENCES teams(id),
    resource_type   TEXT NOT NULL,   -- compute_hours, storage_gb, shares
    quantity        DECIMAL NOT NULL,
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.3 Snapshot Archive Format (`.devbox`)

```
snapshot-name.devbox/
├── manifest.json           # Snapshot metadata
├── config.json             # devbox.yml at time of snapshot
├── volumes/
│   ├── db/                 # Per-service volume dumps
│   │   └── dump.sql        # pg_dump output
│   └── redis/
│       └── dump.rdb        # Redis RDB dump
├── dependencies/
│   ├── node/
│   │   └── package-lock.json
│   └── python/
│       └── requirements.lock
├── logs/
│   ├── api.log
│   └── db.log
├── sbom.spdx.json          # Software Bill of Materials
├── integrity.json          # SHA-256 hashes of all files
└── signature.sig           # Ed25519 signature (optional)
```

**manifest.json schema:**
```json
{
  "version": "1.0",
  "id": "uuid",
  "name": "pre-migration-v2",
  "environment": {
    "name": "my-app",
    "version": "1.0",
    "path": "/path/to/project"
  },
  "services": [
    {
      "name": "api",
      "image": "node:18",
      "status": "running",
      "port": 3000
    }
  ],
  "created_at": "2026-05-15T12:00:00Z",
  "os": "darwin",
  "arch": "arm64",
  "devbox_version": "0.1.0"
}
```

---

## 7. API Contracts

### 7.1 CLI ↔ Engine (gRPC)

```protobuf
syntax = "proto3";
package engine;

service EngineService {
  rpc Start(StartRequest) returns (StreamResponse);
  rpc Stop(StopRequest) returns (StatusResponse);
  rpc Status(StatusRequest) returns (StatusResponse);
  rpc Logs(LogsRequest) returns (stream LogEntry);
  rpc SnapshotSave(SnapshotSaveRequest) returns (StreamResponse);
  rpc SnapshotLoad(SnapshotLoadRequest) returns (StreamResponse);
  rpc SnapshotList(SnapshotListRequest) returns (SnapshotListResponse);
  rpc SnapshotDelete(SnapshotDeleteRequest) returns (StatusResponse);
  rpc Doctor(DoctorRequest) returns (DoctorResponse);
  rpc Reset(ResetRequest) returns (StreamResponse);
}

message StartRequest {
  string project_path = 1;
  bool force = 2;
}

message StopRequest {
  string project_path = 1;
  string service = 2;  // empty = all services
  int32 grace_period_seconds = 3;
}

message StatusRequest {
  string project_path = 1;
}

message LogsRequest {
  string project_path = 1;
  string service = 2;
  int32 tail = 3;
  string since = 4;
  bool follow = 5;
}

message LogEntry {
  string service = 1;
  string timestamp = 2;
  string level = 3;
  string message = 4;
  bytes raw = 5;
}

message StatusResponse {
  string status = 1;
  repeated ServiceStatus services = 2;
  string error = 3;
}

message ServiceStatus {
  string name = 1;
  string status = 2;
  string health = 3;
  int32 port = 4;
  string container_id = 5;
  int32 restart_count = 6;
}

message StreamResponse {
  string status = 1;
  string message = 2;
  bool done = 3;
  string error = 4;
}

message SnapshotSaveRequest {
  string project_path = 1;
  string name = 2;
  bool include_logs = 3;
}

message SnapshotLoadRequest {
  string project_path = 1;
  string snapshot_id = 2;
  string snapshot_path = 3;  // for file-based load
  bool force = 4;
}

message SnapshotListRequest {
  string project_path = 1;
}

message SnapshotListResponse {
  repeated Snapshot snapshots = 1;
}

message Snapshot {
  string id = 1;
  string name = 2;
  int64 size_bytes = 3;
  string hash_sha256 = 4;
  string created_at = 5;
  string metadata = 6;  // JSON
}

message SnapshotDeleteRequest {
  string project_path = 1;
  string snapshot_id = 2;
}

message DoctorRequest {
  string project_path = 1;
  string service = 2;
}

message DoctorResponse {
  repeated DiagnosticIssue issues = 1;
  repeated string suggestions = 2;
}

message DiagnosticIssue {
  string severity = 1;  // error, warning, info
  string service = 2;
  string message = 3;
  string details = 4;
}

message ResetRequest {
  string project_path = 1;
  bool force = 2;
}
```

### 7.2 Cloud REST API (OpenAPI 3.0)

```yaml
openapi: "3.0.3"
info:
  title: DevBoxOS Cloud API
  version: "1.0.0"

servers:
  - url: https://cloud.devboxos.com/api/v1
  - url: http://localhost:8080/api/v1  # Self-hosted

paths:
  /auth/device:
    post:
      summary: Start OAuth 2.0 device flow
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                client_id:
                  type: string
      responses:
        200:
          description: Device code generated
          content:
            application/json:
              schema:
                type: object
                properties:
                  device_code:
                    type: string
                  user_code:
                    type: string
                  verification_uri:
                    type: string
                  expires_in:
                    type: integer

  /auth/token:
    post:
      summary: Poll for token
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              properties:
                grant_type:
                  type: string
                  enum: [urn:ietf:params:oauth:grant-type:device_code]
                device_code:
                  type: string
      responses:
        200:
          description: Access token
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token:
                    type: string
                  token_type:
                    type: string
                  expires_in:
                    type: integer

  /snapshots:
    get:
      summary: List user's snapshots
      security:
        - bearerAuth: []
      parameters:
        - name: team_id
          in: query
          schema:
            type: string
      responses:
        200:
          description: List of snapshots
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Snapshot'
    post:
      summary: Upload a snapshot
      security:
        - bearerAuth: []
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
                name:
                  type: string
                team_id:
                  type: string
      responses:
        201:
          description: Snapshot uploaded
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Snapshot'

  /snapshots/{id}:
    get:
      summary: Download a snapshot
      security:
        - bearerAuth: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        200:
          description: Snapshot file
          content:
            application/octet-stream:
              schema:
                type: string
                format: binary
    delete:
      summary: Delete a snapshot
      security:
        - bearerAuth: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        204:
          description: Deleted

  /shares:
    post:
      summary: Create a share token
      security:
        - bearerAuth: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                snapshot_id:
                  type: string
                  format: uuid
                expires_in:
                  type: integer
                  description: Seconds until expiry
                max_uses:
                  type: integer
      responses:
        201:
          description: Share token created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ShareToken'

  /shares/{token}:
    get:
      summary: Join a shared environment
      parameters:
        - name: token
          in: path
          required: true
          schema:
            type: string
      responses:
        200:
          description: Snapshot download URL
          content:
            application/json:
              schema:
                type: object
                properties:
                  download_url:
                    type: string
                    format: uri
                  expires_at:
                    type: string
                    format: date-time

  /teams:
    get:
      summary: List user's teams
      security:
        - bearerAuth: []
      responses:
        200:
          description: List of teams
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Team'
    post:
      summary: Create a team
      security:
        - bearerAuth: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        201:
          description: Team created

  /teams/{id}/members:
    post:
      summary: Invite a member
      security:
        - bearerAuth: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [email]
              properties:
                email:
                  type: string
                role:
                  type: string
                  enum: [admin, member]
                  default: member
      responses:
        201:
          description: Member invited

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    Snapshot:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        size_bytes:
          type: integer
        hash_sha256:
          type: string
        created_at:
          type: string
          format: date-time
        is_shared:
          type: boolean
        team_id:
          type: string
          format: uuid

    ShareToken:
      type: object
      properties:
        token:
          type: string
        snapshot_id:
          type: string
          format: uuid
        expires_at:
          type: string
          format: date-time
        max_uses:
          type: integer

    Team:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        slug:
          type: string
        member_count:
          type: integer
        created_at:
          type: string
          format: date-time
```

---

## 8. Configuration Schema

### 8.1 `devbox.yml` Full Schema

```yaml
# devbox.yml — Full schema reference

name: string                    # Project name (required)
version: string                 # Config version (required, semver)

runtimes:                       # Language runtimes
  node: string                  # Node.js version (e.g., "18", "20")
  python: string                # Python version (e.g., "3.11")
  go: string                    # Go version
  rust: string                  # Rust toolchain
  java: string                  # Java version
  ruby: string                  # Ruby version

services:                       # Service definitions (required: at least 1)
  <service-name>:
    # Service type (mutually exclusive)
    image: string               # Docker image (e.g., "postgres:16")
    runtime: string             # Language runtime (e.g., "node18", "python311")
    build:                      # Build from Dockerfile
      context: string           # Build context path
      dockerfile: string        # Dockerfile path (default: Dockerfile)

    # Execution
    command: string             # Command to run
    args: [string]              # Command arguments
    working_dir: string         # Working directory inside container

    # Networking
    port: number | string       # Port number or "host:container"
    ports:                      # Multiple ports
      - number | string
    protocol: string            # tcp, udp (default: tcp)

    # Dependencies
    depends_on: [string]        # Service dependencies

    # Environment
    env:                        # Environment variables
      KEY: string               # Value or ${ref} to other services
    env_file: string            # Load env from file

    # Volumes
    data: string                # Persistent data directory
    volumes:                    # Additional volume mounts
      - string                  # "host:container" or named volume

    # Health checks
    healthcheck:
      type: string              # http, tcp, cmd (default: tcp if port set)
      path: string              # HTTP path (for type: http)
      command: string           # Command to run (for type: cmd)
      interval: string          # Check interval (e.g., "10s")
      timeout: string           # Check timeout (e.g., "5s")
      retries: number           # Max retries before unhealthy
      start_period: string      # Grace period before first check

    # Resource limits
    resources:
      memory: string            # Memory limit (e.g., "512m", "1g")
      cpu: string               # CPU limit (e.g., "0.5", "1.0")
      disk: string              # Disk limit (e.g., "5g")

    # Restart policy
    restart_policy:
      on_failure: boolean       # Restart on failure
      always: boolean           # Always restart
      max_retries: number       # Max restart attempts
      backoff: string           # linear, exponential

    # Security
    security:
      tls: boolean              # Enable mTLS (default: true)
      capabilities: [string]    # Additional Linux capabilities
      read_only: boolean        # Read-only root filesystem

networking:
  discovery: boolean            # Enable .local DNS (default: true)
  expose: [number]              # Ports exposed to host
  egress: string                # default-deny, allow-all (default: default-deny)

security:
  tls: string                   # mTLS, disabled (default: mTLS)
  capabilities: string          # default, custom (default: default)

secrets:
  source: string                # .env.devbox.age, vault, 1password, aws-secrets
  vault:                        # Vault-specific config
    address: string
    path: string
  onepassword:                  # 1Password-specific config
    vault: string
  aws:                          # AWS Secrets Manager config
    region: string
    prefix: string

plugins:                        # Plugin configuration
  - name: string
    version: string
    config:                     # Plugin-specific config

telemetry:                      # Telemetry configuration
  enabled: boolean              # Default: true
```

---

## 9. Testing Strategy

### 9.1 Test Pyramid

```
                    ┌─────────┐
                    │  E2E    │  ← Full workflow tests (few, slow)
                   ┌┴─────────┴┐
                   │Integration│  ← CLI ↔ Engine ↔ Docker (medium)
                  ┌┴───────────┴┐
                  │    Unit     │  ← Individual functions (many, fast)
                 ┌┴─────────────┴┐
                 │    Smoke      │  ← Does it start? (CI matrix)
                 └───────────────┘
```

### 9.2 Test Matrix

| Layer | Scope | Tools | Files | Frequency |
|-------|-------|-------|-------|-----------|
| Unit | Individual functions | Go `testing`, `testify` | `*_test.go` alongside source | Every commit |
| Integration | CLI ↔ Engine ↔ Docker | Go `testing`, testcontainers-go | `tests/integration/` | Every PR |
| Smoke | `devbox start` on real OS | GitHub Actions, Docker-in-Docker | `tests/smoke/` | Every PR |
| E2E | Full workflows | Custom test harness | `tests/e2e/` | Nightly |
| Compatibility | Multiple Docker/OS versions | Matrix CI | `tests/compatibility/` | Weekly |

### 9.3 Coverage Targets

- **Unit tests:** 80%+ line coverage
- **Integration tests:** 100% critical paths (start, stop, snapshot save/load)
- **Smoke tests:** All supported OS × Docker version combinations
- **E2E tests:** Core user journeys (init → start → snapshot → restore)

### 9.4 Test Fixtures

```
tests/fixtures/
├── sample-projects/
│   ├── node-api/             # Simple Node.js API + Postgres
│   │   ├── devbox.yml
│   │   ├── package.json
│   │   └── index.js
│   ├── python-worker/        # Python worker + Redis
│   │   ├── devbox.yml
│   │   ├── requirements.txt
│   │   └── worker.py
│   └── monorepo/             # Monorepo with multiple services
│       ├── devbox.yml
│       ├── api/
│       └── worker/
└── snapshots/
    ├── basic.devbox          # Pre-built snapshot for regression testing
    └── complex.devbox
```

---

## 10. CI/CD Pipeline

### 10.1 GitHub Actions Workflows

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - run: go vet ./...
      - run: golangci-lint run

  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v4

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-integration

  smoke-tests:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        docker: ["24", "25", "26"]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make build
      - run: make test-smoke

  e2e-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-e2e
```

### 10.2 Release Pipeline

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags: ["v*"]

jobs:
  build:
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -o dist/devbox-${{ matrix.goos }}-${{ matrix.goarch }}
      - uses: actions/upload-artifact@v4
        with:
          name: devbox-${{ matrix.goos }}-${{ matrix.goarch }}
          path: dist/

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
      - uses: softprops/action-gh-release@v1
        with:
          files: dist/*
          generate_release_notes: true
```

---

## 11. Release & Versioning

### 11.1 Semantic Versioning

```
v0.1.0  — Alpha: Core CLI + Engine, basic service start
v0.2.0  — Alpha: Networking, DNS, secrets
v0.3.0  — Alpha: Snapshots, health checks
v0.4.0  — Alpha: Diagnostics, logging
v0.5.0  — Private Beta: Cross-platform, plugin system
v0.6.0  — Private Beta: Cloud backend (self-hostable)
v0.7.0  — Private Beta: Sharing, team workspace
v0.8.0  — Public Beta: Cloud hosted SaaS launch
v0.9.0  — Public Beta: CI integration
v1.0.0  — GA: Stable release, Pro tier live
v1.1.0  — Cloud compute, remote environments
v2.0.0  — Enterprise: RBAC, SSO, Firecracker runtime
```

### 11.2 Breaking Change Policy

- **Major versions (v1 → v2):** Breaking changes allowed, migration guides provided
- **Minor versions (v1.0 → v1.1):** New features, backward compatible
- **Patch versions (v1.0.0 → v1.0.1):** Bug fixes only
- **Config compatibility:** New versions can read configs from previous 2 major versions
- **Deprecation window:** 6 months notice before removing any public API or config field

---

## 12. Dependency Graph

### 12.1 Phase Dependencies

```
Phase 1-2 (Local MVP)
├── Sprint 1-2: Foundation
│   └── (no dependencies)
├── Sprint 3-4: Service Orchestration
│   └── depends on: Sprint 1-2
├── Sprint 5-6: Networking & DNS
│   └── depends on: Sprint 3-4
├── Sprint 7-8: Secrets & Security
│   └── depends on: Sprint 3-4
├── Sprint 9-10: Snapshots
│   └── depends on: Sprint 3-4
├── Sprint 11-12: Health, Logging, Diagnostics
│   └── depends on: Sprint 3-4
└── Sprint 13-14: Cross-Platform & Polish
    └── depends on: All previous sprints

Phase 3 (Cloud Backend)
├── Sprint 15-16: Cloud API Foundation
│   └── depends on: Phase 1-2 complete
├── Sprint 17-18: Snapshot Storage
│   └── depends on: Sprint 15-16 + Phase 1-2 Snapshots
├── Sprint 19-20: Environment Sharing
│   └── depends on: Sprint 17-18
└── Sprint 21-22: Team Workspace
    └── depends on: Sprint 17-18

Phase 4 (Cloud Compute)
├── Sprint 23-24: Remote Compute
│   └── depends on: Phase 3 complete
├── Sprint 25-26: CI Integration
│   └── depends on: Phase 1-2 + Phase 3 Snapshots
└── Sprint 27-28: Billing & Metering
    └── depends on: Sprint 23-24

Phase 5 (Enterprise)
├── Sprint 29-30: RBAC & SSO
│   └── depends on: Phase 3 Team Workspace
├── Sprint 31-32: Firecracker Runtime
│   └── depends on: Phase 1-2 Runtime abstraction
└── Sprint 33-34: Compliance & Scale
    └── depends on: All previous phases
```

### 12.2 Go Module Dependencies

```
cli/go.mod
├── github.com/spf13/cobra          # CLI framework
├── google.golang.org/grpc          # gRPC client
├── google.golang.org/protobuf      # Protocol buffers
├── github.com/charmbracelet/lipgloss  # Terminal styling
└── shared/ (replace)               # Shared types

engine/go.mod
├── github.com/docker/docker        # Docker SDK
├── github.com/docker/go-connections # Docker networking
├── github.com/FiloSottile/age      # age encryption
├── github.com/glebarez/go-sqlite   # SQLite driver
├── google.golang.org/grpc          # gRPC server
├── google.golang.org/protobuf      # Protocol buffers
├── github.com/miekg/dns            # DNS server
├── github.com/sigstore/cosign      # Image verification
├── github.com/testcontainers/testcontainers-go  # Testing
└── shared/ (replace)               # Shared types

cloud/api/go.mod
├── github.com/gin-gonic/gin        # HTTP framework
├── github.com/lib/pq               # PostgreSQL driver
├── github.com/redis/go-redis       # Redis client
├── github.com/aws/aws-sdk-go-v2    # AWS SDK (S3)
├── github.com/golang-jwt/jwt       # JWT handling
├── github.com/stripe/stripe-go     # Stripe (Phase 4)
├── github.com/coreos/go-oidc       # OIDC (Phase 5)
└── shared/ (replace)               # Shared types
```

---

## 13. Migration Strategy

### 13.1 Docker Compose Migration

**Goal:** Users can import existing `docker-compose.yml` files without rewriting.

```bash
# Automatic conversion
devbox import docker-compose ./docker-compose.yml

# Generates devbox.yml from docker-compose.yml
# - services → services
# - ports → port
# - volumes → data / volumes
# - environment → env
# - depends_on → depends_on
# - image → image
# - build → build
```

**Conversion mapping:**

| Docker Compose | DevBoxOS | Notes |
|---|---|---|
| `services.<name>.image` | `services.<name>.image` | Direct mapping |
| `services.<name>.build` | `services.<name>.build` | Direct mapping |
| `services.<name>.ports` | `services.<name>.port` | First port mapped, rest in `ports` array |
| `services.<name>.volumes` | `services.<name>.data` / `volumes` | Named volumes → `data`, bind mounts → `volumes` |
| `services.<name>.environment` | `services.<name>.env` | Direct mapping |
| `services.<name>.depends_on` | `services.<name>.depends_on` | Direct mapping |
| `services.<name>.healthcheck` | `services.<name>.healthcheck` | Converted with type inference |
| `services.<name>.deploy.resources` | `services.<name>.resources` | Direct mapping |
| `networks` | `networking.discovery` | Simplified — DevBoxOS handles automatically |
| `volumes` (top-level) | `services.<name>.data` | Flattened into service definitions |

**Unsupported features (warned during import):**
- `configs` / `secrets` (top-level) → converted to `secrets.source`
- `deploy.placement` constraints → ignored (DevBoxOS handles scheduling)
- Custom Docker networks → simplified to DevBoxOS managed network
- `extends` → inlined into service definition

### 13.2 Dev Containers Migration

**Goal:** Users with `.devcontainer/devcontainer.json` can migrate.

```bash
devbox import devcontainer ./.devcontainer
```

**Conversion mapping:**

| Dev Container | DevBoxOS | Notes |
|---|---|---|
| `image` | `services.app.image` | Direct mapping |
| `dockerComposeFile` | Imported as multi-service | Parses compose file |
| `features` | `runtimes` | Mapped to language runtimes |
| `forwardPorts` | `networking.expose` | Direct mapping |
| `containerEnv` | `services.app.env` | Direct mapping |
| `postCreateCommand` | `services.app.command` | Converted to startup command |
| `remoteUser` | `services.app.security` | Mapped to security context |

### 13.3 Migration CLI Commands

```bash
# Import from Docker Compose
devbox import docker-compose <path> [--output devbox.yml]

# Import from Dev Container
devbox import devcontainer <path> [--output devbox.yml]

# Import from Vagrant
devbox import vagrant <path> [--output devbox.yml]

# Validate converted config
devbox validate devbox.yml

# Dry-run migration (show what would change)
devbox import docker-compose <path> --dry-run
```

### 13.4 Migration Documentation

- Dedicated migration guides in `docs/guides/migrate-from-docker-compose.md`
- Interactive migration wizard: `devbox migrate`
- Side-by-side comparison tool showing Docker Compose vs DevBoxOS equivalents
- Migration success metrics tracked via telemetry (anonymous)

---

## 14. Security Threat Model

### 14.1 STRIDE Analysis

| Threat Category | Attack Vector | Mitigation | Phase |
|---|---|---|---|
| **Spoofing** | Attacker impersonates a service in the local network | mTLS with per-environment CA, short-lived certificates | Phase 1-2 |
| **Spoofing** | Attacker spoofs `devbox.local` DNS entries | DNS response validation, per-project isolated networks | Phase 1-2 |
| **Spoofing** | Fake DevBoxOS CLI binary | Code signing (cosign), verified download instructions | Phase 1-2 |
| **Tampering** | Modify `devbox.yml` to inject malicious commands | Config validation, schema enforcement, hash-locking | Phase 1-2 |
| **Tampering** | Modify snapshot archives to inject malware | SHA-256 integrity verification, Ed25519 signatures | Phase 1-2 |
| **Tampering** | Modify container images | Cosign/Sigstore verification, content-addressable registry | Phase 1-2 |
| **Repudiation** | User denies running destructive command | Local audit log in SQLite, operation timestamps | Phase 1-2 |
| **Information Disclosure** | Secrets leaked via environment variables | In-memory only decryption, no disk writes, secure file descriptors | Phase 1-2 |
| **Information Disclosure** | Logs contain sensitive data | Log sanitization, secret masking in log output | Phase 1-2 |
| **Information Disclosure** | Telemetry leaks project data | Anonymized, aggregated, opt-out, published schema | Phase 1-2 |
| **Information Disclosure** | Snapshot contains secrets | Snapshot encryption (AES-256-GCM), encrypted at rest | Phase 1-2 |
| **Information Disclosure** | Container escape exposes host filesystem | Rootless containers, capability dropping, seccomp profiles | Phase 1-2 |
| **Denial of Service** | Service consumes all host resources | Resource limits (memory, CPU, disk) with sane defaults | Phase 1-2 |
| **Denial of Service** | Log volume fills disk | Log rotation, 100MB cap per service, backpressure | Phase 1-2 |
| **Denial of Service** | Port conflict prevents service startup | Port conflict detection before startup, auto-resolution | Phase 1-2 |
| **Elevation of Privilege** | Container runs as root and escapes | Rootless by default, `no-new-privileges`, read-only root | Phase 1-2 |
| **Elevation of Privilege** | Plugin executes arbitrary code | Plugin sandboxing, restricted permissions, allowlist | Phase 1-2 |

### 14.2 Cloud-Specific Threats (Phase 3+)

| Threat Category | Attack Vector | Mitigation | Phase |
|---|---|---|---|
| **Spoofing** | Stolen OAuth token | Short-lived tokens, refresh token rotation, device flow | Phase 3 |
| **Tampering** | Unauthorized snapshot modification | Write-once S3 objects, immutable storage, audit trail | Phase 3 |
| **Information Disclosure** | Shared snapshot accessible by unauthorized user | Token-based access control, expiry, max uses | Phase 3 |
| **Denial of Service** | API flood | Rate limiting (per-user, per-IP), circuit breakers | Phase 3 |
| **Elevation of Privilege** | Team member accesses data outside role | RBAC, least-privilege defaults, audit logging | Phase 5 |

### 14.3 Attack Surface Map

```
┌─────────────────────────────────────────────────────┐
│                    Attack Surface                     │
│                                                      │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────┐ │
│  │ CLI Binary  │    │ Engine      │    │ Plugins  │ │
│  │ - Download  │    │ - gRPC API  │    │ - Loader │ │
│  │ - Execution │    │ - Docker    │    │ - Exec   │ │
│  └──────┬──────┘    └──────┬──────┘    └────┬─────┘ │
│         │                  │                 │       │
│  ┌──────▼──────────────────▼─────────────────▼─────┐ │
│  │              Local Machine                       │ │
│  │  - Filesystem access (scoped to project)         │ │
│  │  - Network access (per-project isolated)          │ │
│  │  - Process execution (containerized)              │ │
│  │  - Secrets (encrypted, in-memory only)            │ │
│  └──────────────────────────────────────────────────┘ │
│                                                      │
│  ┌──────────────────────────────────────────────────┐ │
│  │              Cloud (Phase 3+)                     │ │
│  │  - REST API (authenticated, rate-limited)         │ │
│  │  - S3 storage (encrypted, immutable)              │ │
│  │  - PostgreSQL (connection pooled, encrypted)      │ │
│  │  - Web dashboard (HTTPS, CSP, CSRF protection)    │ │
│  └──────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

### 14.4 Security Testing

- **Static analysis:** `gosec` on every commit
- **Dependency scanning:** `govulncheck` weekly
- **Container scanning:** Trivy on all base images
- **Penetration testing:** Annual third-party audit before Enterprise GA
- **Bug bounty:** HackerOne program after Public Beta

---

## 15. Architecture Decision Records

### ADR-001: Language Choice — Go for CLI and Engine

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Needed to choose between Go and Rust for the core platform. Rust offers better performance and memory safety, but Go has the dominant ecosystem in DevOps/container tooling.
**Decision:** Go for CLI, Engine, and Cloud API.
**Consequences:**
- Positive: Official Docker SDK, goroutines for concurrency, faster development, larger hiring pool
- Negative: Slightly larger binaries, GC pauses (negligible for this workload), no compile-time memory safety guarantees
**Alternatives considered:** Rust, Node.js

### ADR-002: Communication Protocol — gRPC for CLI ↔ Engine

**Status:** Accepted
**Date:** 2026-05-15
**Context:** CLI needs to communicate with the background engine daemon. Options: REST, gRPC, Unix socket with custom protocol, subprocess invocation.
**Decision:** gRPC over Unix socket (macOS/Linux) / named pipe (Windows).
**Consequences:**
- Positive: Strong typing, bidirectional streaming (for logs), auto-generated client code, versioned contracts
- Negative: gRPC dependency, requires protobuf compilation
**Alternatives considered:** REST/HTTP, custom binary protocol, subprocess with JSON stdin/stdout

### ADR-003: State Storage — SQLite for Local State

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Engine needs to persist environment state, snapshot metadata, and telemetry. Options: SQLite, JSON files, BoltDB, in-memory with file sync.
**Decision:** SQLite via `glebarez/go-sqlite` (pure Go, no CGO).
**Consequences:**
- Positive: ACID transactions, SQL queries, single file, no external dependencies, pure Go
- Negative: Not suitable for high-concurrency writes (not needed for local tool)
**Alternatives considered:** JSON files, BoltDB, BadgerDB

### ADR-004: Container Runtime — Docker First, containerd Later

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Need a container runtime. Docker is the most widely adopted but containerd is the lower-level standard.
**Decision:** Docker SDK for Phase 1-2, containerd abstraction layer for Phase 3+.
**Consequences:**
- Positive: Fastest time to market, largest user base, mature SDK
- Negative: Docker Desktop licensing for large enterprises (mitigated by containerd fallback)
**Alternatives considered:** containerd only, Podman, direct runc

### ADR-005: Secrets Encryption — age over GPG

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Need to encrypt `.env.devbox` files. GPG is the standard but complex. age is modern, simpler, and designed for this exact use case.
**Decision:** age (github.com/FiloSottile/age) for local secret encryption.
**Consequences:**
- Positive: Simple API, no GPG dependency, passphrase or public-key encryption, small binary
- Negative: Less widely known than GPG (but growing adoption)
**Alternatives considered:** GPG, libsodium, AES-256 with custom key management

### ADR-006: Cloud API Framework — Gin over net/http

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Cloud API needs an HTTP framework. Options: stdlib `net/http`, Gin, Echo, Fiber.
**Decision:** Gin for Phase 3.
**Consequences:**
- Positive: Mature, widely used, middleware ecosystem, good performance
- Negative: Additional dependency, slightly more complex than stdlib
**Alternatives considered:** Echo, Fiber, stdlib `net/http` with chi router

### ADR-007: Snapshot Format — OCI Artifact-Based `.devbox` Bundle

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Need a portable snapshot format. Options: custom tar.gz, OCI artifact, Docker image, custom binary.
**Decision:** tar.gz bundle with `manifest.json` at root, following OCI artifact conventions.
**Consequences:**
- Positive: Simple to implement, compatible with existing tools, easy to inspect, cloud-storage friendly
- Negative: Not a standard OCI index (but follows conventions)
**Alternatives considered:** Full OCI image, custom binary format, Borg-like archive

### ADR-008: Cloud Database — PostgreSQL over MongoDB

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Cloud backend needs a primary database. Options: PostgreSQL, MongoDB, MySQL, CockroachDB.
**Decision:** PostgreSQL.
**Consequences:**
- Positive: ACID compliance, JSONB support, mature ecosystem, strong Go drivers
- Negative: Requires operational expertise (mitigated by managed services)
**Alternatives considered:** MongoDB, MySQL, CockroachDB

### ADR-009: Monorepo Structure — Single Repo with Go Workspaces

**Status:** Accepted
**Date:** 2026-05-15
**Context:** How to organize CLI, Engine, Cloud, and shared code. Options: monorepo, separate repos, Go workspace.
**Decision:** Monorepo with Go workspaces (`go.work`).
**Consequences:**
- Positive: Single source of truth, atomic commits, easy cross-component changes
- Negative: Larger repo, CI builds everything (mitigated by path-based triggers)
**Alternatives considered:** Separate repos per component, Go submodules

### ADR-010: Frontend Framework — Next.js over React SPA

**Status:** Accepted
**Date:** 2026-05-15
**Context:** Web dashboard needs a framework. Options: Next.js, React SPA, Vue, Svelte.
**Decision:** Next.js with App Router.
**Consequences:**
- Positive: SSR for performance, API routes, large ecosystem, TypeScript
- Negative: Heavier than a SPA, Node.js dependency for build
**Alternatives considered:** React SPA, Vue/Nuxt, SvelteKit

---

## 16. Cost Estimation

### 16.1 Phase 1-2 (Local MVP) — Months 1-5

| Category | Monthly Cost | 5-Month Total | Notes |
|---|---|---|---|
| **Team (3 engineers)** | $30,000 | $150,000 | 1 senior Go, 1 mid Go, 1 platform engineer |
| **Cloud infra (CI/CD)** | $500 | $2,500 | GitHub Actions, codecov, artifact storage |
| **Development tools** | $200 | $1,000 | IDE licenses, Docker Desktop (dev), monitoring |
| **Legal & compliance** | $1,000 | $5,000 | Incorporation, IP, open source licensing |
| **Miscellaneous** | $300 | $1,500 | Domain, hosting, marketing assets |
| **Total** | **$31,500** | **$160,000** | |

*Funded by pre-seed ($750K). Runway: ~23 months at this burn rate.*

### 16.2 Phase 3 (Cloud Backend) — Months 6-9

| Category | Monthly Cost | 4-Month Total | Notes |
|---|---|---|---|
| **Team (5 engineers)** | $50,000 | $200,000 | +2 engineers (backend, frontend) |
| **Cloud infra (staging)** | $2,000 | $8,000 | AWS/GCP for API, DB, S3, Redis |
| **Cloud infra (production)** | $1,500 | $6,000 | Production environment |
| **CI/CD expansion** | $800 | $3,200 | Additional runners, e2e test infrastructure |
| **Security audit** | $5,000 | $5,000 | Third-party penetration test |
| **Total** | **$59,300** | **$222,200** | |

*Funded by seed round ($3M). Combined runway with Phase 1-2: ~18 months.*

### 16.3 Phase 4 (Cloud Compute) — Months 9-12

| Category | Monthly Cost | 4-Month Total | Notes |
|---|---|---|---|
| **Team (7 engineers)** | $70,000 | $280,000 | +2 engineers (K8s, billing) |
| **Cloud infra (compute)** | $5,000 | $20,000 | Kubernetes clusters for remote environments |
| **Stripe fees** | Variable | ~$5,000 | Payment processing (2.9% + $0.30) |
| **Monitoring** | $1,000 | $4,000 | Datadog/Grafana, alerting |
| **Total** | **$76,000+** | **$309,000+** | |

### 16.4 Phase 5 (Enterprise) — Months 12-18

| Category | Monthly Cost | 6-Month Total | Notes |
|---|---|---|---|
| **Team (10 engineers)** | $100,000 | $600,000 | +3 engineers (security, enterprise) |
| **Cloud infra** | $8,000 | $48,000 | Multi-region, Firecracker infrastructure |
| **SOC 2 certification** | $15,000 | $30,000 | Audit + compliance tools |
| **Legal** | $3,000 | $18,000 | Enterprise contracts, SLAs |
| **Total** | **$126,000** | **$696,000** | |

### 16.5 Total 18-Month Cost Summary

| Phase | Duration | Total Cost | Cumulative |
|---|---|---|---|
| Phase 1-2 | 5 months | $160,000 | $160,000 |
| Phase 3 | 4 months | $222,200 | $382,200 |
| Phase 4 | 4 months | $309,000 | $691,200 |
| Phase 5 | 6 months | $696,000 | $1,387,200 |

**Total 18-month cost: ~$1.4M**

**Funding coverage:**
- Pre-seed ($750K): Covers Phase 1-2 + partial Phase 3
- Seed ($3M): Covers Phase 3-4 + partial Phase 5
- Series A ($15M): Covers Phase 5 + scale

---

## 17. Error Taxonomy

### 17.1 Error Code Structure

All errors follow a structured format:

```
DEVBOX-<CATEGORY>-<CODE>
```

| Category | Code Range | Description |
|---|---|---|
| `CFG` | 001-099 | Configuration errors |
| `RUN` | 100-199 | Runtime/orchestration errors |
| `NET` | 200-299 | Networking errors |
| `SEC` | 300-399 | Security errors |
| `SNA` | 400-499 | Snapshot errors |
| `PLG` | 500-599 | Plugin errors |
| `CLD` | 600-699 | Cloud/API errors |
| `SYS` | 900-999 | System/environment errors |

### 17.2 Error Definitions

#### Configuration Errors (CFG)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| CFG-001 | error | `devbox.yml not found` | Run `devbox init` to generate a config |
| CFG-002 | error | Invalid YAML syntax: `<details>` | Fix YAML syntax at line `<N>` |
| CFG-003 | error | Unknown service type: `<type>` | Use `image`, `runtime`, or `build` |
| CFG-004 | error | Circular dependency: `<a>` → `<b>` → `<a>` | Remove circular dependency in `depends_on` |
| CFG-005 | error | Invalid port: `<port>` | Use a valid port number (1-65535) |
| CFG-006 | error | Duplicate service name: `<name>` | Rename one of the conflicting services |
| CFG-007 | warning | Deprecated field `<field>` — use `<replacement>` | Update config to use new field name |
| CFG-008 | error | Runtime `<runtime>` not available | Install runtime or use `image` instead |
| CFG-009 | error | Invalid resource limit: `<limit>` | Use format like "512m", "1g", "0.5" |
| CFG-010 | error | Config version `<v>` not supported | Upgrade DevBoxOS or use a supported config version |

#### Runtime Errors (RUN)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| RUN-001 | error | Docker daemon not running | Start Docker Desktop or `systemctl start docker` |
| RUN-002 | error | Failed to pull image `<image>` | Check image name, network, and registry auth |
| RUN-003 | error | Container `<name>` failed to start | Run `devbox logs <name>` for details |
| RUN-004 | error | Service `<name>` health check failed after `<N>` retries | Check service logs, increase `start_period` |
| RUN-005 | error | Dependency `<name>` is not running | Start dependency first: `devbox start <name>` |
| RUN-006 | error | Container exited with code `<code>` | Check application logs for crash reason |
| RUN-007 | warning | Service `<name>` restarted `<N>` times | Check for resource limits or application errors |
| RUN-008 | error | Volume mount failed: `<path>` not found | Create the directory or use a named volume |
| RUN-009 | error | OOM killed: service `<name>` exceeded `<limit>` | Increase memory limit or optimize application |
| RUN-010 | error | Disk space exhausted | Free disk space or reduce volume sizes |

#### Networking Errors (NET)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| NET-001 | error | Port `<port>` already in use by `<process>` (PID `<pid>`) | Kill process or use different port |
| NET-002 | error | Failed to create network: name conflict | Run `devbox reset` to clean up |
| NET-003 | error | DNS resolution failed for `<hostname>` | Check service is running and network is created |
| NET-004 | error | Egress denied: `<service>` → `<destination>` | Add egress rule in `networking.egress` |
| NET-005 | warning | mTLS certificate expired for `<service>` | Restart environment to regenerate certificates |
| NET-006 | error | Network namespace creation failed | Check OS permissions and Docker configuration |

#### Security Errors (SEC)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| SEC-001 | error | Failed to decrypt secrets: invalid passphrase | Check `.env.devbox.age` passphrase |
| SEC-002 | error | Image `<image>` failed signature verification | Use `--skip-verify` (not recommended) or use signed image |
| SEC-003 | warning | Service `<name>` requests elevated capabilities | Review if capability is truly needed |
| SEC-004 | error | Secret `<name>` not found in vault | Check vault path and authentication |
| SEC-005 | error | Certificate authority initialization failed | Clean `~/.devbox/certs` and restart |
| SEC-006 | warning | CVE-`<id>` detected in `<dependency>` | Update dependency to patched version |

#### Snapshot Errors (SNA)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| SNA-001 | error | Failed to capture snapshot: database dump failed | Ensure database service is running |
| SNA-002 | error | Snapshot integrity check failed: hash mismatch | Snapshot may be corrupted, try another |
| SNA-003 | error | Snapshot signature verification failed | Snapshot was modified after creation |
| SNA-004 | error | Snapshot format version `<v>` not supported | Upgrade DevBoxOS to load this snapshot |
| SNA-005 | error | Insufficient disk space for snapshot (need `<size>`) | Free disk space or use smaller snapshot |
| SNA-006 | error | Failed to restore snapshot: incompatible service versions | Update services to match snapshot versions |

#### Plugin Errors (PLG)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| PLG-001 | error | Plugin `<name>` not found | Run `devbox plugin install <name>` |
| PLG-002 | error | Plugin `<name>` version `<v>` incompatible with DevBoxOS `<v>` | Update plugin or DevBoxOS |
| PLG-003 | error | Plugin `<name>` exceeded sandbox permissions | Contact plugin author or review permissions |
| PLG-004 | warning | Plugin `<name>` is deprecated | Migrate to replacement plugin |

#### Cloud Errors (CLD) — Phase 3+

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| CLD-001 | error | Not authenticated | Run `devbox auth login` |
| CLD-002 | error | Authentication expired | Run `devbox auth login` to refresh |
| CLD-003 | error | Snapshot upload failed: `<reason>` | Check network and retry |
| CLD-004 | error | Share token expired | Request a new share from the owner |
| CLD-005 | error | Rate limit exceeded | Wait and retry, or upgrade plan |
| CLD-006 | error | Team quota exceeded | Upgrade team plan or remove unused snapshots |

#### System Errors (SYS)

| Code | Severity | Message | Suggested Fix |
|---|---|---|---|
| SYS-001 | error | Failed to start engine daemon | Check `~/.devbox/daemon.log` for details |
| SYS-002 | error | State database corrupted | Run `devbox reset --state` to rebuild |
| SYS-003 | error | Lock file stale: previous operation crashed | Run `devbox reset --locks` |
| SYS-004 | warning | DevBoxOS update available: `<version>` | Run `devbox update` |
| SYS-005 | error | Unsupported OS: `<os>` `<version>` | Upgrade OS or check compatibility matrix |
| SYS-006 | error | Insufficient permissions | Run with appropriate permissions (not root) |

### 17.3 Error Output Format

```
✗ DEVBOX-RUN-003: Container 'api' failed to start

  Reason: Application crashed during startup with exit code 1

  Service: api
  Container: abc123def456
  Exit code: 1

  Recent logs:
    > node:internal/modules/cjs/loader:1080
    >   throw err;
    >   ^
    > Error: Cannot find module './config'

  Suggested fix:
    → Check that all files are present in the working directory
    → Run: devbox logs api --tail 50
    → Or manually: docker logs abc123def456
```

---

## 18. Performance Targets

### 18.1 CLI Performance

| Metric | Target | Measurement |
|---|---|---|
| CLI startup time (cold) | < 50ms | `time devbox version` |
| CLI startup time (warm, daemon running) | < 20ms | `time devbox status` |
| Binary size (Linux amd64) | < 30MB | `ls -lh dist/devbox-linux-amd64` |
| Memory usage (idle daemon) | < 50MB | `ps aux | grep devbox-engine` |
| Memory usage (active, 5 services) | < 200MB | `ps aux | grep devbox-engine` |

### 18.2 Service Orchestration Performance

| Metric | Target | Measurement |
|---|---|---|
| `devbox start` (single service) | < 3s | Time to service healthy |
| `devbox start` (5 services) | < 10s | Time to all services healthy |
| `devbox stop` (all services) | < 5s | Time to all containers stopped |
| `devbox status` | < 100ms | Time to output |
| Config parsing (large file, 20 services) | < 50ms | Parse + validate time |
| Dependency graph resolution (20 services) | < 10ms | Topological sort time |

### 18.3 Snapshot Performance

| Metric | Target | Measurement |
|---|---|---|
| Snapshot save (small project, no DB data) | < 5s | Time to archive |
| Snapshot save (with 1GB DB dump) | < 30s | Time to archive |
| Snapshot load (small project) | < 10s | Time to restore |
| Snapshot load (with DB restore) | < 60s | Time to restore |
| Snapshot integrity verification | < 2s | SHA-256 check time |

### 18.4 Networking Performance

| Metric | Target | Measurement |
|---|---|---|
| DNS resolution (service.local) | < 5ms | `dig api.local` |
| Inter-service latency (same host) | < 1ms | `curl` between services |
| Network creation | < 2s | Time to create Docker network |
| Port conflict detection | < 500ms | Time to scan for conflicts |

### 18.5 Cloud API Performance (Phase 3+)

| Metric | Target | Measurement |
|---|---|---|
| API response time (authenticated) | < 200ms (p95) | Load test |
| API response time (unauthenticated) | < 50ms (p95) | Load test |
| Snapshot upload (100MB) | < 30s | Upload time |
| Snapshot download (100MB) | < 15s | Download time |
| Concurrent users supported | 10,000+ | Load test |

### 18.6 Performance Testing

```bash
# Benchmark suite (run nightly)
make benchmark

# Outputs:
# - CLI startup time
# - Service start time (various configurations)
# - Snapshot save/load time
# - Memory usage profiles
# - Disk usage profiles
```

Performance regressions > 10% on any metric block release.

---

## 19. Cloud Observability

### 19.1 Metrics (Phase 3+)

| Metric | Type | Description | Alert Threshold |
|---|---|---|---|
| `api_requests_total` | Counter | Total API requests by endpoint, status | — |
| `api_request_duration_seconds` | Histogram | API request latency | p99 > 1s |
| `api_errors_total` | Counter | API errors by type | Rate > 1/min |
| `active_users` | Gauge | Currently authenticated users | — |
| `snapshot_storage_bytes` | Gauge | Total snapshot storage used | > 80% quota |
| `snapshot_uploads_total` | Counter | Snapshot uploads by status | Failure rate > 5% |
| `share_tokens_created_total` | Counter | Share tokens created | — |
| `share_tokens_expired_total` | Counter | Share tokens expired | — |
| `db_connection_pool_active` | Gauge | Active DB connections | > 80% pool |
| `db_query_duration_seconds` | Histogram | Database query latency | p99 > 500ms |
| `redis_operations_total` | Counter | Redis operations by type | Error rate > 1% |
| `s3_operations_total` | Counter | S3 operations by type | Error rate > 1% |
| `cpu_usage_percent` | Gauge | API server CPU usage | > 80% for 5min |
| `memory_usage_bytes` | Gauge | API server memory usage | > 80% limit |
| `rate_limit_rejections_total` | Counter | Rate-limited requests | Spike > 10x normal |

### 19.2 Logging

- **Format:** JSON structured logs
- **Fields:** `timestamp`, `level`, `service`, `request_id`, `user_id`, `message`, `duration_ms`, `status_code`
- **Aggregation:** OpenTelemetry collector → Grafana Loki
- **Retention:** 30 days for raw logs, 90 days for aggregated metrics

### 19.3 Tracing

- **System:** OpenTelemetry + Jaeger
- **Sampling:** 10% of requests (100% for errors)
- **Spans:** API request → auth → DB query → S3 operation → response

### 19.4 Alerting

| Alert | Condition | Action |
|---|---|---|
| API down | Health check fails 3x in 1min | Page on-call |
| High error rate | > 5% of requests return 5xx in 5min | Page on-call |
| Database down | Connection failures > 3 in 1min | Page on-call |
| Disk full | Storage > 90% capacity | Warn ops team |
| Slow queries | p99 DB query > 2s for 10min | Warn engineering |
| Rate limit spike | Rejections > 10x baseline for 5min | Investigate abuse |

### 19.5 Dashboard

```
┌─────────────────────────────────────────────────────┐
│                DevBoxOS Cloud Dashboard               │
├─────────────────────────────────────────────────────┤
│  Requests/s: 1,234    │  p99 Latency: 145ms         │
│  Error Rate: 0.2%     │  Active Users: 3,456        │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐           │
│  │  Request Rate   │  │  Latency (p99)  │           │
│  │  [graph]        │  │  [graph]        │           │
│  └─────────────────┘  └─────────────────┘           │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐           │
│  │  Storage Usage  │  │  Active Teams   │           │
│  │  2.3TB / 5TB    │  │  1,234          │           │
│  └─────────────────┘  └─────────────────┘           │
├─────────────────────────────────────────────────────┤
│  Recent Alerts:                                      │
│  [14:32] Rate limit spike on /api/v1/shares          │
│  [12:15] Slow query detected on snapshots table      │
└─────────────────────────────────────────────────────┘
```

---

## 20. Disaster Recovery

### 20.1 Local (Phase 1-2)

| Scenario | Impact | Recovery |
|---|---|---|
| SQLite state DB corrupted | Lost environment state metadata | Run `devbox reset --state` — engine reconciles with Docker runtime |
| Lock file stale | Cannot run any command | Run `devbox reset --locks` — clears stale locks |
| Orphaned containers | Resources consumed, `devbox status` inaccurate | Run `devbox reset` — cleans up and rebuilds |
| Snapshot corrupted | Cannot restore specific snapshot | Use another snapshot or rebuild from config |
| Engine daemon crash | CLI commands fail | Daemon auto-restarts on next CLI invocation |
| Config file deleted | Cannot start environment | Run `devbox init` to regenerate, or restore from version control |

### 20.2 Cloud (Phase 3+)

| Scenario | Impact | Recovery | RTO | RPO |
|---|---|---|---|---|
| API server crash | All API requests fail | Auto-restart via health check, load balancer routes to healthy instance | < 1min | 0 |
| PostgreSQL failure | All data operations fail | Failover to read replica, promote to primary | < 5min | < 1min |
| S3 outage | Snapshot upload/download fails | Retry with exponential backoff, queue for later | < 30min | 0 |
| Redis failure | Session/cache loss | Auto-reconnect, sessions re-authenticate | < 1min | 0 |
| Data center outage | All services unavailable | Failover to secondary region | < 15min | < 5min |
| Snapshot data loss | User snapshots unavailable | Restore from cross-region backup | < 1hr | 24hr |
| Security breach | Data potentially exposed | Incident response, rotate all keys, notify users | < 1hr | 0 |

### 20.3 Backup Strategy

| Data | Backup Method | Frequency | Retention | Location |
|---|---|---|---|---|
| PostgreSQL | pg_dump + WAL archiving | Continuous | 30 days | Cross-region S3 |
| S3 snapshots | Cross-region replication | Real-time | Per user plan | Secondary region |
| Redis | RDB snapshots | Every 6 hours | 7 days | Local + S3 |
| Config files | Git version control | Every change | Unlimited | GitHub |

### 20.4 DR Testing

- **Monthly:** Restore PostgreSQL from backup to staging environment
- **Quarterly:** Full failover test to secondary region
- **Annually:** Tabletop exercise for security incident response

---

## 21. Rate Limiting & Abuse Prevention

### 21.1 Rate Limit Tiers

| Tier | Requests/min | Snapshot uploads/day | Share tokens/day | Storage |
|---|---|---|---|---|
| **Free** | 60 | 10 | 1 | 1 GB |
| **Pro** | 300 | 100 | 50 | 50 GB |
| **Team** | 1,000 | 1,000 | Unlimited | 500 GB |
| **Enterprise** | 10,000 | Unlimited | Unlimited | Custom |

### 21.2 Rate Limiting Implementation

```go
// Rate limiter middleware (cloud/api/internal/auth/middleware.go)
type RateLimiter struct {
    store    redis.Client    // Distributed rate limit store
    limits   map[string]Tier // Per-tier limits
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r)
        tier := getUserTier(userID)
        limit := rl.limits[tier]

        key := fmt.Sprintf("ratelimit:%s:%s", userID, r.URL.Path)
        count, err := rl.store.Incr(key)
        if err != nil {
            next.ServeHTTP(w, r) // Fail open
            return
        }

        if count > limit.RequestsPerMinute {
            w.Header().Set("Retry-After", "60")
            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.RequestsPerMinute))
            w.Header().Set("X-RateLimit-Remaining", "0")
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.RequestsPerMinute))
        w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit.RequestsPerMinute-int(count)))
        next.ServeHTTP(w, r)
    })
}
```

### 21.3 Abuse Prevention Measures

| Threat | Prevention | Detection |
|---|---|---|
| Brute force auth | Account lockout after 10 failures, CAPTCHA | Alert on > 100 failures/hour from single IP |
| Snapshot spam | Per-user upload limits, file size limits | Alert on > 50 uploads/hour from single user |
| Share token abuse | Token expiry, max uses, IP-based throttling | Alert on > 100 share creations/hour |
| API scraping | Rate limiting, pagination limits | Alert on > 1000 requests/min from single user |
| Storage abuse | Per-user quota enforcement | Alert on > 90% quota usage |
| DDoS | Cloudflare WAF, rate limiting at edge | Automatic scaling, alert on traffic spike |

### 21.4 Response Headers

All API responses include rate limit headers:

```
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 295
X-RateLimit-Reset: 1715789400
Retry-After: 60  (only when rate limited)
```

---

## Appendix A: Current Build Scope (Phase 1-2 Only)

**What we are building NOW:**

```
Phase 1-2: Local MVP
├── cli/                    # ✅ In scope
├── engine/                 # ✅ In scope
├── shared/                 # ✅ In scope
├── plugins/                # ✅ In scope (basic)
├── docs/                   # ✅ In scope
├── tests/                  # ✅ In scope
└── scripts/                # ✅ In scope

cloud/                      # ❌ NOT in scope (documented for planning)
```

**What we are NOT building yet:**

- Cloud API server
- Web dashboard
- Snapshot cloud storage
- Environment sharing
- Team workspace
- Remote compute
- CI integration
- Billing
- RBAC/SSO
- Firecracker runtime

---

## Appendix B: Quick Start Development

```bash
# Clone the repo
git clone https://github.com/devboxos/devboxos.git
cd devboxos

# Set up Go workspace
go work init
go work use ./cli ./engine ./shared

# Build CLI
cd cli
go build -o devbox .

# Build Engine
cd ../engine
go build -o devbox-engine .

# Run engine daemon
./devbox-engine --daemon

# Use CLI
cd ../cli
./devbox version
./devbox init
./devbox start
```

---

*Document Version: 2.0 — Complete End-to-End Blueprint*
*Created: 2026-05-15*
*Status: Ready for Implementation*
