# Clients — Native Language Libraries

This directory contains per-language libraries that retrieve secrets directly from
the OS keychain, with **no dependency on the `keysync` binary**. Each library
calls the native OS secret storage tooling directly, using the same service-naming
convention that `keysync` uses internally.

## Library overview

| Library | Location | macOS | Linux | Windows | Status |
|---------|----------|-------|-------|---------|--------|
| **Go** | `clients/go/` | `security` CLI | `secret-tool` CLI | `wincred` Go lib | Ready |
| **Python** | `clients/python/` | `security` CLI | `secret-tool` CLI | `ctypes` Win32 API | Ready |
| **TypeScript** | `clients/node/` | `security` CLI | `secret-tool` CLI | PowerShell + Win32 | Ready (macOS/Linux) |
| **Swift** | `clients/swift/` | `Security.framework` | `secret-tool` CLI | Not planned | Ready (macOS/Linux) |
| **Java** | `clients/java/` | `security` CLI | `secret-tool` CLI | JNA → Win32 API | Available (Windows: not fully tested) |
| **C# (.NET)** | `clients/csharp/` | `security` CLI | `secret-tool` CLI | P/Invoke → Win32 API | Available (Windows: not fully tested) |
| **Rust** | `clients/rust/` | `security` CLI | `secret-tool` CLI | `windows-sys` crate | Available (Windows: not fully tested) |
| **C++** | `clients/cpp/` | `security` CLI | `secret-tool` CLI | Win32 API (wincred.h) | Available (Windows: not fully tested) |
| **Ruby** | `clients/ruby/` | `security` CLI | `secret-tool` CLI | PowerShell + inline C# | Available (Windows: not fully tested) |
| **Dart** | `clients/dart/` | `security` CLI | `secret-tool` CLI | PowerShell + Credential Manager | Ready (Flutter desktop only) |

## Design

### How secrets are stored

Keysync stores secrets in the OS keychain with a consistent naming scheme:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name (e.g. `DATABASE_URL`) |
| Environment | `keysync/project/<name>/env/<env>` | key name (e.g. `DATABASE_URL`) |

The CLI stores project-wide secrets when you use `-p` without `--env`. Pass an `environment` argument to client libraries only when you use env-scoped keys in the keychain.

When looking up a secret, resolution order is: environment (if requested) → project → global.

### Per-platform access

Each library uses the platform's native tooling rather than wrapping the `keysync` binary:

| Platform | Tool / API | Get Command |
|----------|-----------|-------------|
| macOS | `security` CLI (built-in) | `security find-generic-password -s keysync/global -a KEY -w` |
| Linux | `secret-tool` CLI (libsecret) | `secret-tool lookup service keysync/global account KEY` |
| Windows | `wincred` / `ctypes` | Native Win32 API via Go library or Python ctypes |
| Swift (macOS) | `Security.framework` | `SecItemCopyMatching` — native API, no subprocess |

### Why direct OS access instead of shelling out to `keysync get`

- **No binary dependency** — the keysync CLI is only needed for write operations
  (`set`, `sync`, `rotate`, etc.). Read operations work standalone.
- **No subprocess overhead** — secrets are retrieved in-process, critical for apps
  that load many secrets at startup.
- **Tighter security boundary** — secret values never cross a pipe or process
  boundary at read time.
- **Swift on macOS can use native Security framework** — the cleanest integration
  of all, using `SecItemCopyMatching` directly without shelling out.

### Common API

Every library follows the same pattern:

```python
# Get a secret — checks project scope first, falls back to global
value = get_secret("DATABASE_URL", project="my-api")

# List all secrets with optional filtering
entries = list_secrets(scope="project", project="my-api")
```

Error handling uses language-appropriate patterns:

| Language | Error type |
|----------|-----------|
| Go | `ErrNotFound` sentinel error |
| Python | `SecretNotFoundError` exception |
| TypeScript | `KeySyncError` with `code` property |
| Swift | `KeySyncError` enum case |

### Service name helpers

Every library implements these two functions:

```
serviceName(scope, project) → "keysync/global" | "keysync/project/<name>"
parseServiceName(svc)       → (scope, project)
```

## Quick start

```bash
# Go
go get github.com/dipockdas/keysync/clients/go

# Python
pip install keysync

# TypeScript
npm install @dipockdas/keysync

# Swift — add to Package.swift
// .package(url: "https://github.com/dipockdas/keysync.git", branch: "main"),

# Java — add to pom.xml
// <dependency>
//   <groupId>io.keysync</groupId>
//   <artifactId>keysync</artifactId>
//   <version>1.0.0</version>
// </dependency>

# C# (.NET) — add to .csproj
// dotnet add package KeySync

# Rust — add to Cargo.toml
// keysync = "1.0.0"

# C++ — add via CMake FetchContent or subdirectory
// add_subdirectory(path/to/keysync/clients/cpp)
// target_link_libraries(your_app PRIVATE keysync)

# Ruby — add to Gemfile
// gem "keysync"
```

## Per-library documentation

Each library has its own README.md with full usage docs, CLAUDE.md (Claude Code
instructions), and AGENTS.md (AI agent instructions):

- [Go client](go/) — `clients/go/`
- [Python client](python/) — `clients/python/`
- [TypeScript client](node/) — `clients/node/`
- [Swift client](swift/) — `clients/swift/`
- [Java client](java/) — `clients/java/`
- [C# (.NET) client](csharp/) — `clients/csharp/`
- [Rust client](rust/) — `clients/rust/`
- [C++ client](cpp/) — `clients/cpp/`
- [Ruby client](ruby/) — `clients/ruby/`
- [Dart client](dart/) — `clients/dart/` (Flutter desktop)
