# bdy (beady)

A k9s-style terminal UI for browsing [beads](https://github.com/steveyegge/beads) issues.

Full-screen, keyboard-driven, read-only viewer for your local beads database. Designed for single-repo, single-user workflows where you want to quickly see what's in the queue.

![bdy demo](demo.gif)

## Install

**Homebrew** (macOS):

```bash
brew install poiley/tap/bdy
```

**Install script** (macOS, Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/poiley/beady/main/scripts/install.sh | bash
```

**With Go**:

```bash
go install github.com/poiley/beady/cmd/bdy@latest
```

**From source**:

```bash
git clone https://github.com/poiley/beady.git
cd beady
make install
```

### Prerequisites

- [bd](https://github.com/steveyegge/beads) CLI installed and in PATH
- A project with `bd init` already run

## Usage

```bash
cd your-project
bdy
```

Or point it at a directory:

```bash
bdy /path/to/project
```

### Commands

```
bdy              Launch the TUI
bdy update       Self-update to the latest release
bdy check        Verify bd CLI is available and beads is initialized
bdy version      Show version, commit, and build date
bdy help         Show help
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+u` / `Ctrl+d` | Page up / down |
| `Enter` | Open issue detail view |
| `Esc` | Back to list / cancel filter |

### Sorting

| Key | Action |
|-----|--------|
| `s` | Cycle sort field: priority > created > updated > status > type > id |
| `S` | Reverse sort direction |

Default sort is by **priority** (P0 first), then by **created date** (newest first).

### Filtering

| Key | Action |
|-----|--------|
| `/` | Text search (matches title, ID, type, assignee) |
| `1` | Toggle: open only |
| `2` | Toggle: in_progress only |
| `3` | Toggle: blocked only |
| `4` | Toggle: closed only |
| `5` | Toggle: ready (unblocked) only |
| `0` | Show all statuses |
| `c` | Toggle: show/hide closed issues (hidden by default) |

### Actions

| Key | Action |
|-----|--------|
| `r` | Refresh data from bd |
| `y` | Copy issue ID to clipboard |
| `?` | Toggle help overlay |
| `q` | Quit (or back from detail view) |

## Views

### List view

The default view. Full-screen table showing all issues with columns for ID, priority, status, type, title, assignee, age, and dependency counts.

- Priority is color-coded: P0 red, P1 yellow, P2 white, P3 gray
- Status is color-coded: open green, in_progress cyan, blocked red, closed gray
- Header shows aggregate counts from `bd stats`

### Detail view

Press `Enter` on any issue to see full details: all metadata fields, description, design notes, acceptance criteria, notes, dependencies (with type), dependents, and comments. Scrollable with `j`/`k`.

### Help overlay

Press `?` from anywhere to see all keybindings.

## How it works

bdy is a thin UI layer that shells out to the `bd` CLI with `--json` for all data:

- `bd list --all --json` for the issue table
- `bd ready --json` for the ready filter
- `bd show <id> --json` for detail views
- `bd stats --json` for the header counts

It never touches the `.beads/` database directly. No write operations. No daemon interaction.

## Architecture

```
cmd/bdy/main.go              Entry point, CLI flags, self-update
internal/
  app/app.go                  Root Bubble Tea model, navigation, data loading
  bd/client.go                bd CLI wrapper (exec + JSON parse)
  models/issue.go             Issue/Comment/Stats structs
  selfupdate/update.go        GitHub Releases self-updater
  ui/
    styles.go                 k9s-inspired Lipgloss color theme
    table.go                  Generic table layout engine (Fixed/Fit/Flex columns)
  views/
    list.go                   Main table view (sort, filter, scroll)
    detail.go                 Single issue detail view
    help.go                   Help overlay
scripts/
  install.sh                  curl-pipe-bash installer
```

Built with:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - Text input component
- [go-runewidth](https://github.com/mattn/go-runewidth) - Unicode-aware string width

## Updating

```bash
bdy update
```

This checks GitHub Releases for a newer version, downloads the binary for your platform, and replaces the running binary in place.

## Building

```bash
make build       # Build with version info from git
make install     # Install to $GOPATH/bin
make test        # Run tests
make vet         # go vet
make version     # Print current version
```

Version, commit hash, and build date are embedded via `-ldflags` from git tags.

## Releasing

Tag, push, and publish with [goreleaser](https://goreleaser.com/):

```bash
git tag v1.x.x
git push origin main --tags
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
```

This cross-compiles for 6 platforms (linux/darwin/windows, amd64/arm64), publishes a GitHub Release, and auto-updates the Homebrew cask in [poiley/homebrew-tap](https://github.com/poiley/homebrew-tap).

## License

[MIT](LICENSE)
