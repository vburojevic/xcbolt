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
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xcbolt/xcbolt/internal/core"
)

// =============================================================================
// Run Mode Split View Types
// =============================================================================

// Pane identifies which pane has focus in split view
type Pane int

const (
	PaneBuild   Pane = iota // Build logs pane (top)
	PaneConsole             // Console output pane (bottom)
)

// RunModeState tracks the state for run mode split view
type RunModeState struct {
	Active      bool     // Whether run mode split view is active
	ConsoleLogs []string // App console output (separate from build logs)
	FocusPane   Pane     // Which pane has focus
	ConsolePos  int      // Scroll position for console pane
}

// LogViewMode controls the main log presentation.
type LogViewMode int

const (
	LogViewCards LogViewMode = iota
	LogViewStream
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
	state       core.State // User state (recents, favorites)
	gitBranch   string     // Current git branch

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
	spinner      spinner.Model
	viewport     viewport.Model
	helpViewport viewport.Model // For scrollable help overlay
	wizard       wizardModel
	selector     SelectorModel
	palette      PaletteModel

	// Tab-based log view (replaces phaseView and streamView)
	tabView *TabView

	// Legacy views (kept for gradual migration)
	streamView  StreamView
	phaseView   PhaseView
	logViewMode LogViewMode

	// Status message (shown in results bar)
	statusMsg string

	// Search state
	searchInput   textinput.Model
	searchQuery   string
	searchMatches []SearchMatch
	searchCursor  int
	searchActive  bool

	// Operation state
	running    bool
	runningCmd string
	cancelFn   context.CancelFunc
	eventCh    <-chan core.Event
	doneCh     <-chan opDoneMsg
	tickCount  int // For spinner animation timing

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

	// Run mode split view
	runMode RunModeState
}

// NewModel creates a new TUI model
func NewModel(projectRoot string, configPath string, overrides ConfigOverrides) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	helpVp := viewport.New(0, 0)
	helpVp.YPosition = 0

	h := help.New()
	h.ShowAll = false

	// Load user state (ignore errors - use defaults if not found)
	state, _ := core.LoadState()

	// Initialize layout components
	layout := NewLayout()
	statusBar := NewStatusBar()
	progressBar := NewProgressBar()
	hintsBar := NewHintsBar()

	// Initialize search input
	si := textinput.New()
	si.Placeholder = "Search logs..."
	si.CharLimit = 100
	si.Width = 40

	return Model{
		projectRoot:  projectRoot,
		configPath:   configPath,
		cfgOverride:  overrides,
		styles:       DefaultStyles(),
		keys:         defaultKeyMap(),
		help:         h,
		spinner:      sp,
		viewport:     vp,
		helpViewport: helpVp,
		tabView:      NewTabView(),
		phaseView:   NewPhaseView(),
		streamView:  NewStreamView(),
		searchInput: si,
		mode:        ModeNormal,
		logViewMode: LogViewCards,
		state:       state,
		// Layout components
		layout:      layout,
		statusBar:   statusBar,
		progressBar: progressBar,
		hintsBar:    hintsBar,
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
		tickCmd(), // Start tick for loading spinner animation
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
			// Still mark as loaded (with error) so we don't show endless spinner
			m.tabView.SummaryTab.SetContextLoaded(true)
			break
		}
		m.info = msg.info
		m.cfg = msg.cfg
		m.applyTUIConfig()

		// Mark context as loaded for Dashboard
		m.tabView.SummaryTab.SetContextLoaded(true)

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
		// Continue ticking if we need animation (spinner while running or loading)
		needsAnimation := m.running || !m.tabView.SummaryTab.ContextLoaded
		if needsAnimation {
			cmds = append(cmds, tickCmd())
			// Advance Dashboard spinner (~4 times per second)
			m.tickCount++
			if m.tickCount%15 == 0 {
				m.tabView.SummaryTab.AdvanceSpinner()
			}
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
		m.applyTUIConfig()
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
		// PhaseView handles its own auto-scroll
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

	// PhaseView scrolling is handled through key bindings in handleNormalModeKey

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
		m.setupHelpViewport()
	case "quit":
		return tea.Quit
	}

	return nil
}

func (m *Model) updateViewportSize() {
	// Use new layout calculations
	m.viewport.Width = maxInt(0, m.layout.ContentWidth()-2)
	m.viewport.Height = maxInt(0, m.layout.ContentHeight()-2)
	// Update TabView dimensions (includes tab bar height internally)
	m.tabView.SetSize(m.layout.ContentWidth(), m.layout.ContentHeight())
	// Update legacy views
	m.phaseView.SetSize(m.layout.ContentWidth(), m.layout.ContentHeight())
	m.streamView.SetSize(m.layout.ContentWidth(), m.layout.ContentHeight())
}

func (m *Model) setupHelpViewport() {
	s := m.styles

	// Calculate width: 50-60% of screen, clamped
	width := m.width * 55 / 100
	if width < 50 {
		width = 50
	}
	if width > 80 {
		width = 80
	}

	// Calculate max height for content (leave room for border, padding, title, footer)
	maxHeight := m.height - 10
	if maxHeight < 10 {
		maxHeight = 10
	}

	var b strings.Builder

	groups := m.keys.FullHelp()
	groupNames := []string{"ACTIONS", "CONFIGURATION", "TABS", "VIEW", "SCROLLING", "NAVIGATION"}

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
			keyPart := keyStyle.Render("  " + keys)
			descPart := descStyle.Render(desc)
			b.WriteString(keyPart + descPart + "\n")
		}
		b.WriteString("\n")
	}

	content := b.String()

	// Set up viewport
	m.helpViewport.Width = width - 6 // Account for border and padding
	m.helpViewport.Height = maxHeight
	m.helpViewport.SetContent(content)
	m.helpViewport.GotoTop()
}

func (m *Model) applyTUIConfig() {
	m.phaseView.SmartCollapse = !m.cfg.TUI.ShowAllLogs
	if m.cfg.TUI.ShowAllLogs {
		m.phaseView.ExpandAll()
	}
}

func (m *Model) toggleLogView() {
	if m.logViewMode == LogViewCards {
		m.logViewMode = LogViewStream
		m.setStatus("Stream view")
	} else {
		m.logViewMode = LogViewCards
		m.setStatus("Phase cards")
	}
}

// syncStatusBarState syncs the status bar display with current model state
func (m *Model) syncStatusBarState() {
	// Project name from workspace or project
	projectName := ""
	if m.cfg.Workspace != "" {
		projectName = filepath.Base(m.cfg.Workspace)
		// Remove .xcworkspace extension
		projectName = strings.TrimSuffix(projectName, ".xcworkspace")
		m.statusBar.ProjectName = projectName
	} else if m.cfg.Project != "" {
		projectName = filepath.Base(m.cfg.Project)
		// Remove .xcodeproj extension
		projectName = strings.TrimSuffix(projectName, ".xcodeproj")
		m.statusBar.ProjectName = projectName
	}

	m.statusBar.GitBranch = m.gitBranch
	m.statusBar.Scheme = m.cfg.Scheme
	m.statusBar.Destination = m.cfg.Destination.Name
	m.statusBar.DestOS = m.cfg.Destination.OS
	m.statusBar.Running = m.running
	m.statusBar.RunningCmd = m.runningCmd
	m.statusBar.Stage = m.currentStage
	m.statusBar.Progress = m.stageProgress

	// Error/warning counts from TabView
	m.statusBar.ErrorCount = m.tabView.Counts.ErrorCount
	m.statusBar.WarningCount = m.tabView.Counts.WarningCount

	// Sync project info to Dashboard
	targetDevice := m.cfg.Destination.Name
	if m.cfg.Destination.OS != "" {
		targetDevice += " (" + m.cfg.Destination.OS + ")"
	}
	m.tabView.SummaryTab.SetProjectInfo(
		projectName,
		m.cfg.Scheme,
		targetDevice,
		m.cfg.Configuration,
	)

	// Sync system info to Dashboard
	simulatorStatus := ""
	for _, sim := range m.info.Simulators {
		if sim.State == "Booted" {
			simulatorStatus = "Booted"
			break
		}
	}
	if simulatorStatus == "" && len(m.info.Simulators) > 0 {
		simulatorStatus = "Available"
	}
	deviceConnected := len(m.info.Devices) > 0
	m.tabView.SummaryTab.SetSystemInfo("Xcode", simulatorStatus, deviceConnected)
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

	// Help mode - handle scrolling or close
	if m.mode == ModeHelp {
		switch msg.String() {
		case "esc", "?", "q":
			m.mode = ModeNormal
			return nil
		case "up", "k":
			m.helpViewport.LineUp(1)
			return nil
		case "down", "j":
			m.helpViewport.LineDown(1)
			return nil
		case "pgup", "ctrl+u":
			m.helpViewport.HalfViewUp()
			return nil
		case "pgdown", "ctrl+d":
			m.helpViewport.HalfViewDown()
			return nil
		case "home", "g":
			m.helpViewport.GotoTop()
			return nil
		case "end", "G":
			m.helpViewport.GotoBottom()
			return nil
		default:
			// Ignore other keys
			return nil
		}
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

	// Search mode
	if m.mode == ModeSearch {
		switch msg.String() {
		case "esc":
			m.exitSearchMode(true) // Esc clears search
		case "enter":
			m.exitSearchMode(false) // Enter keeps search results
		case "ctrl+n", "down":
			m.nextSearchMatch()
		case "ctrl+p", "up":
			m.prevSearchMatch()
		default:
			// Update search input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			// Execute search on each keystroke
			m.executeSearch()
			return cmd
		}
		return nil
	}

	// Normal mode
	switch {
	case keyMatches(msg, m.keys.Quit):
		return tea.Quit

	case keyMatches(msg, m.keys.Help):
		m.mode = ModeHelp
		m.setupHelpViewport()

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

	// Tab navigation
	case keyMatches(msg, m.keys.Tab1):
		m.tabView.SetActiveTab(TabDashboard)
		m.setStatus("Dashboard")

	case keyMatches(msg, m.keys.Tab2):
		m.tabView.SetActiveTab(TabStream)
		m.setStatus("Stream")

	case keyMatches(msg, m.keys.Tab3):
		m.tabView.SetActiveTab(TabIssues)
		m.setStatus("Issues")

	case keyMatches(msg, m.keys.TabNext):
		if !m.runMode.Active { // Don't conflict with SwitchPane in run mode
			m.tabView.NextTab()
			m.setStatus(m.tabView.ActiveTab.String())
		}

	// Display toggles
	case keyMatches(msg, m.keys.ToggleLineNumbers):
		m.tabView.StreamTab.ShowLineNumbers = !m.tabView.StreamTab.ShowLineNumbers
		if m.tabView.StreamTab.ShowLineNumbers {
			m.setStatus("Line numbers on")
		} else {
			m.setStatus("Line numbers off")
		}

	case keyMatches(msg, m.keys.ToggleTimestamps):
		m.tabView.StreamTab.ShowTimestamps = !m.tabView.StreamTab.ShowTimestamps
		if m.tabView.StreamTab.ShowTimestamps {
			m.setStatus("Timestamps on")
		} else {
			m.setStatus("Timestamps off")
		}

	// Copy functionality
	case keyMatches(msg, m.keys.CopyLine):
		return m.copyCurrentLine()

	case keyMatches(msg, m.keys.CopyVisible):
		return m.copyVisibleContent()

	case keyMatches(msg, m.keys.ToggleAutoFollow), keyMatches(msg, m.keys.ToggleRawView):
		m.toggleLogView()

	case keyMatches(msg, m.keys.ToggleErrorsOnly):
		m.phaseView.ShowErrorsOnly = !m.phaseView.ShowErrorsOnly
		if m.phaseView.ShowErrorsOnly {
			m.logViewMode = LogViewCards
			m.phaseView.ShowRawMode = false
			m.phaseView.GotoTop()
			m.setStatus("Errors/warnings only")
		} else {
			m.setStatus("All logs")
		}

	// Scrolling - route to TabView or console pane
	case keyMatches(msg, m.keys.ScrollUp):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(-1)
		} else {
			m.tabView.ScrollUp(1)
		}

	case keyMatches(msg, m.keys.ScrollDown):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(1)
		} else {
			m.tabView.ScrollDown(1)
		}

	case keyMatches(msg, m.keys.ScrollTop):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.runMode.ConsolePos = 0
		} else {
			m.tabView.GotoTop()
		}

	case keyMatches(msg, m.keys.ScrollBottom):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			maxPos := len(m.runMode.ConsoleLogs) - m.layout.SplitBottomHeight()
			if maxPos < 0 {
				maxPos = 0
			}
			m.runMode.ConsolePos = maxPos
		} else {
			m.tabView.GotoBottom()
		}

	case keyMatches(msg, m.keys.PageUp):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(-m.layout.SplitBottomHeight())
		} else {
			m.tabView.ScrollUp(10)
		}

	case keyMatches(msg, m.keys.PageDown):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(m.layout.SplitBottomHeight())
		} else {
			m.tabView.ScrollDown(10)
		}

	case keyMatches(msg, m.keys.HalfPageUp):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(-m.layout.SplitBottomHeight() / 2)
		} else {
			m.tabView.ScrollUp(5)
		}

	case keyMatches(msg, m.keys.HalfPageDown):
		if m.runMode.Active && m.runMode.FocusPane == PaneConsole {
			m.scrollConsole(m.layout.SplitBottomHeight() / 2)
		} else {
			m.tabView.ScrollDown(5)
		}

	// Run mode pane switching
	case keyMatches(msg, m.keys.SwitchPane):
		if m.runMode.Active {
			if m.runMode.FocusPane == PaneBuild {
				m.runMode.FocusPane = PaneConsole
				m.setStatus("Console pane")
			} else {
				m.runMode.FocusPane = PaneBuild
				m.setStatus("Build pane")
			}
		}

	// Phase controls
	case keyMatches(msg, m.keys.ToggleCollapse):
		m.phaseView.ToggleSelectedPhase()

	case keyMatches(msg, m.keys.ExpandAll):
		m.phaseView.ExpandAll()
		m.setStatus("Expanded all phases")

	case keyMatches(msg, m.keys.CollapseAll):
		m.phaseView.CollapseAll()
		m.setStatus("Collapsed all phases")

	// Error navigation
	case keyMatches(msg, m.keys.NextError):
		if p, l := m.phaseView.FindNextError(-1, -1); p >= 0 {
			m.setStatus("Next error")
			_ = l // Line index available for future use
		} else {
			m.setStatus("No errors found")
		}

	case keyMatches(msg, m.keys.PrevError):
		if p, l := m.phaseView.FindPrevError(len(m.phaseView.Phases), 0); p >= 0 {
			m.setStatus("Previous error")
			_ = l // Line index available for future use
		} else {
			m.setStatus("No errors found")
		}

	// Open in editor
	case keyMatches(msg, m.keys.OpenXcode):
		return m.openInXcode()

	case keyMatches(msg, m.keys.OpenEditor):
		return m.openInEditor()

	case keyMatches(msg, m.keys.Search):
		m.enterSearchMode()
	}

	return nil
}

// openInXcode opens the project in Xcode
func (m *Model) openInXcode() tea.Cmd {
	// Prefer workspace over project
	var path string
	if m.cfg.Workspace != "" {
		path = filepath.Join(m.projectRoot, m.cfg.Workspace)
	} else if m.cfg.Project != "" {
		path = filepath.Join(m.projectRoot, m.cfg.Project)
	} else {
		m.setStatus("No project configured")
		return nil
	}

	return func() tea.Msg {
		cmd := exec.Command("open", "-a", "Xcode", path)
		if err := cmd.Start(); err != nil {
			return statusMsg("Failed to open Xcode")
		}
		return statusMsg("Opened in Xcode")
	}
}

// openInEditor opens the project in $EDITOR
func (m *Model) openInEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "code" // Fall back to VS Code
	}

	return func() tea.Msg {
		cmd := exec.Command(editor, m.projectRoot)
		if err := cmd.Start(); err != nil {
			return statusMsg("Failed to open editor: " + err.Error())
		}
		return statusMsg("Opened in " + editor)
	}
}

// enterSearchMode enters search mode
func (m *Model) enterSearchMode() {
	m.mode = ModeSearch
	m.searchInput.Reset()
	m.searchInput.Focus()
	m.searchActive = true
}

// exitSearchMode exits search mode and optionally clears search
func (m *Model) exitSearchMode(clearSearch bool) {
	m.mode = ModeNormal
	m.searchInput.Blur()
	m.searchActive = false
	if clearSearch {
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchCursor = 0
		m.phaseView.ClearSearch()
	}
}

// executeSearch performs the search
func (m *Model) executeSearch() {
	query := m.searchInput.Value()
	if query == "" {
		m.searchMatches = nil
		m.searchCursor = 0
		m.phaseView.ClearSearch()
		return
	}

	m.searchQuery = query
	m.searchMatches = m.phaseView.Search(query)
	m.searchCursor = 0

	if len(m.searchMatches) > 0 {
		m.setStatus(fmt.Sprintf("%d matches", len(m.searchMatches)))
		m.jumpToSearchMatch(0)
	} else {
		m.setStatus("No matches found")
	}
}

// jumpToSearchMatch jumps to a specific search match
func (m *Model) jumpToSearchMatch(idx int) {
	if idx < 0 || idx >= len(m.searchMatches) {
		return
	}
	m.searchCursor = idx
	match := m.searchMatches[idx]
	m.phaseView.JumpToMatch(match.Phase, match.Line)
}

// nextSearchMatch moves to the next search match
func (m *Model) nextSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCursor = (m.searchCursor + 1) % len(m.searchMatches)
	m.jumpToSearchMatch(m.searchCursor)
	m.setStatus(fmt.Sprintf("%d/%d", m.searchCursor+1, len(m.searchMatches)))
}

// prevSearchMatch moves to the previous search match
func (m *Model) prevSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCursor--
	if m.searchCursor < 0 {
		m.searchCursor = len(m.searchMatches) - 1
	}
	m.jumpToSearchMatch(m.searchCursor)
	m.setStatus(fmt.Sprintf("%d/%d", m.searchCursor+1, len(m.searchMatches)))
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

	// Route to TabView (new tab-based system)
	switch {
	case ev.Type == "log_raw":
		m.tabView.AddRawLine(ev.Msg)
	case ev.Type == "log":
		m.tabView.AddRawLine(ev.Msg)
	default:
		m.tabView.AddRawLine(line)
	}

	// Stream view always tracks raw or pretty output (legacy).
	switch {
	case ev.Type == "log_raw":
		m.streamView.AddRawLine(ev.Msg)
	case isPrettyEvent(ev):
		m.streamView.AddPrettyLine(ev.Msg)
	case ev.Type == "log":
		m.streamView.AddRawLine(ev.Msg)
	default:
		m.streamView.AddRawLine(line)
	}

	// Phase view should use raw output for structure and ignore pretty lines (legacy).
	appendPhase := false
	phaseLine := line
	switch {
	case ev.Type == "log_raw":
		appendPhase = true
		phaseLine = ev.Msg
	case ev.Type == "log" && !isPrettyEvent(ev):
		appendPhase = true
		phaseLine = ev.Msg
	case ev.Type != "log" && ev.Type != "log_raw":
		appendPhase = true
		phaseLine = line
	}

	if appendPhase {
		// In run mode, detect console output (after app launches).
		if m.runMode.Active && m.runningCmd == "run" && ev.Type == "log" && m.currentStage == "Running" {
			m.appendConsoleLog(phaseLine)
		} else {
			m.appendLog(phaseLine)
		}
	}

	// Track progress for stage indicators
	m.parseProgressFromEvent(ev)
	// Error/warning counts are tracked by PhaseView through categorizeLogLine

	// Track errors/warnings count for Dashboard
	if ev.Type == "error" || strings.Contains(strings.ToLower(ev.Msg), "error:") {
		m.tabView.SummaryTab.IncrementErrors()
	} else if strings.Contains(strings.ToLower(ev.Msg), "warning:") {
		m.tabView.SummaryTab.IncrementWarnings()
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
		m.currentStage = "Compile"
	case strings.Contains(msg, "Linking"):
		m.currentStage = "Link"
	case strings.Contains(msg, "Signing") || strings.Contains(msg, "CodeSign"):
		m.currentStage = "Sign"
	case strings.Contains(msg, "Processing"):
		m.currentStage = "Compile"
	case strings.Contains(msg, "Copying"):
		m.currentStage = "Link"
	case strings.Contains(msg, "Running"):
		m.currentStage = "Running"
	case strings.Contains(msg, "Testing"):
		m.currentStage = "Testing"
	case strings.Contains(msg, "Analyzing"):
		m.currentStage = "Analyzing"
	case strings.Contains(msg, "Resolving") || strings.Contains(msg, "Resolved"):
		m.currentStage = "Resolve"
	}

	// Extract current file being processed
	currentFile := ""
	if strings.Contains(msg, ".swift") || strings.Contains(msg, ".m") || strings.Contains(msg, ".mm") {
		// Try to extract filename from the line
		words := strings.Fields(msg)
		for _, word := range words {
			if strings.HasSuffix(word, ".swift") || strings.HasSuffix(word, ".m") || strings.HasSuffix(word, ".mm") {
				currentFile = word
				break
			}
		}
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

	// Update Dashboard with live progress
	m.tabView.SummaryTab.UpdateProgress(currentFile, m.progressCur, m.progressTotal, m.currentStage)
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

	// Mark build complete in PhaseView (triggers smart collapse)
	m.phaseView.MarkBuildComplete(success)

	// Track result duration for status bar
	var duration time.Duration
	var durationStr string

	if msg.build != nil {
		m.lastBuild = *msg.build
		m.lastResult = &Result{
			Operation: "Build",
			Success:   success,
			Duration:  msg.build.Duration,
			Timestamp: time.Now(),
		}
		duration = msg.build.Duration
		durationStr = duration.Round(100 * time.Millisecond).String()
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
		durationStr = duration.Round(100 * time.Millisecond).String()
	}

	// Update TabView summary with build results
	m.tabView.SetBuildResult(success, durationStr, nil)

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
	m.appendStreamLine(resultLine)

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

	// Activate run mode split view for "run" command
	if name == "run" {
		m.runMode.Active = true
		m.runMode.ConsoleLogs = nil
		m.runMode.FocusPane = PaneBuild
		m.runMode.ConsolePos = 0
	} else {
		m.runMode.Active = false
	}

	// Clear logs for new operation
	m.tabView.Clear()
	m.phaseView.Clear()
	m.streamView.Clear()

	// Initialize Dashboard for live activity
	m.tabView.SummaryTab.SetRunning(name)

	m.appendLog("─────────────────────────────────────────")
	m.appendLog(fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), strings.ToUpper(name)))
	m.appendStreamLine("─────────────────────────────────────────")
	m.appendStreamLine(fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), strings.ToUpper(name)))

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
	m.phaseView.AddLine(line)
}

func (m *Model) appendStreamLine(line string) {
	m.streamView.AddRawLine(line)
}

func (m *Model) appendConsoleLog(line string) {
	m.runMode.ConsoleLogs = append(m.runMode.ConsoleLogs, line)
	// Auto-scroll if at bottom
	maxPos := len(m.runMode.ConsoleLogs) - m.layout.SplitBottomHeight()
	if m.runMode.ConsolePos >= maxPos-1 {
		m.runMode.ConsolePos = maxPos
	}
}

// scrollConsole scrolls the console pane by delta lines
func (m *Model) scrollConsole(delta int) {
	m.runMode.ConsolePos += delta

	// Clamp to valid range
	maxPos := len(m.runMode.ConsoleLogs) - m.layout.SplitBottomHeight()
	if maxPos < 0 {
		maxPos = 0
	}
	if m.runMode.ConsolePos < 0 {
		m.runMode.ConsolePos = 0
	}
	if m.runMode.ConsolePos > maxPos {
		m.runMode.ConsolePos = maxPos
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

	// Search mode - show main view with search bar
	if m.mode == ModeSearch {
		return m.searchView()
	}

	// Normal mode - main layout
	return m.mainView()
}

func (m Model) mainView() string {
	// Sync state to components
	m.syncStatusBarState()

	// Build status bar content (use minimal mode for small terminals)
	statusBarContent := m.statusBar.ViewWithMinimal(m.width, m.styles, m.layout.MinimalMode)

	// Build progress bar content (if running)
	progressBarContent := ""
	if m.running {
		m.layout.ShowProgressBar = true
		progressBarContent = m.progressBar.View(m.width, m.styles)
	} else {
		m.layout.ShowProgressBar = false
	}

	// Build hints bar
	hintsBarContent := m.hintsBar.View(m.width, m.styles)

	// Use split view for run mode
	if m.runMode.Active {
		topContent := m.contentView()
		bottomContent := m.consoleView()
		topFocused := m.runMode.FocusPane == PaneBuild

		// Add tab hint for run mode
		hintsBarContent = m.runModeHintsBar()

		return m.layout.RenderSplitLayout(
			statusBarContent,
			progressBarContent,
			topContent,
			bottomContent,
			hintsBarContent,
			topFocused,
			m.styles,
		)
	}

	// Build main content (logs)
	contentArea := m.contentView()

	// Use layout to render everything
	return m.layout.RenderFullLayout(
		statusBarContent,
		progressBarContent,
		contentArea,
		hintsBarContent,
		m.styles,
	)
}

// searchView renders the main view with search bar at bottom
func (m Model) searchView() string {
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

	// Build search bar instead of hints
	searchBarContent := m.searchBarView()

	// Use layout to render everything
	return m.layout.RenderFullLayout(
		statusBarContent,
		progressBarContent,
		contentArea,
		searchBarContent,
		m.styles,
	)
}

// searchBarView renders the search input bar
func (m Model) searchBarView() string {
	s := m.styles

	// Search icon and input
	searchStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Accent)

	inputStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Text)

	// Match count
	var matchInfo string
	if m.searchQuery != "" {
		countStyle := lipgloss.NewStyle().Foreground(s.Colors.TextMuted)
		if len(m.searchMatches) > 0 {
			matchInfo = countStyle.Render(fmt.Sprintf(" %d/%d", m.searchCursor+1, len(m.searchMatches)))
		} else {
			matchInfo = countStyle.Render(" No matches")
		}
	}

	// Hints
	hintStyle := lipgloss.NewStyle().Foreground(s.Colors.TextSubtle)
	hints := hintStyle.Render("  enter:confirm  esc:cancel  ↑↓:navigate")

	return searchStyle.Render("/") + " " + inputStyle.Render(m.searchInput.View()) + matchInfo + hints
}

// contentView renders the main content area (logs)
func (m Model) contentView() string {
	// Always render TabView - it handles empty state internally
	return m.tabView.View(m.styles)
}

// consoleView renders the console output pane for run mode
func (m Model) consoleView() string {
	s := m.styles

	if len(m.runMode.ConsoleLogs) == 0 {
		// Empty console state
		emptyStyle := lipgloss.NewStyle().
			Foreground(s.Colors.TextSubtle).
			Padding(1, 2)
		return emptyStyle.Render("Waiting for app output...")
	}

	// Calculate visible range
	height := m.layout.SplitBottomHeight()
	start := m.runMode.ConsolePos
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(m.runMode.ConsoleLogs) {
		end = len(m.runMode.ConsoleLogs)
	}

	// Build visible lines
	var lines []string
	lineStyle := lipgloss.NewStyle().Foreground(s.Colors.Text)
	for i := start; i < end; i++ {
		line := m.runMode.ConsoleLogs[i]
		// Truncate long lines
		if len(line) > m.layout.ContentWidth()-4 {
			line = line[:m.layout.ContentWidth()-7] + "..."
		}
		lines = append(lines, lineStyle.Render(line))
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// runModeHintsBar returns hints specific to run mode
func (m Model) runModeHintsBar() string {
	s := m.styles

	keyStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Accent).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextMuted)

	hints := []struct {
		Key  string
		Desc string
	}{
		{"tab", "switch pane"},
		{"x", "stop app"},
		{"↑↓", "scroll"},
		{"esc", "cancel"},
		{"?", "help"},
	}

	var parts []string
	for i, hint := range hints {
		part := keyStyle.Render(hint.Key) + ":" + descStyle.Render(hint.Desc)
		parts = append(parts, part)
		if i < len(hints)-1 {
			parts = append(parts, "  ")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
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

	// Calculate width (same as setupHelpViewport)
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
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width-6)))
	b.WriteString("\n")

	// Scrollable content from viewport
	b.WriteString(m.helpViewport.View())

	// Scroll indicator
	scrollInfo := ""
	if m.helpViewport.TotalLineCount() > m.helpViewport.Height {
		percent := int(m.helpViewport.ScrollPercent() * 100)
		scrollStyle := lipgloss.NewStyle().Foreground(s.Colors.TextSubtle)
		scrollInfo = scrollStyle.Render(fmt.Sprintf(" (%d%%)", percent))
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width-6)))
	b.WriteString("\n")

	hintKeyStyle := lipgloss.NewStyle().Foreground(s.Colors.Accent)
	hintDescStyle := lipgloss.NewStyle().Foreground(s.Colors.TextSubtle)
	hints := "Press " + hintKeyStyle.Render("?") + hintDescStyle.Render(" or ") +
		hintKeyStyle.Render("esc") + hintDescStyle.Render(" to close") +
		hintDescStyle.Render("  ↑↓ scroll") + scrollInfo
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
	headerContent := m.statusBar.View(m.width, m.styles)
	return lipgloss.JoinVertical(lipgloss.Left,
		m.layout.RenderHeader(headerContent, m.styles),
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
// Copy Functions
// =============================================================================

// copyCurrentLine copies the current line from the active tab to clipboard
func (m *Model) copyCurrentLine() tea.Cmd {
	var content string

	switch m.tabView.ActiveTab {
	case TabStream:
		content = m.tabView.StreamTab.GetCurrentLine()
	case TabIssues:
		if issue := m.tabView.IssuesTab.GetSelectedIssue(); issue != nil {
			content = issue.FullText
		}
	case TabDashboard:
		content = "" // Summary tab doesn't have line-by-line content
	}

	if content == "" {
		return func() tea.Msg {
			return statusMsg("Nothing to copy")
		}
	}

	return m.copyToClipboard(content, "Copied line")
}

// copyVisibleContent copies all visible content from the active tab to clipboard
func (m *Model) copyVisibleContent() tea.Cmd {
	var content string

	switch m.tabView.ActiveTab {
	case TabStream:
		content = m.tabView.StreamTab.GetVisibleContent()
	case TabIssues:
		// Copy all visible issues
		var lines []string
		for _, issue := range m.tabView.IssuesTab.Issues {
			lines = append(lines, issue.FullText)
		}
		content = strings.Join(lines, "\n")
	case TabDashboard:
		content = "" // Summary tab doesn't have copyable content
	}

	if content == "" {
		return func() tea.Msg {
			return statusMsg("Nothing to copy")
		}
	}

	return m.copyToClipboard(content, "Copied visible content")
}

// copyToClipboard copies content to system clipboard using pbcopy (macOS)
func (m *Model) copyToClipboard(content, successMsg string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(content)
		if err := cmd.Run(); err != nil {
			return statusMsg("Copy failed: " + err.Error())
		}
		return statusMsg(successMsg)
	}
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
