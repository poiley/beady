# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.1] - 2026-02-14

### Fixed
- Fix status bar not sticking to terminal bottom (overhead was 6, should be 4)

## [1.2.0] - 2026-02-14

### Added
- Drill-down navigation in detail view: Tab/Shift+Tab to cycle through dependencies, Enter to drill in, Esc to pop back
- Pinned issue indicator (`*` prefix in yellow) with sort-to-top
- DUE column with red overdue highlighting for non-closed issues
- CMT column showing comment count
- Copy feedback: status bar shows "copied <id>" for 2 seconds on `y`
- Deferred (`6`) and pinned (`7`) status filters
- Labels now included in `/` text filter search

### Changed
- Styled loading and error views with centered layout, themed colors, and bordered error box
- Renamed `ui_errorView` to `errorView` (Go naming convention)

### Fixed
- Text wrapping for CJK/wide characters in detail view (use runewidth instead of byte length)

### Removed
- Dead code: unused `client.List()` method and duplicate flash detection loop

## [1.1.4] - 2026-02-14

### Fixed
- Fix footer not sticking to bottom: account for table header border line in overhead calculation

## [1.1.3] - 2026-02-14

### Changed
- Rename dependency count column header from `↑/↓` to `DEPS` for clarity

## [1.1.2] - 2026-02-14

### Fixed
- Fix DONE column: use dependency graph (`parent-child` type) instead of ID pattern matching for counting closed children

## [1.1.1] - 2026-02-14

### Added
- Add CHANGELOG.md with complete release history
- Fix panic: increase header array size from 8 to 9 columns
- Add DONE column showing epic/parent completion progress (closed/total dependents)

### Changed
- Change DEPS column header to ↑/↓ for clarity
- Promote fsnotify to direct dependency in go.mod

## [1.1.0] - 2026-02-07

### Added
- Live updates via fsnotify file watching with k9s-style pulse flare
- Demo GIF to README showing TUI in action

### Changed
- Update documentation: add Homebrew install instructions, `c` keybinding, table.go in architecture notes, release process

## [1.0.2] - 2026-02-07

### Changed
- Migrate from deprecated `brews` to `homebrew_casks` in goreleaser config
- Add xattr quarantine fix for macOS downloads

## [1.0.1] - 2026-02-07

### Added
- Homebrew tap configuration in goreleaser
- Add `.gitignore` entry for `dist/` directory

### Changed
- Promote runewidth to direct dependency
- Fix install script to verify newly installed binary, not just PATH

## [1.0.0] - 2026-02-07

### Added
- Generic table layout engine with unicode-safe column alignment (Fixed/Fit/Flex column types)
- Comprehensive README documentation
- MIT License

### Fixed
- Fix trap variable handling in install script

## [0.1.0] - 2026-02-07

### Added
- Initial release of bdy (beady) - a k9s-style TUI for beads issue tracking
- Full-screen terminal UI with k9s-inspired dark theme
- Browse all beads issues with sortable/filterable table view
- Detail view with full issue content, dependencies, dependents, comments
- Sort by priority, created, updated, status, type, or ID
- Filter by status (open/in_progress/blocked/closed/ready) or text search
- Vim-style navigation (j/k, g/G, Ctrl+u/d)
- Copy issue ID to clipboard (y)
- Self-update command (`bdy update`)
- Cross-platform support (macOS, Linux, Windows on amd64/arm64)
- Curl-pipe-bash installer script
- GoReleaser configuration for automated releases

[Unreleased]: https://github.com/poiley/beady/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/poiley/beady/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/poiley/beady/compare/v1.1.4...v1.2.0
[1.1.4]: https://github.com/poiley/beady/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/poiley/beady/compare/v1.1.2...v1.1.3
[1.1.2]: https://github.com/poiley/beady/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/poiley/beady/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/poiley/beady/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/poiley/beady/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/poiley/beady/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/poiley/beady/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/poiley/beady/releases/tag/v0.1.0
