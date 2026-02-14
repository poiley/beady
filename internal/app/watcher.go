package app

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 500 * time.Millisecond
	pollInterval  = 3 * time.Second
)

// fileChangedMsg signals that the beads database has been modified on disk.
type fileChangedMsg struct{}

// dbWatcher watches the .beads/ directory for SQLite database changes
// and emits fileChangedMsg via a channel that Bubble Tea can consume.
//
// Uses a hybrid approach: fsnotify for near-instant detection of local
// mutations, plus a polling fallback (stat-based) every 3 seconds to
// catch edge cases where fsnotify misses events (WAL checkpoint file
// recreation, remote daemon syncs, etc.).
type dbWatcher struct {
	watcher  *fsnotify.Watcher
	events   chan struct{} // debounced change signal
	done     chan struct{} // signals shutdown
	beadsDir string        // path to .beads/ directory
}

// newDBWatcher creates a watcher on the beads SQLite database files.
// Returns nil if the .beads directory doesn't exist or watching fails —
// the app falls back to manual refresh only.
func newDBWatcher(workDir string) *dbWatcher {
	beadsDir := filepath.Join(workDir, ".beads")

	// Check that .beads/ exists before attempting to watch
	if _, err := os.Stat(beadsDir); err != nil {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}

	dw := &dbWatcher{
		watcher:  w,
		events:   make(chan struct{}, 1),
		done:     make(chan struct{}),
		beadsDir: beadsDir,
	}

	// Watch individual DB files if they exist. The WAL file is the primary
	// write target in WAL mode — it changes on every transaction commit.
	// Also watch the main DB file for checkpoints.
	for _, name := range []string{"beads.db", "beads.db-wal"} {
		path := filepath.Join(beadsDir, name)
		if _, err := os.Stat(path); err == nil {
			_ = w.Add(path)
		}
	}

	// Also watch the directory itself so we catch newly created WAL files
	// (e.g., after a checkpoint removes and recreates the WAL).
	_ = w.Add(beadsDir)

	go dw.fsnotifyLoop()
	go dw.pollLoop()

	return dw
}

// fsnotifyLoop runs the debounced fsnotify event processing goroutine.
func (dw *dbWatcher) fsnotifyLoop() {
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case event, ok := <-dw.watcher.Events:
			if !ok {
				return
			}
			// Only care about writes and creates (new WAL file after checkpoint)
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			// Filter to only DB-related files
			base := filepath.Base(event.Name)
			if base != "beads.db" && base != "beads.db-wal" {
				continue
			}

			// Re-add the file to the watcher if it was created (handles
			// WAL file recreation after checkpoint).
			if event.Op&fsnotify.Create != 0 {
				_ = dw.watcher.Add(event.Name)
			}

			// Start/reset the debounce timer
			if timer == nil {
				timer = time.NewTimer(debounceDelay)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(debounceDelay)
			}

		case <-timerC:
			// Debounce window expired — emit a single change event
			timer = nil
			timerC = nil
			dw.notify()

		case _, ok := <-dw.watcher.Errors:
			if !ok {
				return
			}
			// Ignore watch errors silently — worst case the poll loop
			// picks up the change within pollInterval.

		case <-dw.done:
			if timer != nil {
				timer.Stop()
			}
			return
		}
	}
}

// pollLoop periodically stats the DB files and emits a change event if
// modification times have changed. This catches any changes that fsnotify
// misses (WAL recreation, remote daemon writes, platform quirks, etc.).
func (dw *dbWatcher) pollLoop() {
	lastMod := dw.dbModTime()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mod := dw.dbModTime()
			if mod != lastMod {
				lastMod = mod
				dw.notify()
			}
		case <-dw.done:
			return
		}
	}
}

// dbModTime returns the latest modification time across the DB and WAL files.
// Returns zero time if neither file can be stat'd.
func (dw *dbWatcher) dbModTime() time.Time {
	var latest time.Time
	for _, name := range []string{"beads.db", "beads.db-wal"} {
		if info, err := os.Stat(filepath.Join(dw.beadsDir, name)); err == nil {
			if t := info.ModTime(); t.After(latest) {
				latest = t
			}
		}
	}
	return latest
}

// notify sends a change event to the events channel (non-blocking).
func (dw *dbWatcher) notify() {
	select {
	case dw.events <- struct{}{}:
	default:
		// Channel already has a pending event, skip
	}
}

// waitForChange returns a tea.Cmd that blocks until a database change is
// detected, then sends a fileChangedMsg. This is designed to be called
// repeatedly — each invocation waits for the next change.
func (dw *dbWatcher) waitForChange() tea.Cmd {
	if dw == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case <-dw.events:
			return fileChangedMsg{}
		case <-dw.done:
			return nil
		}
	}
}

// close shuts down the watcher and its goroutine.
func (dw *dbWatcher) close() {
	if dw == nil {
		return
	}
	close(dw.done)
	dw.watcher.Close()
}
