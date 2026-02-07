# Agent Instructions

## Project Overview

**bdy** (beady) is a read-only, k9s-style terminal UI for browsing [beads](https://github.com/steveyegge/beads) issues. Built in Go with Bubble Tea / Lipgloss.

- **Module path:** `github.com/poiley/beady`
- **Binary name:** `bdy` (built from `cmd/bdy/main.go`)
- **Architecture:** Shells out to the `bd` CLI with `--json` for all data. No direct database access.

## Project Structure

```
cmd/bdy/main.go              Entry point, CLI flags, version vars (set via ldflags)
internal/
  app/app.go                  Root Bubble Tea model, navigation stack, data loading
  bd/client.go                bd CLI wrapper (exec bd --json, parse response)
  models/issue.go             Data structs matching bd JSON output
  selfupdate/update.go        Self-update via GitHub Releases API
  ui/styles.go                Lipgloss styles and color theme
  views/
    list.go                   Main list view (table, sort, filter)
    detail.go                 Detail view (single issue, scrollable)
    help.go                   Help overlay (keybindings reference)
scripts/
  install.sh                  curl-pipe-bash installer
```

## Build & Test

```bash
make build       # Build binary with version from git tags
make install     # Install to $GOPATH/bin
make test        # Run tests
make vet         # go vet
```

Version is embedded via `-ldflags`:
- `main.Version` from `git describe --tags`
- `main.Commit` from `git rev-parse --short HEAD`
- `main.Date` from `date -u`

## Key Design Decisions

1. **Read-only** - No mutations. The TUI only reads via `bd list/show/ready/stats --json`.
2. **CLI wrapper, not library import** - Shells out to `bd` rather than importing beads Go packages. Avoids SQLite locking conflicts with the daemon and stays decoupled from beads internals.
3. **Single binary** - No config files, no runtime dependencies beyond `bd` in PATH.
4. **Elm architecture** - Standard Bubble Tea model/update/view pattern. All state in `App` struct, views are stateful sub-models.

## Release Process

```bash
git tag v0.X.0
git push origin main --tags
# Cross-compile and publish:
goreleaser release --clean
# Or manually: see Makefile + gh release create
```

## Issue Tracking

This project uses **bd** (beads) for issue tracking.

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
