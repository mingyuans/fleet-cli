# Fleet

A multi-repo workspace management tool for the GitHub Fork workflow, inspired by [Google's `repo` tool](https://gerrit.googlesource.com/git-repo/).

Like `repo`, Fleet uses XML manifest files to declaratively define a workspace of multiple Git repositories — but is purpose-built for the **GitHub Fork + Pull Request** workflow instead of Gerrit. It supports batch clone, sync, branch, push, create PRs, and run commands across all repos in one go.

[中文文档](docs/usage-zh.md)

## Install

```bash
# One-line install (macOS / Linux)
curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# Or specify a version / install directory
FLEET_VERSION=v0.1.0 FLEET_INSTALL_DIR=~/.local/bin \
  curl -sSfL https://raw.githubusercontent.com/mingyuans/fleet-cli/main/install.sh | sh

# Or build from source
git clone https://github.com/mingyuans/fleet-cli.git
cd fleet-cli
make install
```

## Quick Start

**1. Create a workspace with `fleet.xml`:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:my-org/" />
  <default remote="github" revision="master" sync-j="4" />

  <project name="user-service"    path="services/user-service"  groups="core" />
  <project name="order-service"   path="services/order-service" groups="commerce" />
</manifest>
```

**2. Add a personal `local_fleet.xml` for your fork:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:your-username/" />
  <default push="fork" />
</manifest>
```

**3. Clone everything:**

```bash
fleet init
```

## Commands

| Command | Description |
|---------|-------------|
| `fleet init` | Clone repos and configure remotes |
| `fleet sync` | Pull latest from upstream (rebase on default branch, fetch on feature branches) |
| `fleet status` | Show branch, dirty/clean status, ahead/behind for all repos |
| `fleet start <branch>` | Create a feature branch across all repos from upstream HEAD |
| `fleet finish <branch>` | Delete a branch and switch back to default (`-r` to delete remote too) |
| `fleet push` | Push current branch to fork (`--all` to include default branch) |
| `fleet pr` | Push and create PRs via `gh` CLI (`-t` to set title) |
| `fleet forall -c "cmd"` | Run a shell command in every repo |
| `fleet ide-setup idea` | Generate IntelliJ/GoLand VCS mappings |

All commands support `-g <expr>` to filter by group (`,` = OR, `+` = AND).

## Typical Workflow

```bash
fleet init                          # clone all repos
fleet sync                          # pull latest upstream
fleet start feature/my-feature      # create branch everywhere
# ... make changes ...
fleet push                          # push to fork
fleet pr -t "feat: my feature"      # create PRs
fleet finish feature/my-feature     # clean up after merge
```

## Manifest Reference

Fleet uses two XML files in the workspace root:

| File | Purpose |
|------|---------|
| `fleet.xml` | Shared team config (committed to Git) |
| `local_fleet.xml` | Personal overrides — fork remote, extra repos (gitignored) |

When both exist, `local_fleet.xml` is merged into `fleet.xml`:

- **Remotes** — same name replaces; new ones append
- **Default** — per-attribute override
- **Projects** — same name uses per-attribute override; new ones append

See [docs/example-fleet.xml](docs/example-fleet.xml) for a fully annotated example.

### Key attributes

| Element | Attributes |
|---------|------------|
| `<remote>` | `name`, `fetch`, `review` |
| `<default>` | `remote`, `revision`, `sync-j`, `push`, `master-main-compat` |
| `<project>` | `name`, `path`, `groups`, `remote`, `revision`, `push` |

Set `master-main-compat="true"` on `<default>` to auto-fallback between `master` and `main`.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLEET_MANIFEST` | Path to main manifest | `<workspace>/fleet.xml` |
| `FLEET_LOCAL_MANIFEST` | Path to local manifest | `<workspace>/local_fleet.xml` |

## Documentation

- [English User Guide](docs/usage-en.md)
- [中文使用说明](docs/usage-zh.md)
- [Example Manifest](docs/example-fleet.xml)

## License

MIT
