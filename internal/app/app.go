package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

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
}

// detailLoadedMsg is sent when a detail view loads.
type detailLoadedMsg struct {
	issue *models.Issue
	err   error
}

// App is the root Bubble Tea model.
type App struct {
	client   *bd.Client
	list     *views.ListView
	detail   *views.DetailView
	help     *views.HelpView
	viewMode ViewMode
	showHelp bool
	width    int
	height   int
	err      error
	loading  bool

	// Navigation stack for detail -> dependency drill-down
	detailStack []*views.DetailView
}

// New creates a new App model.
func New(workDir string) *App {
	return &App{
		client:   bd.NewClient(workDir),
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

	case dataLoadedMsg:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.err = nil
		a.list.SetData(msg.issues, msg.readyIssues, msg.stats)
		return a, nil

	case detailLoadedMsg:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.err = nil
		a.detail = views.NewDetailView(msg.issue)
		a.detail.SetSize(a.width, a.height)
		a.viewMode = ViewDetail
		return a, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
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
	return func() tea.Msg {
		issues, err := a.client.ListAll()
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		// Ready and stats are non-fatal if they fail
		readyIssues, _ := a.client.Ready()
		stats, _ := a.client.Stats()

		return dataLoadedMsg{
			issues:      issues,
			readyIssues: readyIssues,
			stats:       stats,
		}
	}
}

func (a *App) loadDetail(id string) tea.Cmd {
	return func() tea.Msg {
		issue, err := a.client.Show(id)
		return detailLoadedMsg{issue: issue, err: err}
	}
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
