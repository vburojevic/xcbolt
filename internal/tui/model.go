package tui

import (
	"context"
	"fmt"
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

type tab int

const (
	tabBuild tab = iota
	tabRun
	tabTest
	tabLogs
	tabContext
	tabDoctor
)

type eventMsg core.Event

type contextLoadedMsg struct {
	info core.ContextInfo
	cfg  core.Config
	err  error
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

type Model struct {
	projectRoot string
	configPath  string

	cfg  core.Config
	info core.ContextInfo

	width  int
	height int

	styles styleSet
	keys   keyMap
	help   help.Model

	spinner spinner.Model

	viewport viewport.Model
	logLines []string

	tab tab

	running    bool
	runningCmd string
	cancelFn   context.CancelFunc
	eventCh    <-chan core.Event
	doneCh     <-chan opDoneMsg

	toast toastModel

	mode   string // "main" | "wizard"
	wizard wizardModel

	lastBuild core.BuildResult
	lastRun   core.RunResult
	lastTest  core.TestResult

	lastErr string
}

func NewModel(projectRoot string, configPath string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	h := help.New()
	h.ShowAll = false

	return Model{
		projectRoot: projectRoot,
		configPath:  configPath,
		styles:      defaultStyles(),
		keys:        defaultKeyMap(),
		help:        h,
		spinner:     sp,
		viewport:    vp,
		logLines:    []string{},
		tab:         tabLogs,
		toast:       newToast(),
		mode:        "main",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadContextCmd(m.projectRoot, m.configPath),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func loadContextCmd(projectRoot, configPath string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := core.LoadConfig(projectRoot, configPath)
		if err != nil {
			return contextLoadedMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		// Use a silent emitter (TUI will render its own).
		emit := core.NewTextEmitter(ioDiscard{})
		info, cfg2, err := core.DiscoverContext(ctx, projectRoot, cfg, emit)
		if err != nil {
			return contextLoadedMsg{err: err}
		}
		return contextLoadedMsg{info: info, cfg: cfg2}
	}
}

// ioDiscard is a minimal io.Writer that discards output (avoid importing io for a single use).
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 3
		footerH := 2
		panelH := max(0, m.height-headerH-footerH)
		m.viewport.Width = max(0, m.width-4)
		m.viewport.Height = max(0, panelH-4)
		if m.mode == "wizard" {
			m.wizard = newWizard(m.info, m.cfg, m.width)
		}
		cmds = append(cmds, nil)

	case contextLoadedMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.toast.Show("Context load failed", 2*time.Second)
			break
		}
		m.info = msg.info
		m.cfg = msg.cfg
		m.toast.Show("Context ready", 1200*time.Millisecond)

	case tickMsg:
		m.toast.Update()
		cmds = append(cmds, tickCmd())

	case tea.KeyMsg:
		// Wizard mode: delegate to wizard, but still allow quit/cancel.
		if m.mode == "wizard" {
			switch {
			case keyMatches(msg, m.keys.Quit):
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.wizard, cmd = m.wizard.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch {
		case keyMatches(msg, m.keys.Quit):
			return m, tea.Quit
		case keyMatches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case keyMatches(msg, m.keys.Tab):
			m.tab = (m.tab + 1) % 6
		case keyMatches(msg, m.keys.PrevTab):
			m.tab = (m.tab + 5) % 6
		case keyMatches(msg, m.keys.Logs):
			m.tab = tabLogs
		case keyMatches(msg, m.keys.Refresh):
			cmds = append(cmds, loadContextCmd(m.projectRoot, m.configPath))
			m.toast.Show("Refreshing…", 1*time.Second)
		case keyMatches(msg, m.keys.Init):
			m.mode = "wizard"
			m.wizard = newWizard(m.info, m.cfg, m.width)
			cmds = append(cmds, m.wizard.Init())
		case keyMatches(msg, m.keys.Cancel):
			if m.running && m.cancelFn != nil {
				m.cancelFn()
				m.toast.Show("Canceled", 900*time.Millisecond)
			}
		case keyMatches(msg, m.keys.Build):
			if !m.running {
				cmds = append(cmds, m.startOp("build"))
			}
		case keyMatches(msg, m.keys.Run):
			if !m.running {
				cmds = append(cmds, m.startOp("run"))
			}
		case keyMatches(msg, m.keys.Test):
			if !m.running {
				cmds = append(cmds, m.startOp("test"))
			}
		}

	case wizardDoneMsg:
		m.mode = "main"
		if msg.aborted {
			m.toast.Show("Init canceled", 1200*time.Millisecond)
			break
		}
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.toast.Show("Init failed", 2*time.Second)
			break
		}
		m.cfg = msg.cfg
		if err := core.SaveConfig(m.projectRoot, m.configPath, m.cfg); err != nil {
			m.lastErr = err.Error()
			m.toast.Show("Save failed", 2*time.Second)
			break
		}
		m.toast.Show("Saved .xcbolt/config.json", 1400*time.Millisecond)
		cmds = append(cmds, loadContextCmd(m.projectRoot, m.configPath))

	case eventMsg:
		ev := core.Event(msg)
		m.appendLog(formatEventLine(ev))
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()
		// Keep listening for more events.
		if m.eventCh != nil {
			cmds = append(cmds, waitForEvent(m.eventCh))
		}

	case opDoneMsg:
		m.running = false
		m.runningCmd = ""
		m.cancelFn = nil
		m.eventCh = nil
		m.doneCh = nil
		if msg.cfg.Version != 0 {
			m.cfg = msg.cfg
		}
		if msg.build != nil {
			m.lastBuild = *msg.build
		}
		if msg.run != nil {
			m.lastRun = *msg.run
		}
		if msg.test != nil {
			m.lastTest = *msg.test
		}
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.toast.Show(strings.ToUpper(msg.cmd)+" failed", 2*time.Second)
		} else {
			m.toast.Show(strings.ToUpper(msg.cmd)+" done", 1200*time.Millisecond)
		}
		// Refresh context after build/run/test to catch new devices/sims.
		cmds = append(cmds, loadContextCmd(m.projectRoot, m.configPath))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// viewport update for scroll keys, mouse wheel, etc.
	if m.mode == "main" && m.tab == tabLogs {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) startOp(name string) tea.Cmd {
	m.running = true
	m.runningCmd = name
	m.appendLog("—")
	m.appendLog(fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), strings.ToUpper(name)))

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
		default:
			done <- opDoneMsg{cmd: name, err: fmt.Errorf("unknown op %s", name)}
		}
		close(done)
	}()
	return tea.Batch(waitForEvent(events), waitForDone(done))
}

type chanEmitter struct {
	ch chan<- core.Event
}

func (e *chanEmitter) Emit(ev core.Event) {
	select {
	case e.ch <- ev:
	default:
		// Drop if UI is too slow; keep the tool responsive.
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
	// Cap memory
	if len(m.logLines) > 2000 {
		m.logLines = m.logLines[len(m.logLines)-2000:]
	}
}

func formatEventLine(ev core.Event) string {
	// Compact format.
	prefix := ""
	switch ev.Type {
	case "error":
		prefix = "✖ "
	case "warning":
		prefix = "! "
	case "result":
		prefix = "✓ "
	case "status":
		prefix = "• "
	case "log":
		prefix = ""
	default:
		prefix = ""
	}
	msg := ev.Msg
	if msg == "" && ev.Err != nil {
		msg = ev.Err.Message
	}
	if msg == "" {
		msg = fmt.Sprintf("%s", ev.Type)
	}
	return prefix + msg
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.mode == "wizard" {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.headerView(),
			m.styles.Panel.Render(m.wizard.View()),
			m.footerView(),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.tabsView(),
		m.panelView(),
		m.footerView(),
	)
}

func (m Model) headerView() string {
	name := m.styles.Brand.Render("xcbolt")
	proj := filepath.Base(m.projectRoot)
	meta := []string{proj}
	if m.cfg.Scheme != "" {
		meta = append(meta, "scheme: "+m.cfg.Scheme)
	}
	if m.cfg.Destination.Kind != "" {
		label := string(m.cfg.Destination.Kind)
		if m.cfg.Destination.Name != "" {
			label += " · " + m.cfg.Destination.Name
		}
		meta = append(meta, "dest: "+label)
	}
	status := "idle"
	if m.running {
		status = m.spinner.View() + " " + strings.ToUpper(m.runningCmd)
	}
	line := lipgloss.JoinHorizontal(lipgloss.Center,
		name,
		"  ",
		m.styles.Meta.Render(strings.Join(meta, "  •  ")),
		lipgloss.NewStyle().Width(max(0, m.width-2)).Align(lipgloss.Right).Render(m.styles.Meta.Render(status)),
	)
	return m.styles.Header.Width(m.width).Render(line)
}

func (m Model) tabsView() string {
	tabs := []string{"Build", "Run", "Test", "Logs", "Context", "Doctor"}
	r := []string{}
	for i, t := range tabs {
		if tab(i) == m.tab {
			r = append(r, m.styles.TabActive.Render(t))
		} else {
			r = append(r, m.styles.TabInactive.Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, r...)
}

func (m Model) panelView() string {
	w := max(0, m.width-2)
	h := max(0, m.height-6)

	title := ""
	body := ""

	switch m.tab {
	case tabBuild:
		title = "Build"
		body = m.buildView()
	case tabRun:
		title = "Run"
		body = m.runView()
	case tabTest:
		title = "Test"
		body = m.testView()
	case tabLogs:
		title = "Logs"
		body = m.logsView()
	case tabContext:
		title = "Context"
		body = m.contextView()
	case tabDoctor:
		title = "Doctor"
		body = m.doctorView()
	default:
		title = ""
		body = ""
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.PanelTitle.Render(title),
		body,
	)

	return m.styles.Panel.Width(w).Height(h).Render(content)
}

func (m Model) buildView() string {
	if m.lastBuild.ResultBundle == "" {
		return m.styles.Muted.Render("Press 'b' to build")
	}
	return fmt.Sprintf(
		"Result: %s\nExit: %d\nDuration: %s\n",
		m.lastBuild.ResultBundle,
		m.lastBuild.ExitCode,
		m.lastBuild.Duration.Round(100*time.Millisecond),
	)
}

func (m Model) runView() string {
	if m.lastRun.BundleID == "" {
		return m.styles.Muted.Render("Press 'r' to run")
	}
	return fmt.Sprintf(
		"Bundle: %s\nTarget: %s\nPID: %d\nApp: %s\n",
		m.lastRun.BundleID,
		m.lastRun.Target,
		m.lastRun.PID,
		m.lastRun.AppPath,
	)
}

func (m Model) testView() string {
	if m.lastTest.ResultBundle == "" {
		return m.styles.Muted.Render("Press 't' to test")
	}
	return fmt.Sprintf(
		"Result: %s\nExit: %d\nDuration: %s\n",
		m.lastTest.ResultBundle,
		m.lastTest.ExitCode,
		m.lastTest.Duration.Round(100*time.Millisecond),
	)
}

func (m Model) logsView() string {
	if len(m.logLines) == 0 {
		return m.styles.Muted.Render("No logs yet. Run build/run/test or press 'c' to refresh context.")
	}
	m.viewport.Width = max(0, m.width-8)
	m.viewport.Height = max(0, m.height-12)
	return m.viewport.View()
}

func (m Model) contextView() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("Project: %s", m.projectRoot))
	if len(m.info.Workspaces) > 0 {
		lines = append(lines, fmt.Sprintf("Workspaces: %d", len(m.info.Workspaces)))
	}
	if len(m.info.Projects) > 0 {
		lines = append(lines, fmt.Sprintf("Projects: %d", len(m.info.Projects)))
	}
	lines = append(lines, fmt.Sprintf("Schemes: %d", len(m.info.Schemes)))
	lines = append(lines, fmt.Sprintf("Simulators: %d", len(m.info.Simulators)))
	lines = append(lines, fmt.Sprintf("Devices: %d", len(m.info.Devices)))
	return strings.Join(lines, "\n")
}

func (m Model) doctorView() string {
	// Lightweight hints: full doctor report is available via `xcbolt doctor`.
	lines := []string{
		"Quick checks:",
		"- `xcrun xcodebuild -version`",
		"- `xcrun simctl list --json`",
		"- `xcrun devicectl help`",
		"- `xcrun xcresulttool help`",
		"\nRun `xcbolt doctor` for a full report.",
	}
	if m.lastErr != "" {
		lines = append(lines, "\nLast error:\n"+m.lastErr)
	}
	return strings.Join(lines, "\n")
}

func (m Model) footerView() string {
	left := m.help.View(m.keys)
	right := m.toast.View(m.styles)

	// Keep bottom line clean.
	line := lipgloss.JoinHorizontal(lipgloss.Top,
		left,
		lipgloss.NewStyle().Width(max(0, m.width-lipgloss.Width(left)-2)).Align(lipgloss.Right).Render(right),
	)
	return line
}

func keyMatches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
