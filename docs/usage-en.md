# Fleet User Guide

Fleet is a multi-repo workspace management tool written in Go. It declaratively manages multiple Git repositories via manifest XML files, supporting batch clone, sync, status, push, and cross-repo command execution under the GitHub Fork workflow.

## Installation

```bash
# Install from source
go install github.com/mingyuans/fleet-cli@latest

# Or build locally
git clone https://github.com/mingyuans/fleet-cli.git
cd fleet-cli
make build
# Binary is generated at bin/fleet
```

## Quick Start

### 1. Prepare a Workspace

A workspace is a directory containing manifest files and multiple Git repositories:

```
workspace/
├── default.xml              # Shared team config (committed to Git)
├── local_manifest.xml       # Personal local overrides (gitignored)
└── services/
    ├── user-service/
    ├── order-service/
    └── ...
```

### 2. Configure Manifests

**default.xml** — Shared team configuration:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github"
          fetch="git@github.com:my-org/"
          review="https://github.com/my-org/" />

  <default remote="github" revision="master" sync-j="4" />

  <project name="user-service"
           path="services/user-service"
           groups="core,backend" />

  <project name="order-service"
           path="services/order-service"
           groups="commerce" />
</manifest>
```

**local_manifest.xml** — Personal fork configuration:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:your-username/" />
  <default push="fork" />
</manifest>
```

### 3. Initialize Repositories

```bash
fleet init
```

## Commands

### Global Options

```
fleet <command> [options]

Options:
  -g, --group <group>   Filter projects by group
  -h, --help            Show help
```

All subcommands support the `-g` flag to filter projects by their `groups` attribute.

---

### `fleet init`

Clone repositories and configure fetch / push remotes.

```bash
fleet init [-g <group>]
```

**Behavior:**

- **Repo does not exist** — Runs `git clone` and configures push remote (if any)
- **Repo already exists** — Idempotent check and fix of remote configuration (auto-corrects URL mismatches)

**Example output:**

```
  ▸ Default push remote: fork
  ▸ Local manifest: /path/to/local_manifest.xml

Initializing 5 projects...
  ✓ [1/5] services/user-service (cloned)
  ✓ [2/5] services/order-service (cloned)
  ✓ [3/5] services/svc-a (configured)
  – [4/5] services/svc-b (skipped)
  ✗ [5/5] services/svc-c: clone failed: ...

3 cloned, 1 configured, 1 skipped
```

`fleet init` is safe to run repeatedly. Already-configured repos are skipped.

---

### `fleet sync`

Pull latest code from the upstream fetch remote.

```bash
fleet sync [-g <group>]
```

**Behavior:**

- Current branch == default branch → `git pull --rebase`
- Current branch != default branch (feature branch) → `git fetch` only (no merge)
- Repo not cloned → skip with warning
- Remote fallback: if the manifest-configured remote doesn't exist locally, falls back to `origin`

**Example output:**

```
Syncing 3 projects...
  ✓ [1/3] services/user-service (rebased)
  ✓ [2/3] services/order-service (fetched)
  – [3/3] services/svc-a (skipped)

1 rebased, 1 fetched, 1 skipped
```

---

### `fleet status`

Display the status of all repositories in a table format.

```bash
fleet status [-g <group>]
```

**Example output:**

```
PROJECT                  BRANCH              STATUS     AHEAD/BEHIND
──────────────────────────────────────────────────────────────────────
user-service             master              clean
order-service            feature/new-api     dirty      +3 -1
payment-gateway          –                   not cloned
```

**Color rules:**

- Branch name: yellow when not matching default branch
- Status: `clean` in green, `dirty` in yellow
- Not cloned: entire row in grey

---

### `fleet push`

Push the current branch to the push remote.

```bash
fleet push [-g <group>] [--all]
```

**Behavior:**

- By default, only pushes feature branches (non-default branches)
- `--all` mode pushes all branches including the default branch
- Automatically skips: not cloned, no push remote, detached HEAD, push remote not found locally

**Example output:**

```
  ▸ Push remote: fork

Pushing 3 projects...
  ✓ [1/3] services/user-service (pushed)
  – [2/3] services/order-service (skipped)
  – [3/3] services/svc-a (skipped)

1 pushed, 2 skipped
```

---

### `fleet forall`

Execute a command across all repositories.

```bash
# Shell mode
fleet forall [-g <group>] -c "<command>"

# Direct execution mode
fleet forall [-g <group>] -- <command> [args...]
```

**Environment variables:**

The following environment variables are available in the command execution context:

| Variable | Description |
|----------|-------------|
| `FLEET_PROJECT_NAME` | Project name |
| `FLEET_PROJECT_PATH` | Relative path |
| `FLEET_PROJECT_GROUPS` | Group list (comma-separated) |
| `FLEET_PROJECT_REMOTE` | Fetch remote name |
| `FLEET_PROJECT_REVISION` | Default branch |
| `FLEET_PROJECT_PUSH_REMOTE` | Push remote name |

**Examples:**

```bash
# List branches in all repos
fleet forall -c "git branch"

# View recent commits in the commerce group
fleet forall -g commerce -- git log --oneline -5

# Batch run go mod tidy
fleet forall -c "go mod tidy"
```

A command failure in one project does not interrupt the overall flow — a warning is emitted and execution continues.

---

## Manifest Configuration

### Element Reference

| Element | Description |
|---------|-------------|
| `<remote>` | Defines a Git remote endpoint (`name`, `fetch`, `review`) |
| `<default>` | Default values for all projects (`remote`, `revision`, `sync-j`, `push`) |
| `<project>` | Defines a managed Git repository (`name`, `path`, `groups`, `remote`, `revision`, `push`) |

### Merge Rules

When `local_manifest.xml` exists, it is merged with `default.xml` according to these rules:

- **Remote** — Same-name remotes are fully replaced; new remotes are appended
- **Default** — Per-attribute override (non-empty local attributes overwrite the corresponding base attributes)
- **Project** — Same-name projects use per-attribute override; new projects are appended

### URL Construction

```
clone_url = remote.fetch + project.name + ".git"
push_url  = push_remote.fetch + project.name + ".git"
```

An independent push remote is only configured when the push remote differs from the fetch remote.

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLEET_MANIFEST` | Path to main manifest file | `<workspace_root>/default.xml` |
| `FLEET_LOCAL_MANIFEST` | Path to local override manifest | `<workspace_root>/local_manifest.xml` |

```bash
FLEET_MANIFEST=~/custom/default.xml fleet status
FLEET_LOCAL_MANIFEST=~/custom/local.xml fleet init
```

---

## Typical Workflows

### First-Time Setup

```bash
cd my-workspace
# Create personal local_manifest.xml with fork remote
vim local_manifest.xml
# Initialize all repos
fleet init
# Or initialize only a specific group
fleet init -g core
```

### Daily Development

```bash
# Check status of all repos
fleet status

# Sync from upstream
fleet sync

# Push feature branches to fork
fleet push

# Push including default branch
fleet push --all
```

### Adding a New Service

1. Add a `<project>` entry in `default.xml`
2. Run `fleet init` — the new repo is cloned; existing repos are unaffected

### Adding a Private Repository

1. Add a `<project>` entry in `local_manifest.xml`
2. Run `fleet init`
