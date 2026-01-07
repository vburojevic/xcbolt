package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xcbolt/xcbolt/internal/core"
)

// =============================================================================
// Messages
// =============================================================================

type eventMsg core.Event

type contextLoadedMsg struct {
	info core.ContextInfo
	cfg  core.Config
	err  error
}

type ConfigOverrides struct {
	LogFormat        string
	LogFormatArgs    []string
	HasLogFormat     bool
	HasLogFormatArgs bool
}

type opDoneMsg struct {
	cmd   string
	err   error
	cfg   core.Config
	build *core.BuildResult
	run   *core.RunResult
	test  *core.TestResult
}

type tickMsg time.Time

// =============================================================================
// Result tracks the last operation result for display
// =============================================================================

type Result struct {
	Operation string // "build", "run", "test"
	Success   bool
	Duration  time.Duration
	Message   string
	Timestamp time.Time
}

// =============================================================================
// Model
// =============================================================================

type Model struct {
	// Project configuration
	projectRoot string
	configPath  string
	cfg         core.Config
	cfgOverride ConfigOverrides
	info        core.ContextInfo
	state       core.State  // User state (recents, favorites)
	gitBranch   string      // Current git branch

	// Window dimensions
	width  int
	height int

	// Styles and keys
	styles Styles
	keys   keyMap
	help   help.Model

	// UI mode
	mode         Mode
	selectorType SelectorType

	// Layout components
	layout      Layout
	statusBar   StatusBar
	progressBar ProgressBar
	hintsBar    HintsBar

	// Components
	spinner     spinner.Model
	viewport    viewport.Model
	wizard      wizardModel
	selector    SelectorModel
	palette     PaletteModel
	issuesPanel IssuesPanel

	// Issues panel state
	showIssues    bool
	issuesFocused bool

	// Status message (shown in results bar)
	statusMsg string

	// Logs
	logLines   []string
	autoFollow bool

	// Operation state
	running    bool
	runningCmd string
	cancelFn   context.CancelFunc
	eventCh    <-chan core.Event
	doneCh     <-chan opDoneMsg

	// Progress tracking (for stage indicators)
	currentStage  string
	stageProgress string // e.g., "23/47"
	progressCur   int
	progressTotal int

	// Results
	lastResult *Result
	lastBuild  core.BuildResult
	lastRun    core.RunResult
	lastTest   core.TestResult
	lastErr    string
}

// NewModel creates a new TUI model
func NewModel(projectRoot string, configPath string, overrides ConfigOverrides) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	h := help.New()
	h.ShowAll = false

	// Load user state (ignore errors - use defaults if not found)
	state, _ := core.LoadState()

	// Initialize layout components
	layout := NewLayout()
	statusBar := NewStatusBar()
	progressBar := NewProgressBar()
	hintsBar := NewHintsBar()
	issuesPanel := NewIssuesPanel()

	return Model{
		projectRoot: projectRoot,
		configPath:  configPath,
		cfgOverride: overrides,
		styles:      DefaultStyles(),
		keys:        defaultKeyMap(),
		help:        h,
		spinner:     sp,
		viewport:    vp,
		logLines:    []string{},
		autoFollow:  true,
		mode:        ModeNormal,
		state:       state,
		// Layout components
		layout:      layout,
		statusBar:   statusBar,
		progressBar: progressBar,
		hintsBar:    hintsBar,
		issuesPanel: issuesPanel,
	}
}

// setStatus updates the status message shown in the results bar
func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadContextCmd(m.projectRoot, m.configPath, m.cfgOverride),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func loadContextCmd(projectRoot, configPath string, overrides ConfigOverrides) tea.Cmd {
	return func() tea.Msg {
		cfg, err := core.LoadConfig(projectRoot, configPath)
		if err != nil {
			return contextLoadedMsg{err: err}
		}
		applyConfigOverrides(&cfg, overrides)
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		emit := core.NewTextEmitter(ioDiscard{})
		info, cfg2, err := core.DiscoverContext(ctx, projectRoot, cfg, emit)
		if err != nil {
			return contextLoadedMsg{err: err}
		}
		return contextLoadedMsg{info: info, cfg: cfg2}
	}
}

func applyConfigOverrides(cfg *core.Config, overrides ConfigOverrides) {
	if overrides.HasLogFormat {
		cfg.Xcodebuild.LogFormat = overrides.LogFormat
	}
	if overrides.HasLogFormatArgs {
		cfg.Xcodebuild.LogFormatArgs = overrides.LogFormatArgs
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

// =============================================================================
// Update
// =============================================================================

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update layout dimensions
		m.layout.SetSize(m.width, m.height)
		m.updateViewportSize()
		if m.mode == ModeWizard {
			m.wizard = newWizard(m.info, m.cfg, m.width)
		}
		// Responsive: warn if terminal is too small
		if m.width < 80 || m.height < 20 {
			m.setStatus("Terminal too small (min 80x20)")
		}

	case contextLoadedMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.setStatus("Context load failed")
			break
		}
		m.info = msg.info
		m.cfg = msg.cfg

		// Fetch git branch
		m.gitBranch = getGitBranch(m.projectRoot)

		// Auto-detect: if not configured but context found, auto-select defaults
		needsConfig := m.cfg.Scheme == "" || (m.cfg.Workspace == "" && m.cfg.Project == "")
		if needsConfig && m.tryAutoDetect() {
			// Auto-config applied - save and show toast
			if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err == nil {
				m.setStatus("Ready")
			} else {
				m.setStatus("Context ready")
			}
		} else {
			m.setStatus("Context ready")
		}

	case tickMsg:
		// Only continue ticking if we need animation (spinner while running)
		if m.running {
			cmds = append(cmds, tickCmd())
		}

	case tea.KeyMsg:
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case wizardDoneMsg:
		m.mode = ModeNormal
		if msg.aborted {
			m.setStatus("Init canceled")
			break
		}
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.setStatus("Init failed")
			break
		}
		m.cfg = msg.cfg
		if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err != nil {
			m.lastErr = err.Error()
			m.setStatus("Save failed")
			break
		}
		m.setStatus("Saved config")
		cmds = append(cmds, loadContextCmd(m.projectRoot, m.configPath, m.cfgOverride))

	case eventMsg:
		ev := core.Event(msg)
		m.handleEvent(ev)
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		if m.autoFollow {
			m.viewport.GotoBottom()
		}
		if m.eventCh != nil {
			cmds = append(cmds, waitForEvent(m.eventCh))
		}

	case opDoneMsg:
		m.handleOpDone(msg)
		cmds = append(cmds, loadContextCmd(m.projectRoot, m.configPath, m.cfgOverride))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		// Also update status bar spinner
		m.statusBar.Spinner = m.spinner
		cmds = append(cmds, cmd)

	case statusMsg:
		m.setStatus(string(msg))
	}

	// Viewport scrolling (only in normal mode and not focused on issues)
	if m.mode == ModeNormal && !m.issuesFocused {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		// Check if user scrolled up manually
		if m.viewport.AtBottom() {
			m.autoFollow = true
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) openSchemeSelector() {
	if len(m.info.Schemes) == 0 {
		m.setStatus("No schemes found")
		return
	}

	items := SchemeItems(m.info.Schemes)
	// Pass screen width - selector calculates its own width (50-60%)
	m.selector = NewSelector("Select Scheme", items, m.width, m.styles)
	m.selectorType = SelectorScheme
	m.mode = ModeSelector
}

func (m *Model) openDestinationSelector() {
	// Convert core types to selector types
	sims := make([]SimulatorInfo, len(m.info.Simulators))
	for i, s := range m.info.Simulators {
		sims[i] = SimulatorInfo{
			Name:        s.Name,
			UDID:        s.UDID,
			State:       s.State,
			RuntimeName: s.RuntimeName,
			OSVersion:   s.OSVersion,
			Available:   s.Available,
		}
	}

	devices := make([]DeviceInfo, len(m.info.Devices))
	for i, d := range m.info.Devices {
		devices[i] = DeviceInfo{
			Name:       d.Name,
			Identifier: d.Identifier,
			Platform:   d.Platform,
			OSVersion:  d.OSVersion,
			Model:      d.Model,
		}
	}

	items := DestinationItems(sims, devices)
	if len(items) == 0 {
		m.setStatus("No destinations found")
		return
	}

	// Pass screen width - selector calculates its own width (50-60%)
	m.selector = NewSelector("Select Destination", items, m.width, m.styles)
	m.selectorType = SelectorDestination
	m.mode = ModeSelector
}

func (m *Model) handleSelectorResult(item *SelectorItem) {
	switch m.selectorType {
	case SelectorScheme:
		m.cfg.Scheme = item.ID
		m.setStatus("Scheme: " + item.Title)
		// Save config
		if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err != nil {
			m.lastErr = err.Error()
		}

	case SelectorDestination:
		// Find the destination in our lists
		for _, sim := range m.info.Simulators {
			if sim.UDID == item.ID {
				m.cfg.Destination = core.Destination{
					Kind:     "simulator",
					UDID:     sim.UDID,
					Name:     sim.Name,
					Platform: "iOS Simulator",
					OS:       sim.OSVersion,
				}
				m.setStatus("Destination: " + item.Title)
				if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err != nil {
					m.lastErr = err.Error()
				}
				return
			}
		}
		for _, dev := range m.info.Devices {
			if dev.Identifier == item.ID {
				m.cfg.Destination = core.Destination{
					Kind:     "device",
					UDID:     dev.Identifier,
					Name:     dev.Name,
					Platform: dev.Platform,
					OS:       dev.OSVersion,
				}
				m.setStatus("Destination: " + item.Title)
				if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err != nil {
					m.lastErr = err.Error()
				}
				return
			}
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Model) openPalette() {
	// Pass screen width - palette calculates its own width (50-60%)
	m.palette = NewPalette(m.width, m.styles)
	m.mode = ModePalette
}

func (m *Model) executePaletteCommand(cmd *Command) tea.Cmd {
	switch cmd.ID {
	// Actions
	case "build":
		if !m.running {
			return m.startOp("build")
		}
	case "run":
		if !m.running {
			return m.startOp("run")
		}
	case "test":
		if !m.running {
			return m.startOp("test")
		}
	case "clean":
		if !m.running {
			return m.startOp("clean")
		}
	case "stop":
		m.setStatus("Stop not implemented yet")

	// Archive/Profile (not implemented yet)
	case "archive", "archive-appstore", "archive-adhoc", "profile", "analyze":
		m.setStatus(cmd.Name + " coming soon")

	// Configuration
	case "scheme":
		m.openSchemeSelector()
	case "destination":
		m.openDestinationSelector()
	case "init":
		m.mode = ModeWizard
		m.wizard = newWizard(m.info, m.cfg, m.width)
		return m.wizard.Init()
	case "refresh":
		m.setStatus("Refreshing…")
		return loadContextCmd(m.projectRoot, m.configPath, m.cfgOverride)

	// Utilities
	case "doctor":
		m.setStatus("Doctor not implemented in TUI")
	case "logs":
		m.setStatus("Use CLI: xcbolt logs")
	case "simulator-boot", "simulator-shutdown":
		m.setStatus("Use CLI: xcbolt simulator")

	// Navigation
	case "help":
		m.mode = ModeHelp
	case "quit":
		return tea.Quit
	}

	return nil
}

func (m *Model) updateViewportSize() {
	// Use new layout calculations
	m.viewport.Width = maxInt(0, m.layout.ContentWidth()-2)
	m.viewport.Height = maxInt(0, m.layout.ContentHeight()-2)
}

// syncStatusBarState syncs the status bar display with current model state
func (m *Model) syncStatusBarState() {
	// Project name from workspace or project
	if m.cfg.Workspace != "" {
		m.statusBar.ProjectName = filepath.Base(m.cfg.Workspace)
		// Remove .xcworkspace extension
		m.statusBar.ProjectName = strings.TrimSuffix(m.statusBar.ProjectName, ".xcworkspace")
	} else if m.cfg.Project != "" {
		m.statusBar.ProjectName = filepath.Base(m.cfg.Project)
		// Remove .xcodeproj extension
		m.statusBar.ProjectName = strings.TrimSuffix(m.statusBar.ProjectName, ".xcodeproj")
	}

	m.statusBar.GitBranch = m.gitBranch
	m.statusBar.Scheme = m.cfg.Scheme
	m.statusBar.Destination = m.cfg.Destination.Name
	m.statusBar.DestOS = m.cfg.Destination.OS
	m.statusBar.Running = m.running
	m.statusBar.RunningCmd = m.runningCmd
	m.statusBar.Stage = m.currentStage
	m.statusBar.Progress = m.stageProgress

	// Error/warning counts
	m.statusBar.ErrorCount = m.issuesPanel.ErrorCount()
	m.statusBar.WarningCount = m.issuesPanel.WarningCount()
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// Wizard mode - delegate to wizard
	if m.mode == ModeWizard {
		if keyMatches(msg, m.keys.Quit) {
			return tea.Quit
		}
		var cmd tea.Cmd
		m.wizard, cmd = m.wizard.Update(msg)
		return cmd
	}

	// Help mode - any key closes
	if m.mode == ModeHelp {
		m.mode = ModeNormal
		return nil
	}

	// Selector mode
	if m.mode == ModeSelector {
		var result *SelectorResult
		m.selector, _, result = m.selector.Update(msg)

		if result != nil {
			m.mode = ModeNormal
			if !result.Aborted && result.Selected != nil {
				m.handleSelectorResult(result.Selected)
			}
		}
		return nil
	}

	// Palette mode
	if m.mode == ModePalette {
		var result *PaletteResult
		m.palette, _, result = m.palette.Update(msg)

		if result != nil {
			m.mode = ModeNormal
			if !result.Aborted && result.Command != nil {
				return m.executePaletteCommand(result.Command)
			}
		}
		return nil
	}

	// Issues panel focused - handle navigation
	if m.issuesFocused && m.showIssues {
		switch {
		case keyMatches(msg, m.keys.ScrollUp):
			m.issuesPanel.MoveUp()
			return nil
		case keyMatches(msg, m.keys.ScrollDown):
			m.issuesPanel.MoveDown()
			return nil
		case keyMatches(msg, m.keys.SelectEnter):
			m.issuesPanel.ToggleExpand()
			return nil
		case keyMatches(msg, m.keys.OpenInEditor):
			return m.openIssueInEditor()
		case keyMatches(msg, m.keys.FocusIssues), keyMatches(msg, m.keys.Cancel):
			m.issuesFocused = false
			return nil
		case keyMatches(msg, m.keys.ToggleIssues):
			m.toggleIssuesPanel()
			return nil
		case keyMatches(msg, m.keys.Quit):
			return tea.Quit
		}
		return nil
	}

	// Normal mode
	switch {
	case keyMatches(msg, m.keys.Quit):
		return tea.Quit

	case keyMatches(msg, m.keys.Help):
		m.mode = ModeHelp

	case keyMatches(msg, m.keys.Cancel):
		if m.running && m.cancelFn != nil {
			m.cancelFn()
			m.setStatus("Canceled")
		}

	case keyMatches(msg, m.keys.Build):
		if !m.running {
			return m.startOp("build")
		}

	case keyMatches(msg, m.keys.Run):
		if !m.running {
			return m.startOp("run")
		}

	case keyMatches(msg, m.keys.Test):
		if !m.running {
			return m.startOp("test")
		}

	case keyMatches(msg, m.keys.Clean):
		if !m.running {
			return m.startOp("clean")
		}

	case keyMatches(msg, m.keys.Scheme):
		m.openSchemeSelector()

	case keyMatches(msg, m.keys.Destination):
		m.openDestinationSelector()

	case keyMatches(msg, m.keys.Palette):
		m.openPalette()

	case keyMatches(msg, m.keys.Init):
		m.mode = ModeWizard
		m.wizard = newWizard(m.info, m.cfg, m.width)
		return m.wizard.Init()

	case keyMatches(msg, m.keys.Refresh):
		m.setStatus("Refreshing…")
		return loadContextCmd(m.projectRoot, m.configPath, m.cfgOverride)

	case keyMatches(msg, m.keys.ToggleAutoFollow):
		m.autoFollow = !m.autoFollow
		if m.autoFollow {
			m.viewport.GotoBottom()
			m.setStatus("Following logs")
		} else {
			m.setStatus("Paused log follow")
		}

	case keyMatches(msg, m.keys.ScrollBottom):
		m.viewport.GotoBottom()
		m.autoFollow = true

	case keyMatches(msg, m.keys.Search):
		m.setStatus("Search not implemented yet")

	case keyMatches(msg, m.keys.ToggleIssues):
		m.toggleIssuesPanel()

	case keyMatches(msg, m.keys.FocusIssues):
		if m.showIssues && m.issuesPanel.HasIssues() {
			m.issuesFocused = true
		}
	}

	return nil
}

// toggleIssuesPanel toggles the visibility of the issues panel
func (m *Model) toggleIssuesPanel() {
	if m.issuesPanel.HasIssues() {
		m.showIssues = !m.showIssues
		m.layout.ShowIssuesPanel = m.showIssues
		if m.showIssues {
			m.layout.IssuesPanelHeight = m.layout.CalculateIssuesPanelHeight(len(m.issuesPanel.Issues))
		}
		m.updateViewportSize()
		m.issuesFocused = false
	}
}

// openIssueInEditor opens the selected issue in the user's editor
func (m *Model) openIssueInEditor() tea.Cmd {
	if m.issuesPanel.SelectedIdx >= len(m.issuesPanel.Issues) {
		return nil
	}

	issue := m.issuesPanel.Issues[m.issuesPanel.SelectedIdx]
	if issue.FilePath == "" {
		m.setStatus("No file location")
		return nil
	}

	// Build editor command
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "open" // macOS default
	}

	// Format: editor +line file or open -t file
	var cmd *exec.Cmd
	if editor == "open" {
		cmd = exec.Command("open", "-t", issue.FilePath)
	} else if issue.Line > 0 {
		cmd = exec.Command(editor, fmt.Sprintf("+%d", issue.Line), issue.FilePath)
	} else {
		cmd = exec.Command(editor, issue.FilePath)
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return statusMsg("Failed to open editor")
		}
		return nil
	})
}

// statusMsg is a message for setting status
type statusMsg string

// tryAutoDetect attempts to auto-configure if the project has a single obvious setup.
// Returns true if auto-configuration was applied.
func (m *Model) tryAutoDetect() bool {
	// Need at least one project/workspace
	hasProject := len(m.info.Workspaces) > 0 || len(m.info.Projects) > 0
	if !hasProject {
		return false
	}

	// Need at least one scheme
	if len(m.info.Schemes) == 0 {
		return false
	}

	// Auto-select project: prefer workspace, then project
	if m.cfg.Workspace == "" && m.cfg.Project == "" {
		if len(m.info.Workspaces) == 1 {
			m.cfg.Workspace = m.info.Workspaces[0]
		} else if len(m.info.Projects) == 1 {
			m.cfg.Project = m.info.Projects[0]
		} else if len(m.info.Workspaces) > 0 {
			m.cfg.Workspace = m.info.Workspaces[0]
		} else if len(m.info.Projects) > 0 {
			m.cfg.Project = m.info.Projects[0]
		}
	}

	// Auto-select scheme: prefer first one
	if m.cfg.Scheme == "" && len(m.info.Schemes) > 0 {
		m.cfg.Scheme = m.info.Schemes[0]
	}

	// Auto-select configuration
	if m.cfg.Configuration == "" {
		m.cfg.Configuration = "Debug"
	}

	// Auto-select destination: prefer first booted simulator
	if m.cfg.Destination.UDID == "" && len(m.info.Simulators) > 0 {
		m.cfg.Destination.Kind = core.DestSimulator
		for _, sim := range m.info.Simulators {
			if sim.Available && sim.State == "Booted" {
				m.cfg.Destination.UDID = sim.UDID
				m.cfg.Destination.Name = sim.Name
				m.cfg.Destination.Platform = "iOS Simulator"
				m.cfg.Destination.OS = sim.OSVersion
				break
			}
		}
		// If no booted simulator, use first available
		if m.cfg.Destination.UDID == "" {
			for _, sim := range m.info.Simulators {
				if sim.Available {
					m.cfg.Destination.UDID = sim.UDID
					m.cfg.Destination.Name = sim.Name
					m.cfg.Destination.Platform = "iOS Simulator"
					m.cfg.Destination.OS = sim.OSVersion
					break
				}
			}
		}
	}

	return m.cfg.Scheme != "" && (m.cfg.Workspace != "" || m.cfg.Project != "")
}

// saveRecentCombo saves the current scheme+destination combo to recents
func (m *Model) saveRecentCombo() {
	if m.cfg.Scheme == "" || m.cfg.Destination.UDID == "" {
		return
	}

	combo := core.RecentCombo{
		Scheme:      m.cfg.Scheme,
		Destination: m.cfg.Destination.Name,
		DestUDID:    m.cfg.Destination.UDID,
		DestKind:    string(m.cfg.Destination.Kind),
		UsedAt:      time.Now().Format(time.RFC3339),
	}

	m.state.AddRecentCombo(m.projectRoot, combo)
	// Save state in background (ignore errors)
	go func() {
		_ = core.SaveState(m.state)
	}()
}

func (m *Model) handleEvent(ev core.Event) {
	line := m.formatEventLine(ev)
	m.appendLog(line)

	// Track progress for stage indicators
	m.parseProgressFromEvent(ev)

	// Collect issues from events
	m.collectIssueFromEvent(ev)
}

// collectIssueFromEvent parses errors/warnings from events and adds them to the panel
func (m *Model) collectIssueFromEvent(ev core.Event) {
	switch ev.Type {
	case "error":
		issue := Issue{
			Type:    IssueError,
			Message: ev.Msg,
		}
		if ev.Err != nil {
			issue.Message = ev.Err.Message
			issue.Context = ev.Err.Detail
		}
		// Parse file location from message
		filePath, lineNum, col := parseIssueLocation(ev.Msg)
		issue.FilePath = filePath
		issue.Line = lineNum
		issue.Column = col
		m.issuesPanel.AddIssue(issue)

	case "warning":
		issue := Issue{
			Type:    IssueWarning,
			Message: ev.Msg,
		}
		filePath, lineNum, col := parseIssueLocation(ev.Msg)
		issue.FilePath = filePath
		issue.Line = lineNum
		issue.Column = col
		m.issuesPanel.AddIssue(issue)

	case "log", "log_raw":
		// Also parse from log lines for xcpretty/xcbeautify output
		m.issuesPanel.AddFromLogLine(ev.Msg)
	}

	// Auto-show panel when errors exist
	if m.issuesPanel.ErrorCount() > 0 && !m.showIssues {
		m.showIssues = true
		m.layout.ShowIssuesPanel = true
		m.layout.IssuesPanelHeight = m.layout.CalculateIssuesPanelHeight(len(m.issuesPanel.Issues))
		m.updateViewportSize()
	}
}

func (m *Model) parseProgressFromEvent(ev core.Event) {
	if ev.Type == "log" && isPrettyEvent(ev) {
		return
	}
	msg := ev.Msg

	// Reset progress on new operation
	if strings.Contains(msg, "Starting") || strings.Contains(msg, "Build started") {
		m.currentStage = ""
		m.stageProgress = ""
		m.progressCur = 0
		m.progressTotal = 0
		return
	}

	// Extract stage from common xcodebuild output patterns
	switch {
	case strings.Contains(msg, "Compiling"):
		m.currentStage = "Compiling"
	case strings.Contains(msg, "Linking"):
		m.currentStage = "Linking"
	case strings.Contains(msg, "Signing"):
		m.currentStage = "Signing"
	case strings.Contains(msg, "Processing"):
		m.currentStage = "Processing"
	case strings.Contains(msg, "Copying"):
		m.currentStage = "Copying"
	case strings.Contains(msg, "Running"):
		m.currentStage = "Running"
	case strings.Contains(msg, "Testing"):
		m.currentStage = "Testing"
	case strings.Contains(msg, "Analyzing"):
		m.currentStage = "Analyzing"
	}

	// Try to extract progress numbers (e.g., "47 of 100 tasks")
	// This is a simple heuristic since xcodebuild output varies
	if strings.Contains(msg, " of ") && strings.Contains(msg, "task") {
		// Try to find pattern like "X of Y"
		parts := strings.Split(msg, " of ")
		if len(parts) >= 2 {
			// Extract last number from first part
			words1 := strings.Fields(parts[0])
			if len(words1) > 0 {
				num1 := words1[len(words1)-1]
				// Extract first number from second part
				words2 := strings.Fields(parts[1])
				if len(words2) > 0 {
					num2 := words2[0]
					m.stageProgress = num1 + "/" + num2
					// Parse numbers for progress bar
					fmt.Sscanf(num1, "%d", &m.progressCur)
					fmt.Sscanf(num2, "%d", &m.progressTotal)
				}
			}
		}
	}

	// Update progress bar
	m.progressBar.SetProgress(m.progressCur, m.progressTotal, m.currentStage)
}

func isPrettyEvent(ev core.Event) bool {
	m, ok := ev.Data.(map[string]any)
	if !ok {
		return false
	}
	v, ok := m["pretty"]
	if !ok {
		return false
	}
	pretty, ok := v.(bool)
	return ok && pretty
}

func (m *Model) handleOpDone(msg opDoneMsg) {
	m.running = false
	m.runningCmd = ""
	m.cancelFn = nil
	m.eventCh = nil
	m.doneCh = nil
	m.currentStage = ""
	m.stageProgress = ""
	m.progressCur = 0
	m.progressTotal = 0

	// Hide progress bar
	m.progressBar.Hide()
	m.layout.ShowProgressBar = false

	if msg.cfg.Version != 0 {
		m.cfg = msg.cfg
	}

	success := msg.err == nil

	// Auto-hide issues panel on success with no errors
	if success && m.issuesPanel.ErrorCount() == 0 {
		m.showIssues = false
		m.layout.ShowIssuesPanel = false
		m.updateViewportSize()
	}

	// Track result duration for status bar
	var duration time.Duration

	if msg.build != nil {
		m.lastBuild = *msg.build
		m.lastResult = &Result{
			Operation: "Build",
			Success:   success,
			Duration:  msg.build.Duration,
			Timestamp: time.Now(),
		}
		duration = msg.build.Duration
	}
	if msg.run != nil {
		m.lastRun = *msg.run
		m.lastResult = &Result{
			Operation: "Run",
			Success:   success,
			Message:   fmt.Sprintf("PID %d", msg.run.PID),
			Timestamp: time.Now(),
		}
	}
	if msg.test != nil {
		m.lastTest = *msg.test
		m.lastResult = &Result{
			Operation: "Test",
			Success:   success,
			Duration:  msg.test.Duration,
			Timestamp: time.Now(),
		}
		duration = msg.test.Duration
	}

	// Update status bar with last result
	m.statusBar.HasLastResult = true
	m.statusBar.LastResultSuccess = success
	m.statusBar.LastResultOp = msg.cmd
	if duration > 0 {
		m.statusBar.LastResultTime = duration.Round(100 * time.Millisecond).String()
	} else {
		m.statusBar.LastResultTime = ""
	}

	// Append result line to logs
	resultLine := m.formatResultLine(msg.cmd, success, duration)
	m.appendLog(resultLine)

	if msg.err != nil {
		m.lastErr = msg.err.Error()
		m.setStatus(strings.ToUpper(msg.cmd) + " failed")
	} else {
		m.setStatus(strings.ToUpper(msg.cmd) + " done")
	}
}

func (m *Model) startOp(name string) tea.Cmd {
	m.running = true
	m.runningCmd = name

	// Update progress bar
	m.progressBar.Visible = true
	m.progressBar.Stage = "Starting..."
	m.progressBar.Current = 0
	m.progressBar.Total = 0
	m.layout.ShowProgressBar = true

	// Clear issues panel for new operation
	m.issuesPanel.Clear()
	m.showIssues = false
	m.layout.ShowIssuesPanel = false
	m.updateViewportSize()

	m.appendLog("─────────────────────────────────────────")
	m.appendLog(fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), strings.ToUpper(name)))

	// Save this scheme+destination combo to recents
	m.saveRecentCombo()

	events := make(chan core.Event, 256)
	done := make(chan opDoneMsg, 1)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFn = cancel
	m.eventCh = events
	m.doneCh = done

	emitter := &chanEmitter{ch: events}

	cfg := m.cfg
	root := m.projectRoot

	go func() {
		defer close(events)
		switch name {
		case "build":
			res, cfg2, err := core.Build(ctx, root, cfg, emitter)
			done <- opDoneMsg{cmd: name, err: err, cfg: cfg2, build: &res}
		case "run":
			res, cfg2, err := core.Run(ctx, root, cfg, false, emitter)
			done <- opDoneMsg{cmd: name, err: err, cfg: cfg2, run: &res}
		case "test":
			res, cfg2, err := core.Test(ctx, root, cfg, nil, nil, emitter)
			done <- opDoneMsg{cmd: name, err: err, cfg: cfg2, test: &res}
		case "clean":
			// Clean derived data and results
			paths := []string{
				filepath.Join(root, ".xcbolt", "DerivedData"),
				filepath.Join(root, ".xcbolt", "Results"),
			}
			var cleanErr error
			for _, p := range paths {
				if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
					cleanErr = err
				}
			}
			done <- opDoneMsg{cmd: name, err: cleanErr}
		default:
			done <- opDoneMsg{cmd: name, err: fmt.Errorf("unknown op %s", name)}
		}
		close(done)
	}()
	return tea.Batch(waitForEvent(events), waitForDone(done), tickCmd())
}

type chanEmitter struct {
	ch chan<- core.Event
}

func (e *chanEmitter) Emit(ev core.Event) {
	select {
	case e.ch <- ev:
	default:
	}
}

func waitForEvent(ch <-chan core.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg(ev)
	}
}

func waitForDone(ch <-chan opDoneMsg) tea.Cmd {
	return func() tea.Msg {
		m, ok := <-ch
		if !ok {
			return nil
		}
		return m
	}
}

func (m *Model) appendLog(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > 2000 {
		m.logLines = m.logLines[len(m.logLines)-2000:]
	}
}

func (m *Model) formatEventLine(ev core.Event) string {
	icons := m.styles.Icons
	prefix := ""

	switch ev.Type {
	case "error":
		prefix = m.styles.StatusStyle("error").Render(icons.Error) + " "
	case "warning":
		prefix = m.styles.StatusStyle("warning").Render(icons.Warning) + " "
	case "result":
		prefix = m.styles.StatusStyle("success").Render(icons.Success) + " "
	case "status":
		prefix = m.styles.StatusStyle("running").Render(icons.ChevronRight) + " "
	}

	msg := ev.Msg
	if msg == "" && ev.Err != nil {
		msg = ev.Err.Message
	}
	if msg == "" {
		msg = ev.Type
	}
	return prefix + msg
}

func (m *Model) formatResultLine(op string, success bool, duration time.Duration) string {
	icons := m.styles.Icons

	icon := icons.Success
	status := "success"
	verb := "Succeeded"
	if !success {
		icon = icons.Error
		status = "error"
		verb = "Failed"
	}

	iconStyled := m.styles.StatusStyle(status).Render(icon)
	text := strings.ToUpper(op[:1]) + op[1:] + " " + verb
	if duration > 0 {
		text += " · " + duration.Round(100*time.Millisecond).String()
	}

	return iconStyled + " " + text
}

// =============================================================================
// View
// =============================================================================

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Help overlay mode
	if m.mode == ModeHelp {
		return m.helpOverlayView()
	}

	// Wizard mode
	if m.mode == ModeWizard {
		return m.wizardView()
	}

	// Selector mode - show selector overlay on top of main view
	if m.mode == ModeSelector {
		return m.selectorOverlayView()
	}

	// Palette mode - show palette overlay
	if m.mode == ModePalette {
		return m.paletteOverlayView()
	}

	// Normal mode - main layout
	return m.mainView()
}

func (m Model) mainView() string {
	// Sync state to components
	m.syncStatusBarState()

	// Build status bar content
	statusBarContent := m.statusBar.View(m.width, m.styles)

	// Build progress bar content (if running)
	progressBarContent := ""
	if m.running {
		m.layout.ShowProgressBar = true
		progressBarContent = m.progressBar.View(m.width, m.styles)
	} else {
		m.layout.ShowProgressBar = false
	}

	// Build main content (logs)
	contentArea := m.contentView()

	// Build issues panel content (if has issues)
	issuesPanelContent := ""
	if m.showIssues && m.issuesPanel.HasIssues() {
		m.issuesPanel.Width = m.width
		m.issuesPanel.Height = m.layout.IssuesPanelHeight
		issuesPanelContent = m.issuesPanel.View(m.styles)
	}

	// Build context-aware hints bar
	hintsBarContent := m.hintsBar.ViewWithContext(m.width, m.styles, m.issuesFocused, m.issuesPanel.HasIssues())

	// Use layout to render everything
	return m.layout.RenderFullLayout(
		statusBarContent,
		progressBarContent,
		contentArea,
		issuesPanelContent,
		hintsBarContent,
		m.styles,
	)
}

// contentView renders the main content area (logs)
func (m Model) contentView() string {
	if len(m.logLines) == 0 {
		return m.emptyStateView()
	}

	return m.viewport.View()
}

// emptyStateView renders the empty state with icon + message + hint
func (m Model) emptyStateView() string {
	s := m.styles
	icons := s.Icons

	var lines []string

	// Check if we have a valid configuration
	isConfigured := m.cfg.Scheme != "" && (m.cfg.Workspace != "" || m.cfg.Project != "")

	iconStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextSubtle).
		MarginBottom(1)

	msgStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextMuted).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextSubtle)

	if isConfigured {
		// Ready to work
		lines = append(lines, iconStyle.Render(icons.Idle))
		lines = append(lines, msgStyle.Render("Ready to build"))
		lines = append(lines, hintStyle.Render("r run  b build  t test"))
	} else if m.info.Schemes != nil && len(m.info.Schemes) > 0 {
		// Context loaded but not configured
		lines = append(lines, iconStyle.Render(icons.Settings))
		lines = append(lines, msgStyle.Render("Press i to configure"))
		lines = append(lines, hintStyle.Render("s scheme  d destination  ? help"))
	} else {
		// Context loading or no project detected
		lines = append(lines, iconStyle.Render(m.spinner.View()))
		lines = append(lines, msgStyle.Render("Loading project..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, lines...)

	// Center in the available space
	centered := lipgloss.Place(
		m.layout.ContentWidth(),
		m.layout.ContentHeight(),
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	return centered
}

func (m Model) helpOverlayView() string {
	s := m.styles

	// Calculate width: 50-60% of screen, clamped
	width := m.width * 55 / 100
	if width < 50 {
		width = 50
	}
	if width > 80 {
		width = 80
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(s.Colors.Text)
	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n")

	dividerStyle := lipgloss.NewStyle().Foreground(s.Colors.BorderMuted)
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")

	groups := m.keys.FullHelp()
	groupNames := []string{"ACTIONS", "CONFIGURATION", "LAYOUT", "SCROLLING", "OTHER"}

	sectionStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextSubtle).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Accent).
		Width(12)

	descStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Text)

	for i, group := range groups {
		if i < len(groupNames) {
			b.WriteString(sectionStyle.Render("  " + groupNames[i]))
			b.WriteString("\n")
		}
		for _, binding := range group {
			keys := binding.Help().Key
			desc := binding.Help().Desc
			// Format: key (padded) description
			keyPart := keyStyle.Render("  " + keys)
			descPart := descStyle.Render(desc)
			b.WriteString(keyPart + descPart + "\n")
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	hintKeyStyle := lipgloss.NewStyle().Foreground(s.Colors.Accent)
	hintDescStyle := lipgloss.NewStyle().Foreground(s.Colors.TextSubtle)
	hints := "Press " + hintKeyStyle.Render("?") + hintDescStyle.Render(" or ") +
		hintKeyStyle.Render("esc") + hintDescStyle.Render(" to close")
	b.WriteString(hints)

	// Container with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Colors.Border).
		Padding(1, 2)

	helpContent := containerStyle.Width(width).Render(b.String())

	// Center the help overlay
	overlay := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		helpContent,
	)

	return overlay
}

func (m Model) wizardView() string {
	s := m.styles
	statusBarContent := m.statusBar.View(m.width, m.styles)
	return lipgloss.JoinVertical(lipgloss.Left,
		m.layout.RenderStatusBar(statusBarContent, m.styles),
		s.Popup.Container.Width(m.width-4).Render(m.wizard.View()),
	)
}

func (m Model) selectorOverlayView() string {
	// Render selector popup centered on screen
	selectorContent := m.selector.View()
	return RenderCenteredPopup(selectorContent, m.width, m.height)
}

func (m Model) paletteOverlayView() string {
	// Render palette popup centered on screen
	paletteContent := m.palette.View()
	return RenderPaletteCentered(paletteContent, m.width, m.height)
}

// =============================================================================
// Git Helpers
// =============================================================================

// getGitBranch returns the current git branch name, or empty string if not in a git repo
func getGitBranch(projectRoot string) string {
	cmd := exec.Command("git", "-C", projectRoot, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// =============================================================================
// Helpers
// =============================================================================

func keyMatches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
