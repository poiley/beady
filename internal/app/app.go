package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/poiley/beady/internal/bd"
	"github.com/poiley/beady/internal/models"
	"github.com/poiley/beady/internal/views"
)

// ViewMode tracks which view is active.
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
)

// dataLoadedMsg is sent when data is loaded from bd.
type dataLoadedMsg struct {
	issues      []models.Issue
	readyIssues []models.Issue
	stats       *models.StatsSummary
	err         error
	quiet       bool // true for auto-refresh (don't flash loading screen)
}

// detailLoadedMsg is sent when a detail view loads.
type detailLoadedMsg struct {
	issue *models.Issue
	err   error
	quiet bool // true for auto-refresh
}

// statusClearMsg signals that the status message should be cleared.
type statusClearMsg struct{}

// App is the root Bubble Tea model.
type App struct {
	client   *bd.Client
	workDir  string
	watcher  *dbWatcher
	list     *views.ListView
	detail   *views.DetailView
	help     *views.HelpView
	viewMode ViewMode
	showHelp bool
	width    int
	height   int
	err      error
	loading  bool

	// Temporary status bar message (e.g., "copied kubrick-drj").
	statusMsg string

	// Navigation stack for detail -> dependency drill-down
	detailStack []*views.DetailView
}

// New creates a new App model.
func New(workDir string) *App {
	return &App{
		client:   bd.NewClient(workDir),
		workDir:  workDir,
		watcher:  newDBWatcher(workDir),
		list:     views.NewListView(),
		help:     views.NewHelpView(),
		viewMode: ViewList,
		loading:  true,
	}
}

// Init runs the initial command.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.loadData(),
		a.watcher.waitForChange(),
	)
}

// Update is the Bubble Tea update function.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(msg.Width, msg.Height)
		if a.detail != nil {
			a.detail.SetSize(msg.Width, msg.Height)
		}
		a.help.SetSize(msg.Width, msg.Height)
		return a, nil

	case fileChangedMsg:
		// Database changed on disk — silently reload data in the background.
		// Don't set a.loading (no loading screen flash).
		var reloadCmd tea.Cmd
		switch a.viewMode {
		case ViewList:
			reloadCmd = a.loadDataQuiet()
		case ViewDetail:
			if a.detail != nil {
				reloadCmd = a.loadDetailQuiet(a.detail.IssueID())
			}
		}
		// Re-arm the watcher for the next change
		return a, tea.Batch(reloadCmd, a.watcher.waitForChange())

	case dataLoadedMsg:
		if !msg.quiet {
			a.loading = false
		}
		if msg.err != nil {
			// Quiet refreshes silently ignore errors — stale data is better
			// than flashing an error the user didn't ask for.
			if msg.quiet {
				return a, nil
			}
			a.err = msg.err
			return a, nil
		}
		a.err = nil
		hasFlashes := a.list.SetData(msg.issues, msg.readyIssues, msg.stats)
		if hasFlashes {
			return a, tea.Tick(views.FlashDuration(), func(t time.Time) tea.Msg {
				return views.FlashExpiredMsg{}
			})
		}
		return a, nil

	case views.FlashExpiredMsg:
		a.list.ClearFlashes()
		return a, nil

	case statusClearMsg:
		a.statusMsg = ""
		a.list.SetStatusMsg("")
		if a.detail != nil {
			a.detail.SetStatusMsg("")
		}
		return a, nil

	case detailLoadedMsg:
		if !msg.quiet {
			a.loading = false
		}
		if msg.err != nil {
			if msg.quiet {
				return a, nil
			}
			a.err = msg.err
			return a, nil
		}
		a.err = nil
		if msg.quiet && a.detail != nil {
			// Quiet refresh: update the existing detail in-place
			a.detail.UpdateIssue(msg.issue)
		} else {
			a.detail = views.NewDetailView(msg.issue)
			a.detail.SetSize(a.width, a.height)
			a.viewMode = ViewDetail
		}
		return a, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			a.watcher.close()
			return a, tea.Quit
		case "q":
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if a.viewMode == ViewDetail {
				a.popDetail()
				return a, nil
			}
			if a.list.IsFiltering() {
				// let list handle it
			} else {
				a.watcher.close()
				return a, tea.Quit
			}
		case "?":
			a.showHelp = !a.showHelp
			return a, nil
		case "r":
			// Global retry when in error state
			if a.err != nil {
				a.err = nil
				a.loading = true
				return a, a.loadData()
			}
		}

		// If help is showing, close it on any other key
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}

		// View-specific handling
		switch a.viewMode {
		case ViewList:
			return a.updateList(msg)
		case ViewDetail:
			return a.updateDetail(msg)
		}
	}

	// Pass through to active view for non-key messages (e.g., blink)
	if a.viewMode == ViewList {
		cmd := a.list.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if !a.list.IsFiltering() {
			if issue := a.list.SelectedIssue(); issue != nil {
				a.loading = true
				return a, a.loadDetail(issue.ID)
			}
		}
	case "r":
		if !a.list.IsFiltering() {
			a.loading = true
			return a, a.loadData()
		}
	case "y":
		if !a.list.IsFiltering() {
			if issue := a.list.SelectedIssue(); issue != nil {
				copyToClipboard(issue.ID)
				return a, a.setStatus(fmt.Sprintf("copied %s", issue.ID))
			}
			return a, nil
		}
	}

	cmd := a.list.Update(msg)
	return a, cmd
}

func (a *App) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.popDetail()
		return a, nil
	case "r":
		if a.detail != nil {
			a.loading = true
			return a, a.loadDetail(a.detail.IssueID())
		}
	case "y":
		if a.detail != nil {
			copyToClipboard(a.detail.IssueID())
			return a, a.setStatus(fmt.Sprintf("copied %s", a.detail.IssueID()))
		}
		return a, nil
	}

	if a.detail != nil {
		cmd := a.detail.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a *App) popDetail() {
	if len(a.detailStack) > 0 {
		a.detail = a.detailStack[len(a.detailStack)-1]
		a.detailStack = a.detailStack[:len(a.detailStack)-1]
	} else {
		a.detail = nil
		a.viewMode = ViewList
	}
}

// View renders the current view.
func (a *App) View() string {
	if a.showHelp {
		return a.help.View()
	}

	if a.err != nil {
		errMsg := ui_errorView(a.err, a.width, a.height)
		return errMsg
	}

	if a.loading {
		return loadingView(a.width, a.height)
	}

	switch a.viewMode {
	case ViewDetail:
		if a.detail != nil {
			return a.detail.View()
		}
	}

	return a.list.View()
}

func (a *App) loadData() tea.Cmd {
	return a.loadDataWithOpts(false)
}

func (a *App) loadDataQuiet() tea.Cmd {
	return a.loadDataWithOpts(true)
}

func (a *App) loadDataWithOpts(quiet bool) tea.Cmd {
	return func() tea.Msg {
		issues, err := a.client.ListAll()
		if err != nil {
			return dataLoadedMsg{err: err, quiet: quiet}
		}

		// Ready and stats are non-fatal if they fail
		readyIssues, _ := a.client.Ready()
		stats, _ := a.client.Stats()

		return dataLoadedMsg{
			issues:      issues,
			readyIssues: readyIssues,
			stats:       stats,
			quiet:       quiet,
		}
	}
}

func (a *App) loadDetail(id string) tea.Cmd {
	return a.loadDetailWithOpts(id, false)
}

func (a *App) loadDetailQuiet(id string) tea.Cmd {
	return a.loadDetailWithOpts(id, true)
}

func (a *App) loadDetailWithOpts(id string, quiet bool) tea.Cmd {
	return func() tea.Msg {
		issue, err := a.client.Show(id)
		return detailLoadedMsg{issue: issue, err: err, quiet: quiet}
	}
}

func (a *App) setStatus(msg string) tea.Cmd {
	a.statusMsg = msg
	a.list.SetStatusMsg(msg)
	if a.detail != nil {
		a.detail.SetStatusMsg(msg)
	}
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusClearMsg{}
	})
}

func copyToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run()
}

func loadingView(width, height int) string {
	msg := "Loading beads..."
	return fmt.Sprintf("%*s", width/2+len(msg)/2, msg)
}

func ui_errorView(err error, width, height int) string {
	msg := fmt.Sprintf("Error: %s\n\nPress 'r' to retry or 'q' to quit.", err)
	return fmt.Sprintf("\n\n  %s", msg)
}
