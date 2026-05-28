# DevBoxOS — Comprehensive Testing Strategy

> **Goal:** 100% confidence that all local v1 features work correctly across Windows, macOS, and Linux — with automated testing at every layer of the stack.

---

## Table of Contents

1. [Test Pyramid Overview](#1-test-pyramid-overview)
2. [Unit Tests](#2-unit-tests)
3. [Integration Tests](#3-integration-tests)
4. [End-to-End (E2E) Tests](#4-end-to-end-e2e-tests)
5. [Smoke Tests](#5-smoke-tests)
6. [Compatibility Tests](#6-compatibility-tests)
7. [Performance Benchmarks](#7-performance-benchmarks)
8. [Security Tests](#8-security-tests)
9. [Manual Smoke Test Sequence](#9-manual-smoke-test-sequence)
10. [CI Pipeline Integration](#10-ci-pipeline-integration)
11. [Test Infrastructure](#11-test-infrastructure)
12. [Coverage Targets & Tracking](#12-coverage-targets--tracking)
13. [Test Fixtures](#13-test-fixtures)
14. [Edge Cases & Error Paths](#14-edge-cases--error-paths)

---

## 1. Test Pyramid Overview

```
            ╱╲
           ╱  ╲
          ╱ E2E╲
         ╱──────╲
        ╱Integration╲
       ╱──────────────╲
      ╱   Smoke Tests  ╲
     ╱──────────────────╲
    ╱    Unit Tests      ╲
   ╱──────────────────────╲
  ╱   Compilation Checks   ╲
 ╱──────────────────────────╲
╱    Static Analysis / Lint  ╲
╱─────────────────────────────╲
```

| Layer | Scope | Tools | Target Coverage | Run Frequency |
|-------|-------|-------|-----------------|---------------|
| **Static Analysis** | Format, lint, vet, security scan | `golangci-lint`, `go vet`, `staticcheck`, `gosec` | All source files | Every commit / CI |
| **Compilation** | All packages compile | `go build ./...` | 100% | Every commit / CI |
| **Unit** | Individual functions, logic branches | Go `testing`, `testify/assert` | 80%+ | Every commit / CI |
| **Smoke** | Basic CLI commands against real Docker | Custom test runner | All 30+ CLI commands | Every commit / CI |
| **Integration** | CLI ↔ Engine ↔ Docker interaction | Go `testing`, `testcontainers-go` | All critical paths | Every commit / CI |
| **E2E** | Full workflows (init → start → snapshot → restore → destroy) | Custom test harness | Core user flows | Daily / Release |
| **Compatibility** | OS matrix, Docker versions | Matrix CI | All supported combos | Weekly / Release |
| **Performance** | Startup time, memory, disk usage | `testing`, `benchmark` | Tracked per release | Release |

---

## 2. Unit Tests

### 2.1 `shared/config` — Config Parser & Validator

| Test Case | Description | Input | Expected Output |
|-----------|-------------|-------|-----------------|
| `TestParser_Parse_ValidYAML` | Parse a valid devbox.yml | Minimal YAML with name + services | Parsed Config struct, no error |
| `TestParser_Parse_MissingFile` | File does not exist | Non-existent path | Error: file not found |
| `TestParser_Parse_InvalidYAML` | Malformed YAML content | Invalid syntax | Parse error with details |
| `TestParser_Parse_EmptyFile` | Empty YAML file | Empty string | Error: empty config |
| `TestParser_Parse_RequiredFields` | Missing required fields | Config without `name` | Validation error |
| `TestParser_Parse_UnknownFields` | Extraneous fields that should be ignored | Config with extra unknown fields | Parsed without error (forward compat) |
| `TestParser_Parse_MaxDepth` | Deeply nested config | 50 levels of nesting | Parsed |
| `TestParser_Generate_NewProject` | Generate a default devbox.yml | Directory + project name | File created, valid YAML |
| `TestParser_Generate_ExistingFile` | File already exists | Existing file | Error: already exists |
| `TestParser_Generate_InvalidDir` | Directory does not exist | Non-existent path | Error |
| `TestValidator_Validate_ValidConfig` | Full valid config | Multi-service config with all fields | No errors |
| `TestValidator_Validate_MissingServices` | Empty services list | Config with services: [] | Validation error |
| `TestValidator_Validate_ServiceNoImage` | Service without image, runtime, or build | Service with none of the three | Validation error |
| `TestValidator_Validate_InvalidPortFormat` | Malformed port mapping | `port: "abc"` | Error |
| `TestValidator_Validate_SchemaFileMissing` | Schema file not found | No schema on disk | Graceful degradation (skips schema) |
| `TestValidator_Validate_SchemaViolation` | Config violates JSON schema | Wrong type for a field | Schema validation error |

### 2.2 `shared/platform` — Platform Detection

| Test Case | Description | Expected |
|-----------|-------------|----------|
| `TestDetect_Windows` | Goos=windows | PlatformWindows |
| `TestDetect_Darwin` | Goos=darwin | PlatformDarwin |
| `TestDetect_Linux` | Goos=linux | PlatformLinux |
| `TestIsWindows_True` | On Windows | true |
| `TestIsWindows_False` | On macOS/Linux | false |
| `TestConfigDir_Windows` | Windows %APPDATA% | `C:\Users\user\AppData\Roaming\DevBoxOS` |
| `TestConfigDir_Unix` | macOS/Linux $HOME | `/home/user/.config/devboxos` |
| `TestDataDir_Windows` | Windows | uses LocalAppData |
| `TestDataDir_Unix` | macOS/Linux | uses `~/.local/share/devboxos` |
| `TestEngineSocketPath_Unix` | On Unix | `~/.devbox/engine.sock` |
| `TestEngineSocketPath_Windows` | On Windows | `127.0.0.1:51000` (TCP) |
| `TestEngineAddress_Windows` | Returns bare IP:port | `127.0.0.1:51000` (no tcp:// prefix) |
| `TestEngineAddress_Unix` | Returns unix:// path | `unix:///home/user/.devbox/engine.sock` |
| `TestDefaultEnginePort` | Default port | `51000` |
| `TestDockerSocketPath_Linux` | On Linux | `/var/run/docker.sock` |
| `TestDockerSocketPath_Windows` | On Windows | `//./pipe/docker_engine` |
| `TestDockerSocketPath_Darwin` | On macOS | `~/.docker/run/docker.sock` |
| `TestNormalizePath_Windows` | Forward to backslash | `C:\foo\bar` |
| `TestNormalizePath_Unix` | No change | `/foo/bar` |
| `TestHomeDir` | Returns os.UserHomeDir | Varies by OS |

### 2.3 `shared/secrets` — Secrets Store & Encryption

| Test Case | Description | Expected |
|-----------|-------------|----------|
| `TestAgeCrypto_GenerateKey` | Generate a new age identity | Valid key pair |
| `TestAgeCrypto_EncryptDecrypt` | Round-trip encryption | Plaintext matches after decrypt |
| `TestAgeCrypto_LoadOrCreateKey_New` | Key file doesn't exist | Created + saved |
| `TestAgeCrypto_LoadOrCreateKey_Existing` | Key file exists | Loaded without changes |
| `TestAgeCrypto_InvalidKeyFile` | Corrupted key file | Error |
| `TestStore_SetGet` | Set and get a secret | Value matches |
| `TestStore_Set_Overwrite` | Overwrite existing secret | Updated value and timestamp |
| `TestStore_Get_NotFound` | Get non-existent secret | Error or empty |
| `TestStore_List` | List all secrets | Correct count, values masked as `****` |
| `TestStore_List_Empty` | Empty store | Empty list |
| `TestStore_Delete` | Delete a secret | Gone from store |
| `TestStore_Delete_NotExist` | Delete non-existent | No error (idempotent) |
| `TestStore_Persistence` | Save, reload, verify | Data survives reload |
| `TestStore_Persistence_Empty` | Empty store saved and reloaded | Clean state |
| `TestStore_ConcurrentWrite` | Concurrent Set operations | No data races |
| `TestResolver_Resolve_Generate` | Resolve a generated secret | Random string of correct length |
| `TestResolver_Resolve_Env` | Resolve from environment | Matches env var |
| `TestResolver_Resolve_File` | Resolve from file | File contents |
| `TestResolver_Resolve_Generate_AlphaNumeric` | Generate alphanumeric | Only alphanumeric chars |
| `TestResolver_Rotate` | Rotate a generated secret | New value, different from old |
| `TestResolver_SetGetListDelete` | Full CRUD on resolver | All operations succeed |
| `TestEnvProvider_Resolve_Missing` | Env var not set | Error or empty |
| `TestFileProvider_Resolve_Missing` | File not found | Error |
| `TestRegistry_Resolve_Priority` | Multiple providers, correct priority | Higher priority wins |

### 2.4 `engine/internal/orchestrator` — Graph & Dependency Resolution

| Test Case | Description | Expected |
|-----------|-------------|----------|
| `TestGraph_AddNode_Resolve` | Simple linear dependency | `A → B → C` |
| `TestGraph_Reverse` | Reverse the graph order | Correct topological sort |
| `TestGraph_Empty` | No nodes | Empty result |
| `TestGraph_Deterministic` | Same input always same output | Deterministic ordering |
| `TestGraph_CircularDependency` | A→B→A | Error: cycle detected |
| `TestGraph_Disconnected` | Two independent chains | Both resolved, order preserved |
| `TestGraph_Diamond` | A→B, A→C, B→D, C→D | All dependencies before dependents |
| `TestGraph_SingleNode` | Just one node | Returns that node |
| `TestGraph_DeepChain` | 1000 nodes in sequence | All resolved, no stack overflow |
| `TestGraph_DuplicateEdges` | Same edge added twice | No duplicates in output |
| `TestGraph_MissingNode` | Edge references non-existent node | Error |

### 2.5 `engine/internal/orchestrator` — Lifecycle & Recovery

| Test Case | Description | Expected |
|-----------|-------------|----------|
| `TestLifecycle_StartStop` | Full start/stop cycle | All services started, then stopped |
| `TestLifecycle_DependencyOrder` | Services start in dependency order | Dependencies start first |
| `TestLifecycle_FailedDependency` | A dependency fails to start | Dependent not started, error reported |
| `TestRecovery_OrphanedContainers` | Detect containers from crashed engine | Cleaned up on restart |
| `TestRecovery_StaleLock` | Expired lock file | Removed and re-acquired |
| `TestRecovery_StaleNetwork` | Orphaned Docker network | Removed on cleanup |
| `TestRecovery_StaleVolumes` | Orphaned volumes | Removed on cleanup |

### 2.6 `shared/snapshot` — Manager (Unit)

| Test Case | Description | Expected |
|-----------|-------------|----------|
| `TestManager_New` | Constructor | Valid Manager, non-nil store |
| `TestManager_CalculateSnapshotDir` | Path generation | Correct path |
| `TestManager_EmptyList` | No snapshots | Empty list |
| `TestManager_Delete_NonExistent` | Delete non-existent snapshot | Error or no-op |
| `TestManifest_Serialize_Deserialize` | Round-trip manifest JSON | All fields preserved |
| `TestManifest_Validate` | Valid manifest | No error |
| `TestManifest_Validate_Corrupted` | Corrupted manifest | Validation error |

### 2.7 `cli/` — CLI Commands (NEW — zero tests currently)

| Test Case | Description | Method |
|-----------|-------------|--------|
| `TestVersionCmd` | Version output | Capture stdout, check format |
| `TestInitCmd` | Init creates devbox.yml | Run in temp dir, check file |
| `TestInitCmd_ExistingDir` | Init where file exists | Error: already exists |
| `TestValidateCmd_Valid` | Validate a valid config | Exit 0 |
| `TestValidateCmd_Invalid` | Validate an invalid config | Non-zero exit |
| `TestValidateCmd_NoFile` | Validate with no devbox.yml | Error: file not found |
| `TestConfigGetCmd` | Get a config key | Correct value |
| `TestConfigSetCmd` | Set a config key | Value persisted |
| `TestConfigGetCmd_MissingKey` | Get non-existent key | Empty or error |
| `TestCompletionCmd` | Generate completions | Valid shell script output |
| `TestComposeImportCmd` | Import compose file | devbox.yml created |
| `TestComposeImportCmd_NoFile` | No docker-compose.yml | Error |
| `TestDoctorCmd_NoEngine` | Doctor without engine running | Graceful degradation |
| `TestSecretsListCmd` | List secrets | Formatted output |
| `TestSecretsGetCmd` | Get specific secret | Value displayed |
| `TestSecretsAddCmd` | Add a secret | Persisted |
| `TestSecretsDeleteCmd` | Delete a secret | Removed |
| `TestSecretsRotateCmd` | Rotate a generated secret | New value |
| `TestRootCmd_Help` | `devbox help` or `devbox --help` | All commands listed |
| `TestRootCmd_NoArgs` | `devbox` with no args | Help output shown |

### 2.8 Additional Unit Tests Needed

| Package | Test Cases |
|---------|------------|
| `shared/config` | Test autodetect (`package.json`, `go.mod`, `requirements.txt`, `Cargo.toml`) |
| `shared/config` | Test `ConfigToMap` for schema validation |
| `shared/diagnostics` | Test `Checker.Run()` with mocked runtime |
| `shared/diagnostics` | Test each diagnostic check individually (Docker, config, secrets, plugins) |
| `shared/logging` | Test `Store.Append` and `Store.Stream` |
| `shared/logging` | Test log rotation (max size, max files) |
| `shared/logging` | Test concurrent append |
| `shared/plugins` | Test `Manager.RunHooks` with mock commands |
| `shared/plugins` | Test plugin timeout |
| `shared/plugins` | Test env variable substitution (`$DEVBOX_PROJECT_NAME`) |
| `engine/internal/state` | Test SQLite state init + schema creation |
| `engine/internal/state` | Test Set/Get/Delete state keys |
| `engine/internal/state` | Test concurrent state access |
| `engine/internal/state` | Test lock Acquire/Release/Expiry |
| `engine/internal/state` | Test lock concurrent (only one acquires) |
| `engine/internal/networking` | Test DNS resolver add/remove/resolve |
| `engine/internal/networking` | Test mTLS CA generation |
| `engine/internal/networking` | Test mTLS cert signing + validation |
| `engine/internal/networking` | Test network manager create/remove/exists |

---

## 3. Integration Tests

Integration tests exercise real interactions between components, requiring a Docker daemon and optionally a running engine.

### 3.1 Docker Runtime Integration (`shared/runtime/docker`)

| Test Case | Description | Setup | Verification |
|-----------|-------------|-------|-------------|
| `TestConnect_Default` | Connect to Docker daemon | Docker running | No error |
| `TestConnect_NoDocker` | Connect without Docker | Docker not running | Specific error message |
| `TestPullImage_Valid` | Pull a known image | `alpine:latest` | Image exists locally |
| `TestPullImage_Invalid` | Pull non-existent image | `invalid/image:fake` | Error |
| `TestPullImage_AlreadyExists` | Image already cached | Already pulled | Fast (no download) |
| `TestBuildImage_Simple` | Build a simple Dockerfile | Temp dir with Dockerfile | Image built, ID returned |
| `TestBuildImage_NoDockerfile` | No Dockerfile in context | Empty dir | Error |
| `TestBuildImage_WithBuildArgs` | Build with --build-arg | ARG in Dockerfile | Correct value in image |
| `TestBuildImage_NoCache` | Build with --no-cache | --no-cache: true | Fresh build |
| `TestBuildImage_Pull` | Build with --pull | --pull: true | Base image re-pulled |
| `TestBuildImage_TargetStage` | Multi-stage build target | Dockerfile with stages | Correct stage built |
| `TestCreateContainer_Basic` | Create and start a container | `alpine echo hello` | Container runs, exits 0 |
| `TestCreateContainer_WithPorts` | Port mapping | `nginx` with 8080:80 | Port accessible |
| `TestCreateContainer_WithEnv` | Environment variables | `env` command | Correct env vars |
| `TestCreateContainer_WithVolumes` | Volume mounting | Write to volume, read back | Data persists |
| `TestCreateContainer_WithLabels` | Labels on container | Custom labels | Labels present |
| `TestCreateContainer_WithMemoryLimit` | Memory limit | `--memory=128m` | Limit enforced |
| `TestCreateContainer_WithCPULimit` | CPU limit | `--cpus=0.5` | Limit enforced |
| `TestCreateContainer_InvalidImage` | Non-existent image | `no-such-image:invalid` | Error |
| `TestCreateContainer_PortConflict` | Port already in use | Two containers, same port | Error: port conflict |
| `TestStartStopContainer` | Start then stop | `nginx`, wait, stop | Status: created → running → exited |
| `TestStopContainer_Timeout` | Stop with timeout | Long-running process | Killed after timeout |
| `TestStopContainer_AlreadyStopped` | Stop already stopped | Stopped container | No error (idempotent) |
| `TestRemoveContainer` | Create + remove | Container created | Removed (docker ps -a not listed) |
| `TestRemoveContainer_Running` | Remove running (force=false) | Running container | Error |
| `TestRemoveContainer_RunningForce` | Remove running (force=true) | Running container | Removed |
| `TestGetContainerInfo` | Inspect container | Created container | Correct name, status, ports, labels |
| `TestGetContainerInfo_NotFound` | Non-existent ID | Random ID | Error |
| `TestListContainers_ByLabels` | Filter by labels | Containers with matching labels | Correct subset |
| `TestListContainers_NoMatch` | Filter with no match | Labels with no match | Empty list |
| `TestStreamLogs_Follow` | Stream logs in real-time | Container writing to stdout | Lines received |
| `TestStreamLogs_Tail` | Tail last N lines | Container with 100 log lines | Last N lines |
| `TestStreamLogs_Since` | Logs since timestamp | Container with timed logs | Only recent entries |
| `TestStreamLogs_StoppedContainer` | Logs from stopped container | Container that exited | Logs still available |
| `TestCreateNetwork` | Create Docker network | Network name | Network exists |
| `TestRemoveNetwork` | Remove Docker network | Created network | Network gone |
| `TestRemoveNetwork_InUse` | Network with connected containers | Containers attached | Error or detach first |
| `TestNetworkExists_True` | Network exists | Created network | true |
| `TestNetworkExists_False` | Network does not exist | Random name | false |
| `TestCreateVolume` | Create Docker volume | Volume name | Volume exists |
| `TestRemoveVolume` | Remove Docker volume | Created volume | Volume gone |
| `TestVolumeExists_True` | Volume exists | Created volume | true |
| `TestVolumeExists_False` | Volume does not exist | Random name | false |

### 3.2 Snapshot Integration (`shared/snapshot`)

| Test Case | Description | Verification |
|-----------|-------------|-------------|
| `TestSnapshotSaveLoad_RoundTrip` | Save then load a snapshot with real Docker volumes | Volume data preserved |
| `TestSnapshotSaveLoad_WithSecrets` | Snapshot includes secrets | Secrets restored correctly |
| `TestSnapshotSaveLoad_MultipleVolumes` | Multiple service volumes | All volumes restored |
| `TestSnapshotSave_WithLogs` | Include logs in snapshot | Logs present in snapshot |
| `TestSnapshotList_AfterSave` | List after saving | Snapshot appears in list |
| `TestSnapshotDelete` | Save then delete | Snapshot removed from list |
| `TestSnapshotExportImport` | Export to .tar.gz, delete, import | Data restored |
| `TestSnapshotExportImport_CrossMachine` | Export on one host, import on another | Environment identical |
| `TestSnapshotLoad_NonExistent` | Load non-existent snapshot | Error |
| `TestSnapshotOverwrite` | Save with same name twice | Second overwrites, both not lost |

### 3.3 Engine gRPC Integration

| Test Case | Description | Verification |
|-----------|-------------|-------------|
| `TestGRPC_Ping` | Ping the engine | Pong |
| `TestGRPC_StartStop` | Start then stop a project | All services started, then stopped |
| `TestGRPC_Start_InvalidConfig` | Start with invalid devbox.yml | Error with details |
| `TestGRPC_Start_AlreadyRunning` | Start when already started | Error or no-op |
| `TestGRPC_Stop_NotRunning` | Stop when nothing running | No error (idempotent) |
| `TestGRPC_Status_Running` | Status when running | Correct service states |
| `TestGRPC_Status_Stopped` | Status when stopped | All services stopped |
| `TestGRPC_Logs_Streaming` | Stream logs from running service | Log lines received in real-time |
| `TestGRPC_Logs_Tail` | Get last N log lines | Correct number of lines |
| `TestGRPC_Logs_StoppedService` | Logs from stopped service | Historical logs available |
| `TestGRPC_Logs_NonExistentService` | Logs for non-existent service | Error |
| `TestGRPC_Doctor_DockerRunning` | Doctor with Docker running | Healthy |
| `TestGRPC_Doctor_ConfigCheck` | Doctor with valid config | No config issues |
| `TestGRPC_Reset` | Reset a running environment | All resources cleaned, ready to restart |
| `TestGRPC_SnapshotSaveLoad` | Full snapshot lifecycle via gRPC | Save → List → Load → Delete |
| `TestGRPC_ConcurrentStart` | Two start calls simultaneously | One succeeds, one fails |
| `TestGRPC_StreamInterrupted` | Client disconnects mid-stream | Engine cleans up gracefully |

---

## 4. End-to-End (E2E) Tests

E2E tests exercise full user workflows from CLI invocation through engine to Docker, verifying the complete stack.

### 4.1 Core Workflow: init → validate → start → status → stop

```
Test: Core Lifecycle
Steps:
  1. devbox init my-e2e-test          → devbox.yml created
  2. devbox validate                   → Config valid
  3. Start engine daemon               → Engine running
  4. devbox start                      → All services healthy
  5. devbox status                     → All services "running"
  6. devbox ps                         → Containers listed
  7. devbox stop                       → All services stopped
  8. devbox status                     → All services "stopped"
  9. Stop engine daemon                → Engine stopped
```

### 4.2 Build + Start Workflow

```
Test: Dockerfile Build
Steps:
  1. Use test-build-project/           → Has Dockerfile + devbox.yml
  2. devbox build                      → Image built successfully
  3. devbox start                      → Service starts from built image
  4. curl localhost:8090               → Response contains "DevBoxOS Build Test"
  5. devbox stop                       → Cleanup
```

### 4.3 Secrets Workflow

```
Test: Secrets Lifecycle
Steps:
  1. devbox secrets list               → Empty (or initial)
  2. devbox secrets add API_KEY=abc123 → Added
  3. devbox secrets get API_KEY        → Shows masked value
  4. devbox secrets list               → API_KEY listed
  5. devbox start                      → Service has API_KEY env var
  6. devbox exec web -- env            → API_KEY visible in container
  7. devbox secrets rotate API_KEY     → New value generated
  8. devbox exec web -- env            → New value in container
  9. devbox secrets delete API_KEY     → Removed
 10. devbox secrets list               → API_KEY gone
```

### 4.4 Snapshot Workflow

```
Test: Snapshot Save/Restore
Steps:
  1. devbox start                      → Environment running
  2. Write test data to service        → e.g., curl POST /data
  3. devbox snapshot save --name test1 → Saved
  4. devbox snapshot list              → test1 in list
  5. Modify service data               → Data changed
  6. devbox stop                       → Stop environment
  7. devbox snapshot load --name test1 → Restored from snapshot
  8. devbox start                      → Environment restarted
  9. Verify test data restored         → Original data present
 10. devbox snapshot delete --name test1 → Deleted
 11. devbox snapshot list              → Empty
```

### 4.5 Compose Import Workflow

```
Test: Docker Compose Import
Steps:
  1. Create docker-compose.yml with nginx + redis
  2. devbox compose-import             → devbox.yml created
  3. devbox validate                   → Generated config is valid
  4. devbox start                      → Both services start
  5. curl localhost:8080               → nginx responding
  6. devbox exec redis redis-cli ping  → redis responding
  7. devbox stop                       → Cleanup
```

### 4.6 Error Recovery Workflow

```
Test: Engine Crash Recovery
Steps:
  1. devbox start                      → Environment running
  2. Kill engine daemon (SIGKILL)      → Engine process killed
  3. Restart engine daemon             → Engine starts
  4. devbox status                     → Orphaned containers detected
  5. devbox reset                      → Orphaned resources cleaned
  6. devbox start                      → Clean start works
```

### 4.7 Full Multi-Service Workflow

```
Test: Multi-Service with test-project/
Services: web (nginx), redis (7-alpine)
Steps:
  1. devbox start                      → Both services start
  2. devbox status                     → web: running, redis: running
  3. curl http://localhost:8080        → nginx default page
  4. devbox exec redis redis-cli ping  → PONG
  5. devbox logs web --tail 5          → Last 5 nginx log lines
  6. devbox logs redis --tail 5        → Last 5 redis log lines
  7. devbox snapshot save --name e2e-test → Snapshot saved
  8. devbox snapshot list              → e2e-test in list
  9. devbox stop                       → Stop all
  10. devbox snapshot load --name e2e-test  → Restored
  11. devbox start                     → Restarted
  12. Verify curl still works          → Data preserved
  13. devbox reset                     → Full destroy
  14. devbox snapshot delete --name e2e-test → Cleanup
```

### 4.8 Plugin Hooks Workflow

```
Test: Plugin Lifecycle Hooks (test-build-project/)
Services: web (nginx:alpine with Dockerfile), plugins: notify-start, notify-stop
Steps:
  1. devbox build web                  → Image built
  2. devbox start                      → Plugins execute post-start
     → stdout shows: "Environment started for test-build-app"
  3. devbox status                     → web running
  4. devbox stop                       → Plugins execute post-stop
     → stdout shows: "Environment stopped for test-build-app"
```

### 4.9 Cross-Platform IPC Workflow

```
Test: Platform-Specific IPC
Windows:
  1. Start engine daemon               → Listens on 127.0.0.1:51000 (TCP)
  2. devbox start                      → CLI connects via TCP
  3. devbox status                     → Works over TCP
  4. Stop engine                       → Cleanup

macOS/Linux:
  1. Start engine daemon               → Listens on ~/.devbox/engine.sock (Unix)
  2. devbox start                      → CLI connects via Unix socket
  3. devbox status                     → Works over Unix socket
  4. Stop engine                       → Cleanup
```

### 4.10 Config Persistence Workflow

```
Test: Config Get/Set Round Trip
Steps:
  1. devbox config get engine.port     → Returns "51000" (default)
  2. devbox config set engine.port 51001 → Set
  3. devbox config get engine.port     → Returns "51001"
  4. Verify ~/.config/devboxos/config.json → File contains port: 51001
  5. devbox config set engine.port 51000 → Reset to default
```

### 4.11 `devbox exec` Workflow

```
Test: Execute Command in Service
Steps:
  1. devbox start                      → Environment running
  2. devbox exec web -- nginx -v       → nginx version output
  3. devbox exec web -- ls /usr/share/nginx/html → File listing
  4. devbox exec nonexistent -- ls     → Error: service not found
  5. devbox exec web -- nonexistent-cmd → Error: command not found
  6. devbox stop                       → Cleanup
```

### 4.12 `devbox doctor` Workflow

```
Test: Diagnostics
Steps:
  1. devbox doctor                     → All checks pass (Docker running, config exists)
  2. Stop Docker daemon                → Docker unavailable
  3. devbox doctor                     → Reports Docker as critical issue with fix hint
  4. Start Docker daemon               → Docker available
  5. devbox doctor                     → All healthy again
```

---

## 5. Smoke Tests

Smoke tests are lightweight sanity checks run on every CI build to quickly verify the application works.

### 5.1 CLI Smoke Tests (no engine needed)

```bash
# Version
devbox version

# Init + Validate (temp dir)
cd $(mktemp -d)
devbox init smoke-test
devbox validate
devbox validate --help

# Config
devbox config get engine.port
devbox config set engine.test.key test-value
devbox config get engine.test.key
devbox config set engine.test.key ""

# Completion
devbox completion bash > /dev/null
devbox completion zsh > /dev/null
devbox completion fish > /dev/null

# Compose import (with known docker-compose.yml)
cd ..
devbox compose-import --help

# Secrets (local, no engine)
devbox secrets add SMOKE_TEST_TOKEN smoke-value
devbox secrets get SMOKE_TEST_TOKEN
devbox secrets list
devbox secrets delete SMOKE_TEST_TOKEN
```

### 5.2 Engine Smoke Tests

```bash
# Start engine
devbox-engine &
sleep 2

# Verify engine is running
devbox status    # Should show "engine running" or similar

# Start test-project
cd test-project
devbox start     # Both services healthy

# Quick verification
curl -f http://localhost:8080
devbox exec web -- nginx -v
devbox exec redis -- redis-cli ping

# Snapshot quick test
devbox snapshot save --name smoke-test
devbox snapshot list
devbox snapshot delete --name smoke-test

# Stop
devbox stop

# Kill engine
kill %1
```

### 5.3 Quick Smoke Script

```bash
#!/usr/bin/env bash
# devbox-smoke.sh — Quick smoke test for CI
set -euo pipefail

echo "=== DevBoxOS Smoke Test ==="

echo "1. CLI Basics"
devbox version
devbox doctor

echo "2. Init & Validate"
cd "$(mktemp -d)"
devbox init smoke-test
devbox validate
echo "   ✓ Init + Validate"

echo "3. Engine Start"
devbox-engine &
ENGINE_PID=$!
sleep 3
devbox status
echo "   ✓ Engine running"

echo "4. Project Start"
cd ../test-project
devbox start
sleep 5
devbox status | grep -q "web.*running" || exit 1
echo "   ✓ Services running"

echo "5. HTTP Check"
curl -f http://localhost:8080 > /dev/null
echo "   ✓ HTTP accessible"

echo "6. Logs"
devbox logs web --tail 3 > /dev/null
echo "   ✓ Logs available"

echo "7. Snapshot"
devbox snapshot save --name smoke-ci
devbox snapshot list | grep -q smoke-ci || exit 1
devbox snapshot delete --name smoke-ci
echo "   ✓ Snapshots work"

echo "8. Stop & Cleanup"
devbox stop
kill $ENGINE_PID 2>/dev/null || true
echo "   ✓ Cleanup done"

echo "=== All smoke tests passed ==="
```

---

## 6. Compatibility Tests

### 6.1 OS Compatibility Matrix

| Feature | Windows (Docker Desktop) | macOS Intel | macOS Apple Silicon | Linux (Docker Engine) |
|---------|-------------------------|-------------|---------------------|----------------------|
| CLI commands | ✅ | ✅ | ✅ | ✅ |
| Engine daemon | ✅ (TCP) | ✅ (Unix sock) | ✅ (Unix sock) | ✅ (Unix sock) |
| `devbox start` | ✅ | ✅ | ✅ | ✅ |
| `devbox stop` | ✅ | ✅ | ✅ | ✅ |
| Port mapping | ✅ | ✅ | ✅ | ✅ |
| Volume mounting | ✅ | ✅ | ✅ | ✅ |
| Secrets | ✅ | ✅ | ✅ | ✅ |
| Snapshots | ✅ | ✅ | ✅ | ✅ |
| Build | ✅ | ✅ | ✅ | ✅ |
| Logs | ✅ | ✅ | ✅ | ✅ |
| Plugins | ✅ | ✅ | ✅ | ✅ |
| Diagnostics | ✅ | ✅ | ✅ | ✅ |
| Config persistence | ✅ (AppData) | ✅ (~/.config) | ✅ (~/.config) | ✅ (~/.config) |

### 6.2 Docker Version Compatibility

| Docker Version | Windows | macOS | Linux |
|----------------|---------|-------|-------|
| 24.x | ✅ | ✅ | ✅ |
| 25.x | ✅ | ✅ | ✅ |
| 26.x | ✅ | ✅ | ✅ |
| 27.x (latest) | ✅ | ✅ | ✅ |

### 6.3 Go Version Compatibility

| Go Version | Build | Tests |
|------------|-------|-------|
| 1.22.x | ✅ | ✅ |
| 1.23.x | ✅ | ✅ |
| 1.24.x | ✅ | ✅ |

---

## 7. Performance Benchmarks

### 7.1 Startup Time Benchmarks

| Benchmark | Target | Threshold |
|-----------|--------|-----------|
| Config parsing (50 services) | < 50ms | < 200ms |
| Config validation (50 services) | < 100ms | < 500ms |
| Dependency graph resolution (50 services) | < 10ms | < 50ms |
| Engine daemon startup | < 2s | < 5s |
| Single service start (nginx, image cached) | < 3s | < 10s |
| Multi-service start (5 services, all cached) | < 15s | < 30s |
| Snapshot save (small project, no DB) | < 5s | < 15s |
| Snapshot load (small project, no DB) | < 5s | < 15s |
| `devbox exec` (simple command) | < 1s | < 3s |
| Log stream startup | < 500ms | < 2s |
| CLI `--help` output | < 100ms | < 500ms |

### 7.2 Resource Benchmarks

| Benchmark | Target | Threshold |
|-----------|--------|-----------|
| Engine memory (idle, no projects) | < 15MB | < 50MB |
| Engine memory (1 project, 3 services) | < 30MB | < 100MB |
| Engine CPU (idle) | < 0.5% | < 2% |
| CLI memory (baseline) | < 10MB | < 30MB |
| Snapshot storage overhead | < 5% of artifact size | < 15% |
| Config file read/write latency | < 1ms | < 10ms |

### 7.3 Scale Benchmarks

| Benchmark | Target | Threshold |
|-----------|--------|-----------|
| 100 services in devbox.yml | Parse + validate < 1s | < 5s |
| 20 services starting concurrently | All healthy < 60s | < 120s |
| 1000 secrets in store | Get operation < 10ms | < 50ms |
| 100 log entries/second per service | No backpressure | < 10% dropped |

---

## 8. Security Tests

### 8.1 Secrets Security

| Test Case | Description | Expected |
|-----------|-------------|----------|
| Secrets never written to disk in plaintext | Inspect filesystem during startup | No plaintext files |
| Secrets masked in output | `devbox secrets list/get` | Values shown as `****` |
| Secrets masked in logs | Check log output | No secret values leaked |
| Secrets masked in error messages | Trigger error with secrets | No values in error text |
| Age key file permissions | Check file mode | 0600 (owner-only) |
| Snapshot secrets encrypted | Inspect snapshot archive | Secrets encrypted |

### 8.2 Network Security

| Test Case | Description | Expected |
|-----------|-------------|----------|
| Default-deny egress | Service tries to reach internet | Blocked unless explicitly allowed |
| mTLS enforced by default | Inter-service traffic | TLS handshake required |
| Port conflict isolation | Two projects on same port | No conflict, separate networks |
| Cross-project isolation | Project A cannot reach Project B | Network-level isolation |

### 8.3 Container Security

| Test Case | Description | Expected |
|-----------|-------------|----------|
| Rootless containers | Inspect container user | Non-root |
| Capabilities dropped | Inspect container capabilities | Minimal set |
| Read-only root filesystem | Try to write to / | Denied |

---

## 9. Manual Smoke Test Sequence

This is the complete sequence for manually verifying a fresh build on a target machine. Run these commands in order.

### Phase 0: Prerequisites

```bash
# 1. Verify Docker
docker ps

# 2. Run unit tests
cd shared && go test -v -race ./... && cd ..
cd engine && go test -v -race ./... && cd ..
cd cli && go test -v -race ./... && cd ..

# 3. Build fresh binaries
make clean && make build

# 4. Verify binaries exist
ls -la dist/
```

### Phase 1: Basic Commands (no engine needed)

```bash
# 5. Version
./dist/devbox version

# 6. Doctor (diagnostics)
./dist/devbox doctor

# 7. Init — new project
mkdir -p /tmp/devbox-smoke
cd /tmp/devbox-smoke
../path/to/dist/devbox init my-smoke-app
cat devbox.yml

# 8. Validate — both test projects
cd /path/to/test-project
/path/to/dist/devbox validate
cd /path/to/test-build-project
/path/to/dist/devbox validate

# 9. Config get/set
/path/to/dist/devbox config get engine.port
/path/to/dist/devbox config set engine.port 51001
/path/to/dist/devbox config get engine.port
/path/to/dist/devbox config set engine.port 51000

# 10. Completion generation
/path/to/dist/devbox completion bash > /dev/null
/path/to/dist/devbox completion zsh > /dev/null
/path/to/dist/devbox completion fish > /dev/null

# 11. Compose import (if docker-compose.yml exists)
cd /tmp/devbox-smoke
cat > docker-compose.yml << 'EOF'
services:
  web:
    image: nginx:alpine
    ports:
      - "8081:80"
  redis:
    image: redis:7-alpine
EOF
/path/to/dist/devbox compose-import
cat devbox.yml

# 12. Secrets (local, no engine)
cd /path/to/test-project
/path/to/dist/devbox secrets list
/path/to/dist/devbox secrets add MY_SECRET test123
/path/to/dist/devbox secrets get MY_SECRET
/path/to/dist/devbox secrets list
/path/to/dist/devbox secrets delete MY_SECRET
```

### Phase 2: Start & Stop Engine + Services

```bash
# 13. Start engine
cd /path/to/test-project
/path/to/dist/devbox-engine &
ENGINE_PID=$!
sleep 2

# 14. Start services
/path/to/dist/devbox start

# 15. Status
/path/to/dist/devbox status

# 16. PS
/path/to/dist/devbox ps

# 17. Logs
/path/to/dist/devbox logs web --tail 5
/path/to/dist/devbox logs redis --tail 5

# 18. Exec into services
/path/to/dist/devbox exec web -- nginx -v
/path/to/dist/devbox exec redis -- redis-cli ping

# 19. Verify HTTP access
curl http://localhost:8080
```

### Phase 3: Build + Start (test-build-project)

```bash
# 20. Stop current project
cd /path/to/test-project
/path/to/dist/devbox stop

# 21. Build and start build project
cd /path/to/test-build-project
/path/to/dist/devbox build
/path/to/dist/devbox start

# 22. Verify
/path/to/dist/devbox ps
curl http://localhost:8090
/path/to/dist/devbox logs web --tail 5
```

### Phase 4: Full Secrets Workflow

```bash
# 23. Secrets CRUD
cd /path/to/test-project
/path/to/dist/devbox start
/path/to/dist/devbox secrets add CLI_TEST_TOKEN my-test-token
/path/to/dist/devbox secrets get CLI_TEST_TOKEN
/path/to/dist/devbox secrets list
/path/to/dist/devbox secrets delete CLI_TEST_TOKEN
```

### Phase 5: Snapshot Lifecycle

```bash
# 24. Save snapshot
cd /path/to/test-project
/path/to/dist/devbox snapshot save --name manual-test

# 25. List snapshots
/path/to/dist/devbox snapshot list

# 26. Load snapshot
/path/to/dist/devbox stop
/path/to/dist/devbox snapshot load --name manual-test
/path/to/dist/devbox start

# 27. Delete snapshot
/path/to/dist/devbox snapshot delete --name manual-test
```

### Phase 6: Stop, Reset, Destroy, Prune

```bash
# 28. Stop
/path/to/dist/devbox stop
/path/to/dist/devbox status

# 29. Reset
/path/to/dist/devbox reset

# 30. Prune
/path/to/dist/devbox prune

# 31. Destroy (full cleanup)
/path/to/dist/devbox destroy
```

### Phase 7: Kill Engine

```bash
# 32. Kill engine
kill $ENGINE_PID 2>/dev/null
wait $ENGINE_PID 2>/dev/null

# 33. Cleanup temp files
rm -rf /tmp/devbox-smoke
```

---

## 10. CI Pipeline Integration

### 10.1 CI Jobs (`.github/workflows/ci.yml`)

```
┌──────────────────────────────────────────────────────────┐
│                   CI Pipeline                             │
├────────────┬──────────────┬──────────────┬───────────────┤
│  Lint      │  Unit Tests  │  Integration  │  Build Check  │
│  (ubuntu)  │  (3 OS)      │  (ubuntu)     │  (ubuntu)     │
├────────────┼──────────────┼──────────────┼───────────────┤
│ golangci-  │ go test -v   │ go test -v   │ go build ./…  │
│ lint       │ -race ./…    │ -tags=inte-  │                │
│            │              │ gration ./…  │ go mod tidy    │
│            │              │              │                │
├────────────┴──────────────┴──────────────┴───────────────┤
│                       Post-CI                             │
├──────────────────────────────────────────────────────────┤
│  Coverage Report (codecov.io)  │  Artifact Upload         │
└──────────────────────────────────────────────────────────┘
```

### 10.2 CI Flow (Recommended Additions)

```
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - uses: golangci/golangci-lint-action@v6
        with: { timeout: 5m }
      - run: go vet ./...

  unit:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go test -v -race -coverprofile=coverage.out ./shared/... ./engine/... ./cli/...

  integration:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
        options: --privileged
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go test -v -tags=integration -count=1 ./shared/runtime/docker/...
      - run: go test -v -tags=integration -count=1 ./shared/snapshot/...

  smoke:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: make build
      - run: bash scripts/smoke-test.sh  # automated smoke test script

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go build ./cli/...
      - run: go build ./engine/...
      - run: go mod tidy && git diff --exit-code go.mod go.sum

  bench:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go test -bench=. -benchmem -count=3 ./... > bench.out
      - uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: bench.out
          alert-threshold: '200%'
          comment-on-alert: true

  e2e:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v')
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: make build
      - run: bash scripts/e2e-test.sh  # automated E2E test script

  compatibility:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-13, macos-14, windows-latest]
        docker-version: [24, 25, 26]
    runs-on: ${{ matrix.os }}
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: make build
      - run: bash scripts/compatibility-test.sh
```

---

## 11. Test Infrastructure

### 11.1 Test Helpers Needed

| Helper | Purpose | Package |
|--------|---------|---------|
| `NewTestRuntime()` | Creates a runtime connected to test Docker | `shared/runtime/docker/testing.go` |
| `NewTestEngine()` | Starts an engine instance for testing | `engine/testing.go` |
| `NewTestProject()` | Creates a temp project with devbox.yml | `shared/config/testing.go` |
| `AssertContainerRunning()` | Helper to verify container state | `testhelpers/assert.go` |
| `AssertLogContains()` | Helper to verify log output | `testhelpers/assert.go` |
| `WaitForPort()` | Wait for a TCP port to be open | `testhelpers/wait.go` |
| `TempDir()` | Create temp dir with cleanup | `testhelpers/tempdir.go` |
| `SkipIfNoDocker()` | Skip test if Docker unavailable | `testhelpers/skip.go` |
| `SkipIfRoot()` | Skip test if running as root | `testhelpers/skip.go` |
| `SkipIfWindows()` | Skip test on Windows (Unix-only features) | `testhelpers/skip.go` |

### 11.2 Test Fixtures

| Fixture | Location | Purpose |
|---------|----------|---------|
| `test-project/devbox.yml` | `test-project/` | Multi-service (nginx + redis) with generated secrets |
| `test-build-project/devbox.yml` | `test-build-project/` | Build-based service with Dockerfile + plugins |
| `test-build-project/Dockerfile` | `test-build-project/` | Simple nginx Dockerfile |
| `test-build-project/nginx.conf` | `test-build-project/` | Custom nginx config |
| `test-fixtures/valid-config.yml` | `test-fixtures/` | Valid multi-service config for parser tests |
| `test-fixtures/invalid-config.yml` | `test-fixtures/` | Invalid config for validation tests |
| `test-fixtures/minimal-config.yml` | `test-fixtures/` | Minimal valid config |
| `test-fixtures/complex-config.yml` | `test-fixtures/` | 20+ services for scale tests |
| `test-fixtures/docker-compose.yml` | `test-fixtures/` | Standard Compose file for import tests |
| `test-fixtures/snapshot.tar.gz` | `test-fixtures/` | Pre-built snapshot for load tests |
| `test-fixtures/age-key.txt` | `test-fixtures/` | Test age key for deterministic encryption tests |

### 11.3 Mock Interfaces

| Interface | Mock | Purpose |
|-----------|------|---------|
| `runtime.Runtime` | `MockRuntime` | Test engine without Docker |
| `secrets.Provider` | `MockProvider` | Test resolver with controlled values |
| `logging.Store` | `MockLogStore` | Test log-dependent features without real logs |
| `monitor.Monitor` | `MockMonitor` | Test monitor-dependent features with fake stats |

---

## 12. Coverage Targets & Tracking

### 12.1 Per-Package Coverage Targets

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| `shared/config` | ~20% | 90% | High |
| `shared/platform` | ~85% | 95% | Low (already good) |
| `shared/secrets` | ~30% | 90% | High |
| `shared/runtime` | 0% | 75% | High (integration-heavy) |
| `shared/snapshot` | 0% | 70% | High |
| `shared/logging` | 0% | 80% | Medium |
| `shared/diagnostics` | 0% | 75% | Medium |
| `shared/plugins` | 0% | 75% | Medium |
| `engine/internal/orchestrator` | ~5% | 80% | High |
| `engine/internal/networking` | 0% | 70% | Medium |
| `engine/internal/state` | 0% | 80% | Medium |
| `engine/internal/monitor` | 0% | 60% | Low |
| `cli/cmd` | 0% | 70% | High |
| `cli/internal/client` | 0% | 60% | Medium |

### 12.2 Overall Targets

| Metric | Target |
|--------|--------|
| Overall line coverage | 70%+ |
| Critical path coverage | 90%+ |
| Test files with assertions | 100% of packages |
| Integration tests covering critical paths | 100% |
| E2E tests covering core workflows | 100% |
| Race-condition-free (-race passes) | 100% |
| Lint warnings | 0 (blocking) |
| Benchmark regression alerts | All PRs |

---

## 13. Test Fixtures

### 13.1 Standard Test Configs

**`test-fixtures/valid-config.yml`**:
```yaml
name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
    port: "8080:80"
  redis:
    image: redis:7-alpine
    port: "6379:6379"
```

**`test-fixtures/invalid-config.yml`**:
```yaml
# Missing required field: "name"
version: "1.0"
services:
  web:
    # Missing image, runtime, or build
    port: "abc:80"  # invalid port format
```

**`test-fixtures/complex-config.yml`**:
```yaml
name: complex-test
version: "2.0"
services:
  web:     { image: nginx:alpine,    port: "8080:80",    depends_on: [api] }
  api:     { image: node:20-alpine,  port: "3000:3000",  depends_on: [db, redis] }
  db:      { image: postgres:16,     port: "5432:5432" }
  redis:   { image: redis:7-alpine,  port: "6379:6379" }
  worker:  { image: node:20-alpine,  depends_on: [db, redis] }
  cache:   { image: memcached:1,     port: "11211:11211" }
  mongo:   { image: mongo:7,         port: "27017:27017", depends_on: [] }
  # ... up to 20 services
```

---

## 14. Edge Cases & Error Paths

### 14.1 Network & Connectivity

| Scenario | Expected Behavior |
|----------|-------------------|
| Docker daemon not running | Clear error: "Docker is not running. Start Docker Desktop or Docker Engine." |
| Docker socket permission denied | Error with fix hint: "Add user to docker group or check socket permissions." |
| Port already in use | Error: "Port 8080 is in use by PID 1234 (nginx). Use different port or stop the process." |
| Disk full during snapshot | Error: "Insufficient disk space for snapshot. Need 500MB, have 200MB available." |
| Network timeout during pull | Retry with backoff, then error: "Failed to pull image after 3 retries." |
| Engine address already in use | Engine fails to start with clear port conflict message |

### 14.2 Config & Validation

| Scenario | Expected Behavior |
|----------|-------------------|
| devbox.yml missing | Error: "devbox.yml not found. Run 'devbox init' to create one." |
| devbox.yml has syntax error | Error with line number and column: "YAML parse error at line 12, column 5." |
| Service name contains spaces | Validation error: "Service names must be lowercase alphanumeric." |
| Circular dependency detected | Error: "Circular dependency detected: web → api → db → web." |
| Unknown runtime specified | Validation warning, falls back to Docker image |
| Invalid port format | Validation error with expected format: "Use 'host:container' format (e.g., '8080:80')." |

### 14.3 Runtime & Containers

| Scenario | Expected Behavior |
|----------|-------------------|
| Image not found | Pull fails with 404, lists similar image names if available |
| Container exits immediately | Status shows "exited (1)" with exit code and last log lines |
| Container OOM killed (exit 137) | Status shows "OOMKilled" with suggestion to increase memory limit |
| Container health check fails | Service marked unhealthy, auto-restart up to retry limit |
| Concurrent `devbox start` calls | Second call blocked by lock file: "Another operation is in progress." |
| Engine killed during start | On next start, orphaned containers detected and cleaned up |

### 14.4 Secrets & Security

| Scenario | Expected Behavior |
|----------|-------------------|
| Age key file missing | Generated automatically on first use |
| Age key file corrupted | Error: "Age key file corrupted. Restore from backup or delete to regenerate." |
| Secret file unreadable | Error with file path and suggested fix |
| Secret value contains special chars | Properly escaped in shell execution |
| Attempt to add duplicate secret | Overwrites with confirmation prompt |

### 14.5 Snapshot Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Snapshot name with special chars | Rejected: "Use alphanumeric characters and hyphens only." |
| Duplicate snapshot name | Overwritten with warning (unless --force) |
| Snapshot load with no matching name | Error: "Snapshot 'name' not found. Available: [list]." |
| Corrupted snapshot archive | Error: "Snapshot integrity check failed (expected hash X, got Y)." |
| Load snapshot from different project | Warning: "Snapshot was saved from project 'other', not current project 'current'." |
| Very large snapshot (>10GB) | Progress reporting during save/load |

### 14.6 Cross-Platform Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Windows paths in devbox.yml | Automatically normalized to platform format |
| macOS with Docker Desktop vs Colima | Auto-detect Docker socket location |
| Linux with rootless Docker | Auto-detect user-scoped socket path |
| WSL2 with Docker Desktop | Linux binary inside WSL connects to Windows Docker socket |
| Non-English locale | All error messages in English, no locale-dependent parsing |

---

## Appendix A: Testing Commands Quick Reference

```bash
# Run all unit tests with race detection
go test -race -count=1 ./shared/... ./engine/... ./cli/...

# Run specific package tests
go test -v -race ./shared/config/...
go test -v -race ./shared/secrets/...
go test -v -race ./shared/platform/...
go test -v -race ./engine/internal/orchestrator/...

# Run integration tests (requires Docker)
go test -v -tags=integration -count=1 ./shared/runtime/docker/...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out

# Lint
golangci-lint run ./...

# Build test
go build ./shared/...
go build ./engine/...
go build ./cli/...
```

## Appendix B: Test File Checklist

| File | Package | Status |
|------|---------|--------|
| `shared/config/parser_test.go` | config | ✅ Existing |
| `shared/platform/platform_test.go` | platform | ✅ Existing |
| `shared/secrets/secrets_test.go` | secrets | ✅ Existing |
| `engine/internal/orchestrator/graph_test.go` | orchestrator | ✅ Existing |
| `shared/config/validator_test.go` | config | ❌ Missing |
| `shared/config/autodetect_test.go` | config | ❌ Missing |
| `shared/runtime/docker/client_test.go` | docker | ❌ Missing |
| `shared/snapshot/manager_test.go` | snapshot | ❌ Missing |
| `shared/snapshot/manifest_test.go` | snapshot | ❌ Missing |
| `shared/logging/store_test.go` | logging | ❌ Missing |
| `shared/diagnostics/checker_test.go` | diagnostics | ❌ Missing |
| `shared/plugins/manager_test.go` | plugins | ❌ Missing |
| `engine/internal/orchestrator/orchestrator_test.go` | orchestrator | ❌ Missing |
| `engine/internal/orchestrator/lifecycle_test.go` | orchestrator | ❌ Missing |
| `engine/internal/orchestrator/recovery_test.go` | orchestrator | ❌ Missing |
| `engine/internal/networking/network_test.go` | networking | ❌ Missing |
| `engine/internal/networking/mtls_test.go` | networking | ❌ Missing |
| `engine/internal/networking/dns_test.go` | networking | ❌ Missing |
| `engine/internal/state/sqlite_test.go` | state | ❌ Missing |
| `engine/internal/state/lock_test.go` | state | ❌ Missing |
| `engine/internal/monitor/monitor_test.go` | monitor | ❌ Missing |
| `cli/cmd/version_test.go` | cmd | ❌ Missing |
| `cli/cmd/init_test.go` | cmd | ❌ Missing |
| `cli/cmd/validate_test.go` | cmd | ❌ Missing |
| `cli/cmd/config_test.go` | cmd | ❌ Missing |
| `cli/cmd/start_test.go` | cmd | ❌ Missing |
| `cli/cmd/stop_test.go` | cmd | ❌ Missing |
| `cli/cmd/status_test.go` | cmd | ❌ Missing |
| `cli/cmd/logs_test.go` | cmd | ❌ Missing |
| `cli/cmd/doctor_test.go` | cmd | ❌ Missing |
| `cli/cmd/secrets_test.go` | cmd | ❌ Missing |
| `cli/cmd/snapshot_test.go` | cmd | ❌ Missing |
| `cli/cmd/completion_test.go` | cmd | ❌ Missing |
| `cli/cmd/compose_import_test.go` | cmd | ❌ Missing |
| `cli/cmd/build_test.go` | cmd | ❌ Missing |
| `cli/cmd/exec_test.go` | cmd | ❌ Missing |
| `cli/cmd/destroy_test.go` | cmd | ❌ Missing |
| `cli/cmd/prune_test.go` | cmd | ❌ Missing |
| `cli/cmd/ps_test.go` | cmd | ❌ Missing |
| `cli/internal/client/grpc_client_test.go` | client | ❌ Missing |

---

*Document Version: 1.0*
*Last Updated: 2026*
