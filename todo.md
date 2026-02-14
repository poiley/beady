# bdy Improvement Plan

## High Impact

### 1. Drill-down navigation from detail view
The `detailStack` in `app.go` already supports push/pop, but nothing pushes a second detail view. Add `enter` keybinding in detail view on dependency/dependent items to drill into them. `esc`/`q` already pops correctly.

- Files: `internal/views/detail.go`, `internal/app/app.go`
- Complexity: medium (need cursor/selection in detail view's dep/dependent sections)

### 2. Copy feedback
`y` copies the issue ID to clipboard but provides no visual confirmation. Add a brief status bar message or flash to confirm the copy succeeded.

- Files: `internal/app/app.go`, `internal/views/list.go`
- Complexity: low

### 3. Comment count column
`CommentCount` is already parsed from `bd list` JSON but never displayed. Add a small column (e.g., `CMT` or `#`) to surface issues with active discussion.

- Files: `internal/views/list.go`
- Complexity: low

### 4. Overdue/due date indicator
`DueAt` is parsed but only shown in detail view. Highlight overdue issues in the list with a red marker or modified age display.

- Files: `internal/views/list.go`
- Complexity: low

## Medium Impact

### 5. Pinned indicator and sort-to-top
`Pinned` field is parsed but completely ignored. Add a pin marker on pinned issue IDs and sort them above unpinned issues at the same priority.

- Files: `internal/views/list.go`
- Complexity: low

### 6. Label search in text filter
The `/` text filter searches ID, title, type, and assignee but not labels. Add labels to the search fields.

- Files: `internal/views/list.go`
- Complexity: trivial

### 7. Detail view text wrapping fix
`wrapLine()` in `detail.go` uses `len()` (byte length) instead of `runewidth.StringWidth()`. CJK and other wide characters overflow the wrap boundary. The table engine already handles this correctly.

- Files: `internal/views/detail.go`
- Complexity: low

### 8. Loading and error view styling
Both `loadingView()` and `ui_errorView()` are unstyled raw text. Center them vertically and use existing `ErrorStyle`/`BorderStyle` from `styles.go`.

- Files: `internal/app/app.go`
- Complexity: low

## Polish

### 9. Deferred/pinned status filters
Filters 1-5 cover open/in_progress/blocked/closed/ready but skip deferred and pinned. Add `6` for deferred and `7` for pinned.

- Files: `internal/views/list.go`
- Complexity: trivial

### 10. Dead code cleanup
- `client.List()` is never called (only `ListAll()` is used) — remove it
- Duplicate flash detection loop in `SetData()` — the second loop checking for new issues is redundant since the first loop already catches them via `!existed`

- Files: `internal/bd/client.go`, `internal/views/list.go`
- Complexity: trivial
