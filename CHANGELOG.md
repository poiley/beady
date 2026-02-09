# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Fix panic: increase header array size from 8 to 9 columns to match actual column count

### Added
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

[Unreleased]: https://github.com/poiley/beady/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/poiley/beady/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/poiley/beady/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/poiley/beady/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/poiley/beady/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/poiley/beady/releases/tag/v0.1.0
