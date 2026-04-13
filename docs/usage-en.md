# Fleet User Guide

Fleet is a multi-repo workspace management tool written in Go. It declaratively manages multiple Git repositories via manifest XML files, supporting batch clone, sync, branch management, push, PR creation, and cross-repo command execution under the GitHub Fork workflow.

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
├── fleet.xml              # Shared team config (committed to Git)
├── local_fleet.xml       # Personal local overrides (gitignored)
└── services/
    ├── user-service/
    ├── order-service/
    └── ...
```

### 2. Configure Manifests

**fleet.xml** — Shared team configuration:

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

**local_fleet.xml** — Personal fork configuration:

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
  -g, --group <expr>    Filter projects by group expression
  -h, --help            Show help
```

All subcommands support the `-g` flag to filter projects by their `groups` attribute.

**Group filter expressions** support `,` for OR and `+` for AND:

| Expression | Meaning |
|------------|---------|
| `core,web` | Projects in group `core` **OR** `web` |
| `core+backend` | Projects in group `core` **AND** `backend` |
| `core+backend,infra` | (`core` AND `backend`) **OR** `infra` |

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
  ▸ Local manifest: /path/to/local_fleet.xml

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
PROJECT              BRANCH              STATUS     AHEAD/BEHIND  FETCH              PUSH
─────────────────────────────────────────────────────────────────────────────────────────────
user-service         master              clean                    github/master      fork
order-service        feature/new-api     dirty      +3 -1        github/master      fork
payment-gateway      –                   not cloned               github/master      fork
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

### `fleet start`

Create and switch to a new branch across all repositories, based on the upstream default branch.

```bash
fleet start [-g <group>] <branch>
```

**Behavior:**

- Already on the target branch → skip
- Target branch already exists locally → switch to it (`git checkout`)
- Target branch does not exist → fetch from remote, then create from `<remote>/<default-branch>` (`git checkout -b`)
- Supports `master-main-compat`: if `master` is not found on the remote, automatically tries `main` (and vice versa)

**Example output:**

```
Starting branch feature/new-api across 3 projects...
  ✓ [1/3] services/user-service (created from github/master)
  ✓ [2/3] services/order-service (switched)
  – [3/3] services/svc-a (skipped: not cloned)

1 created, 1 switched, 1 skipped
```

---

### `fleet finish`

Delete a branch and switch back to the default branch across all repositories.

```bash
fleet finish [-g <group>] [-r] <branch>
```

**Options:**

| Flag | Description |
|------|-------------|
| `-r, --remote` | Also delete the branch on the push remote |

**Behavior:**

- Branch does not exist locally → skip
- Currently on the target branch → switch to the default branch first, then delete
- `-r` flag → also runs `git push <push-remote> --delete <branch>`

**Example output:**

```
Finishing branch feature/new-api across 3 projects...
  ✓ [1/3] services/user-service (finished)
  – [2/3] services/order-service (skipped: branch feature/new-api not found)
  – [3/3] services/svc-a (skipped: not cloned)

1 finished, 2 skipped
```

---

### `fleet pr`

Push the current branch and create a pull request via `gh` CLI across all repositories.

```bash
fleet pr [-g <group>] [-t <title>]
```

**Options:**

| Flag | Description |
|------|-------------|
| `-t, --title` | PR title (defaults to branch name) |

**Prerequisites:** Requires the [GitHub CLI (`gh`)](https://cli.github.com/) to be installed and authenticated.

**Behavior:**

- Pushes the current branch to the push remote
- Creates a PR targeting the upstream default branch (fetch remote)
- For fork workflows, automatically sets `--head` to `<fork-owner>:<branch>`
- Skips repos on the default branch, with detached HEAD, or without a push remote

**Example output:**

```
Creating PRs for 2 projects...
  ✓ [1/2] services/user-service (created https://github.com/my-org/user-service/pull/42)
  – [2/2] services/order-service (skipped: on default branch)

1 created, 1 skipped
```

---

### `fleet worktree`

Create a git worktree for every managed repository under a shared base directory, mirroring the original workspace structure.

```bash
fleet worktree <name> [-b <branch>] [-r <revision>] [-g <group>]
```

**Options:**

| Flag | Description |
|------|-------------|
| `-b, --branch` | Branch name to create or check out (default: same as `name`) |
| `-r, --revision` | Upstream revision to base the new branch on (default: each project's `revision` in fleet.xml) |

**Use case:** Work on a second feature in parallel without switching branches in your main workspace. Each repo gets a worktree at `<worktree-base>/<name>/<proj.path>`, preserving the same directory structure.

**Configuration** — add `worktree-base` (required) and `worktree-copy` (optional) to `<default>` in `fleet.xml`:

```xml
<default remote="github"
         revision="main"
         sync-j="4"
         worktree-base="~/worktrees/myproject"
         worktree-copy=".env,.env.*" />
```

| Attribute | Description |
|-----------|-------------|
| `worktree-base` | Base directory for all worktrees. Supports `~` expansion. Required to use `fleet worktree`. |
| `worktree-copy` | Comma-separated glob patterns for gitignored files to copy into each new worktree (e.g. `.env`). Inherited by all projects; individual projects can override with their own `worktree-copy` attribute. |

**Behavior:**

- Worktrees are placed at `<worktree-base>/<name>/<proj.path>`
- If the branch does not exist locally, it is created from `<remote>/<revision>`
- The `<revision>` flag overrides the per-project `revision` field for branch creation
- Workspace root project (`path="."`) is processed first to avoid directory races with parallel service projects
- Files matching `worktree-copy` patterns are copied from the source repo into the new worktree after creation
- Worktree already exists at the target path → skip

**Example output:**

```
Creating worktree JIRA-123 across 3 projects...
  ✓ [1/3] . (created)
  ✓ [2/3] services/user-service (created)
  ✓ [3/3] services/order-service (created)

3 created
```

The resulting worktree workspace mirrors your original layout:

```
~/worktrees/myproject/
└── JIRA-123/
    ├── .                      ← workspace repo worktree
    └── services/
        ├── user-service/      ← worktree on branch JIRA-123
        └── order-service/     ← worktree on branch JIRA-123
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

### `fleet ide-setup idea`

Generate IntelliJ IDEA / GoLand VCS directory mappings for all repositories in the workspace.

```bash
fleet ide-setup idea [-g <group>]
```

**Behavior:**

- Creates `.idea/vcs.xml` in the workspace root
- Adds a VCS mapping for the root project and each cloned sub-project
- Uncloned projects are skipped

This allows IntelliJ-based IDEs to recognize all repos when opening the workspace root as a project.

---

## Manifest Configuration

### Element Reference

| Element | Description |
|---------|-------------|
| `<remote>` | Defines a Git remote endpoint (`name`, `fetch`, `review`) |
| `<default>` | Default values for all projects (`remote`, `revision`, `sync-j`, `push`, `master-main-compat`, `worktree-base`, `worktree-copy`) |
| `<project>` | Defines a managed Git repository (`name`, `path`, `groups`, `remote`, `revision`, `push`, `worktree-copy`) |

### `master-main-compat` Attribute

When `master-main-compat="true"` is set on `<default>`, Fleet automatically falls back between `master` and `main` if the configured revision branch is not found on the remote. This is useful for workspaces where some repos use `master` and others use `main`.

```xml
<default remote="github" revision="master" sync-j="4" master-main-compat="true" />
```

### Merge Rules

When `local_fleet.xml` exists, it is merged with `fleet.xml` according to these rules:

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
| `FLEET_MANIFEST` | Path to main manifest file | `<workspace_root>/fleet.xml` |
| `FLEET_LOCAL_MANIFEST` | Path to local override manifest | `<workspace_root>/local_fleet.xml` |

```bash
FLEET_MANIFEST=~/custom/fleet.xml fleet status
FLEET_LOCAL_MANIFEST=~/custom/local.xml fleet init
```

---

## Typical Workflows

### First-Time Setup

```bash
cd my-workspace
# Create personal local_fleet.xml with fork remote
vim local_fleet.xml
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

# Start a new feature branch across all repos
fleet start feature/my-feature

# Push feature branches to fork
fleet push

# Push and create PRs
fleet pr -t "feat: my feature"

# Clean up after merge
fleet finish feature/my-feature

# Push including default branch
fleet push --all
```

### Adding a New Service

1. Add a `<project>` entry in `fleet.xml`
2. Run `fleet init` — the new repo is cloned; existing repos are unaffected

### Adding a Private Repository

1. Add a `<project>` entry in `local_fleet.xml`
2. Run `fleet init`
