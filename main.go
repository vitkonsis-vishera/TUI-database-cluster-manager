package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cluster-tui/pkg/patroni"
	"cluster-tui/pkg/postgres"
)

// --- Styles ---
var (
	primaryColor   = lipgloss.Color("#89B4FA") // Blue
	secondaryColor = lipgloss.Color("#F5C2E7") // Pink/Purple
	successColor   = lipgloss.Color("#A6E3A1") // Green
	warningColor   = lipgloss.Color("#F9E2AF") // Yellow
	dangerColor    = lipgloss.Color("#F38BA8") // Red
	subtleColor    = lipgloss.Color("#6C7086") // Muted Gray

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(primaryColor).
			Padding(0, 1)

	statusLineStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(secondaryColor).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CDD6F4")).
				Background(lipgloss.Color("#313244")).
				Padding(0, 2)

	tabBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#45475A"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1)

	badgePrimary = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(successColor).
			Padding(0, 1)

	badgeReplica = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#11111B")).
			Background(primaryColor).
			Padding(0, 1)

	badgeStandalone = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(warningColor).
			Padding(0, 1)
)

type activeTab int

const (
	tabConnect activeTab = iota
	tabTopology
	tabEngineStats
	tabActions
	tabLogs
)

type engineType int

const (
	EngineAutoDetect engineType = iota
	EnginePatroni
	EnginePacemaker
	EngineSingleNode
)

func (e engineType) String() string {
	switch e {
	case EnginePatroni:
		return "Patroni (etcd)"
	case EnginePacemaker:
		return "Corosync + Pacemaker"
	case EngineSingleNode:
		return "Single Node (Standalone PG)"
	default:
		return "Auto-Detecting..."
	}
}

// --- Custom Messages ---
type tickMsg time.Time
type clusterUpdateMsg struct {
	detectedEngine engineType
	status         *patroni.ClusterStatus
	metrics        map[string]postgres.NodeMetrics
	err            error
}

type model struct {
	tab            activeTab
	table          table.Model
	detectedEngine engineType
	pgFlavor       string
	targetHost     string
	targetPort     int
	width          int
	height         int

	// Form Inputs
	inputs     []textinput.Model
	focusIndex int
	connecting bool
	connected  bool

	patroniClient *patroni.Client
	pgManager     *postgres.PGPoolManager

	lastStatus  *patroni.ClusterStatus
	lastMetrics map[string]postgres.NodeMetrics
	lastError   error
}

func initialModel() model {
	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "localhost:5432 или http://127.0.0.1:8008"
	inputs[0].Focus()
	inputs[0].CharLimit = 128
	inputs[0].Width = 60
	inputs[0].Prompt = "Endpoint / Host: "

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "postgres"
	inputs[1].CharLimit = 64
	inputs[1].Width = 60
	inputs[1].Prompt = "PG User:         "

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "password"
	inputs[2].EchoMode = textinput.EchoPassword
	inputs[2].EchoCharacter = '•'
	inputs[2].CharLimit = 64
	inputs[2].Width = 60
	inputs[2].Prompt = "PG Password:     "

	columns := []table.Column{
		{Title: "Node Name", Width: 16},
		{Title: "Host", Width: 14},
		{Title: "Role", Width: 14},
		{Title: "State", Width: 12},
		{Title: "TL", Width: 5},
		{Title: "Lag (MB)", Width: 10},
		{Title: "Conns", Width: 10},
		{Title: "Cache Hit", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#45475A")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#11111B")).
		Background(primaryColor).
		Bold(false)
	t.SetStyles(s)

	return model{
		tab:            tabConnect,
		table:          t,
		inputs:         inputs,
		focusIndex:     0,
		detectedEngine: EngineAutoDetect,
		pgFlavor:       "PostgreSQL / Postgres Pro",
		lastMetrics:    make(map[string]postgres.NodeMetrics),
	}
}

// Извлечение чистого хоста и порта из ввода
func parseHostAndPort(rawInput string) (string, int) {
	cleaned := strings.TrimPrefix(rawInput, "http://")
	cleaned = strings.TrimPrefix(cleaned, "https://")
	if idx := strings.Index(cleaned, "/"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	host, portStr, err := net.SplitHostPort(cleaned)
	if err != nil {
		host = cleaned
		portStr = "5432"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 5432
	}

	if host == "" {
		host = "127.0.0.1"
	}

	return host, port
}

func (m model) fetchClusterDataCmd() tea.Cmd {
	return func() tea.Msg {
		if m.pgManager == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var status *patroni.ClusterStatus
		detected := EnginePatroni

		// 1. Пробуем подключиться через Patroni REST API
		if m.patroniClient != nil {
			st, err := m.patroniClient.GetClusterState(ctx)
			if err == nil {
				status = st
			}
		}

		// 2. Если Patroni недоступен, проверяем прямое подключение к PostgreSQL (Single Node)
		if status == nil {
			metrics := m.pgManager.FetchNodeMetrics(ctx, m.targetHost)
			if metrics.Error == nil {
				detected = EngineSingleNode
				status = &patroni.ClusterStatus{
					Members: []patroni.Member{
						{
							Name:     m.targetHost,
							Host:     m.targetHost,
							Role:     "standalone",
							State:    "running",
							Timeline: 1,
							Lag:      0,
						},
					},
				}
			} else {
				return clusterUpdateMsg{
					err: fmt.Errorf("unable to connect to PG at %s:%d: %v", m.targetHost, m.targetPort, metrics.Error),
				}
			}
		}

		// 3. Сбор метрик
		metricsMap := make(map[string]postgres.NodeMetrics)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, member := range status.Members {
			wg.Add(1)
			go func(host string) {
				defer wg.Done()
				metrics := m.pgManager.FetchNodeMetrics(ctx, host)

				mu.Lock()
				metricsMap[host] = metrics
				mu.Unlock()
			}(member.Host)
		}

		wg.Wait()

		return clusterUpdateMsg{
			detectedEngine: detected,
			status:         status,
			metrics:        metricsMap,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 6)

	case tickMsg:
		if m.connected {
			return m, tea.Batch(m.fetchClusterDataCmd(), tickCmd())
		}

	case clusterUpdateMsg:
		m.connecting = false
		if msg.err != nil {
			m.lastError = msg.err
			m.connected = false
			return m, nil
		}

		m.lastError = nil
		m.connected = true
		m.detectedEngine = msg.detectedEngine
		m.lastStatus = msg.status
		m.lastMetrics = msg.metrics

		if m.tab == tabConnect {
			m.tab = tabTopology
		}

		rows := []table.Row{}
		for _, member := range msg.status.Members {
			metrics, hasMetrics := msg.metrics[member.Host]

			connsStr := "N/A"
			cacheStr := "N/A"
			if hasMetrics && metrics.Error == nil {
				connsStr = fmt.Sprintf("%d/%d", metrics.ActiveConnections, metrics.MaxConnections)
				cacheStr = fmt.Sprintf("%.1f%%", metrics.CacheHitRatio)
			}

			lagMb := fmt.Sprintf("%d", member.Lag/(1024*1024))

			rows = append(rows, table.Row{
				member.Name,
				member.Host,
				member.Role,
				member.State,
				fmt.Sprintf("%d", member.Timeline),
				lagMb,
				connsStr,
				cacheStr,
			})
		}
		m.table.SetRows(rows)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.pgManager != nil {
				m.pgManager.Close()
			}
			return m, tea.Quit

		case "q":
			if m.tab != tabConnect {
				if m.pgManager != nil {
					m.pgManager.Close()
				}
				return m, tea.Quit
			}
		}

		if m.connected {
			switch msg.String() {
			case "tab", "l", "right":
				if m.tab > tabConnect {
					m.tab = (m.tab % 4) + 1
				}
			case "shift+tab", "h", "left":
				if m.tab > tabConnect {
					m.tab = (m.tab-2+4)%4 + 1
				}
			case "1":
				m.tab = tabTopology
			case "2":
				m.tab = tabEngineStats
			case "3":
				m.tab = tabActions
			case "4":
				m.tab = tabLogs
			}
		}

		if m.tab == tabConnect {
			switch msg.String() {
			case "up":
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) - 1
				}
				return m, m.updateFocus()

			case "down":
				m.focusIndex++
				if m.focusIndex >= len(m.inputs) {
					m.focusIndex = 0
				}
				return m, m.updateFocus()

			case "enter":
				rawEndpoint := m.inputs[0].Value()
				if rawEndpoint == "" {
					rawEndpoint = m.inputs[0].Placeholder
				}

				host, port := parseHostAndPort(rawEndpoint)
				m.targetHost = host
				m.targetPort = port

				pgUser := m.inputs[1].Value()
				if pgUser == "" {
					pgUser = m.inputs[1].Placeholder
				}

				pgPass := m.inputs[2].Value()
				if pgPass == "" {
					pgPass = m.inputs[2].Placeholder
				}

				httpEndpoint := rawEndpoint
				if !strings.HasPrefix(httpEndpoint, "http://") && !strings.HasPrefix(httpEndpoint, "https://") {
					httpEndpoint = "http://" + httpEndpoint
				}

				m.patroniClient = patroni.NewClient([]string{httpEndpoint}, 1500*time.Millisecond)
				m.pgManager = postgres.NewPGPoolManager(postgres.Config{
					User:     pgUser,
					Password: pgPass,
					Database: "postgres",
					Port:     port,
				})

				m.connecting = true
				m.lastError = nil
				return m, tea.Batch(m.fetchClusterDataCmd(), tickCmd())
			}

			cmd := m.updateInputs(msg)
			return m, cmd
		}
	}

	if m.tab == tabTopology && m.connected {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing TUI..."
	}

	containerWidth := m.width - 4
	if containerWidth < 40 {
		containerWidth = 40
	}

	header := headerStyle.Render(" PG-CLUSTER TUI ") + "  " +
		lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Engine: "+m.detectedEngine.String()) +
		statusLineStyle.Render(" | Target: "+m.pgFlavor)

	tabs := []string{"0: Connect", "1: Topology", "2: Metrics", "3: Actions", "4: Event Logs"}
	var renderedTabs []string
	for i, t := range tabs {
		if activeTab(i) == m.tab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(t))
		}
	}

	tabRow := tabBorderStyle.Width(containerWidth + 2).Render(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...))
	currentBoxStyle := boxStyle.Width(containerWidth)

	var body string

	if m.tab == tabConnect {
		var statusMsg string
		if m.connecting {
			statusMsg = lipgloss.NewStyle().Foreground(warningColor).Render("Detecting HA Engine / Standalone PG & Connecting...")
		} else if m.lastError != nil {
			errText := fmt.Sprintf("Connection Error: %v", m.lastError)
			statusMsg = lipgloss.NewStyle().
				Foreground(dangerColor).
				Width(containerWidth - 4).
				Render(errText)
		}

		form := lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Connect to PostgreSQL Cluster or Single Database"),
			"\n\n",
			m.inputs[0].View(),
			"\n",
			m.inputs[1].View(),
			"\n",
			m.inputs[2].View(),
			"\n",
			statusMsg,
			"\n\n",
			lipgloss.NewStyle().Foreground(subtleColor).Render("Press [ENTER] to Connect  •  [Up/Down] Move Focus"),
		)
		body = currentBoxStyle.Render(form)
	} else {
		switch m.tab {
		case tabTopology:
			summary := badgeStandalone.Render("SINGLE NODE / STANDALONE MODE")
			if m.detectedEngine != EngineSingleNode {
				primaryNode := "Unknown"
				totalNodes := 0
				isPause := false

				if m.lastStatus != nil {
					totalNodes = len(m.lastStatus.Members)
					isPause = m.lastStatus.Pause
					for _, mem := range m.lastStatus.Members {
						if mem.Role == "leader" {
							primaryNode = mem.Name
							break
						}
					}
				}

				maintStr := lipgloss.NewStyle().Foreground(successColor).Render("Maintenance: OFF")
				if isPause {
					maintStr = lipgloss.NewStyle().Foreground(warningColor).Render("Maintenance: ON (Paused)")
				}

				summary = lipgloss.JoinHorizontal(
					lipgloss.Left,
					badgePrimary.Render("PRIMARY: "+primaryNode),
					"  ",
					badgeReplica.Render(fmt.Sprintf("NODES: %d", totalNodes)),
					"  ",
					maintStr,
				)
			}

			body = currentBoxStyle.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					summary,
					"\n",
					m.table.View(),
				),
			)

		case tabEngineStats:
			var statsText string
			if len(m.lastMetrics) == 0 {
				statsText = "Waiting for metrics..."
			} else {
				for host, mtr := range m.lastMetrics {
					if mtr.Error != nil {
						statsText += fmt.Sprintf("[%s] Error: %v\n", host, mtr.Error)
					} else {
						statsText += fmt.Sprintf("[%s] Connections: %d/%d | Cache Hit Ratio: %.2f%%\n",
							host, mtr.ActiveConnections, mtr.MaxConnections, mtr.CacheHitRatio)
					}
				}
			}

			body = currentBoxStyle.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("PostgreSQL Performance Metrics"),
					"\n",
					statsText,
				),
			)

		case tabActions:
			if m.detectedEngine == EngineSingleNode {
				body = currentBoxStyle.Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render("Cluster Management Actions Disabled"),
						"\n",
						lipgloss.NewStyle().Foreground(subtleColor).Render(
							"HA operations (Switchover, Failover, Reinitialization) are unavailable for Single-Node / Standalone PostgreSQL instances.",
						),
					),
				)
			} else {
				body = currentBoxStyle.Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						lipgloss.NewStyle().Bold(true).Foreground(dangerColor).Render("Cluster Management Actions"),
						"\n",
						"[S] Switchover (Graceful Leader Transfer)",
						"[F] Failover (Force Leader Selection)",
						"[R] Reinitialize Replica",
						"[P] Pause/Resume Auto-failover (Maintenance Mode)",
					),
				)
			}

		case tabLogs:
			body = currentBoxStyle.Render(
				lipgloss.NewStyle().Foreground(subtleColor).Render(
					"Real-time events log viewer...",
				),
			)
		}
	}

	footer := statusLineStyle.Render("Tab/Arrow: Move / Switch Tab  •  Ctrl+C: Quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabRow,
		"\n",
		body,
		"\n",
		footer,
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
