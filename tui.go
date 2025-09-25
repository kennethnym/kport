package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppState represents the current state of the application
type AppState int

const (
	StateSelectHost AppState = iota
	StateConnecting
	StateSelectPort
	StateManualPort
	StateForwarding
)

// Model represents the TUI model
type Model struct {
	state       AppState
	sshConfig   *SSHConfig
	hosts       []SSHHost
	selectedHost int
	ports       []int
	selectedPort int
	cursor      int
	manualPort  string
	forwarder   *PortForwarder
	message     string
	err         error
}

// NewModel creates a new TUI model
func NewModel() *Model {
	return &Model{
		state:     StateSelectHost,
		sshConfig: NewSSHConfig(),
		cursor:    0,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// Load SSH config
	if err := m.sshConfig.LoadConfig(); err != nil {
		m.err = err
		// Don't quit immediately, let user see the error
		return nil
	}
	m.hosts = m.sshConfig.GetHosts()
	
	// Check if we have any hosts
	if len(m.hosts) == 0 {
		m.err = fmt.Errorf("no SSH hosts found in config file")
		return nil
	}
	
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateSelectHost:
			return m.updateHostSelection(msg)
		case StateConnecting:
			return m.updateConnecting(msg)
		case StateSelectPort:
			return m.updatePortSelection(msg)
		case StateManualPort:
			return m.updateManualPort(msg)
		case StateForwarding:
			return m.updateForwarding(msg)
		}
	case PortsDetectedMsg:
		m.ports = msg.Ports
		m.state = StateSelectPort
		m.cursor = 0
		// Set a message about the connection attempt
		if len(msg.Ports) == 0 {
			m.message = fmt.Sprintf("Could not connect to %s or no ports detected", m.hosts[m.selectedHost].Name)
		} else {
			m.message = ""
		}
		return m, nil
	case ForwardingStartedMsg:
		m.message = fmt.Sprintf("Port forwarding started: localhost:%d -> %s:%d", 
			msg.LocalPort, m.hosts[m.selectedHost].Name, msg.RemotePort)
		m.state = StateForwarding
		return m, nil
	case ErrorMsg:
		m.err = msg.Error
		return m, tea.Quit
	}
	return m, nil
}

// updateHostSelection handles host selection state
func (m *Model) updateHostSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.hosts)-1 {
			m.cursor++
		}
	case "enter", " ":
		m.selectedHost = m.cursor
		m.state = StateConnecting
		m.message = fmt.Sprintf("Connecting to %s...", m.hosts[m.selectedHost].Name)
		// Detect ports on selected host
		return m, DetectPorts(m.hosts[m.selectedHost])
	case "m":
		// Manual port forwarding
		m.selectedHost = m.cursor
		m.state = StateManualPort
		m.manualPort = ""
		return m, nil
	}
	return m, nil
}

// updateConnecting handles connecting state
func (m *Model) updateConnecting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = StateSelectHost
		m.cursor = m.selectedHost
		m.message = ""
		return m, nil
	}
	return m, nil
}

// updatePortSelection handles port selection state
func (m *Model) updatePortSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = StateSelectHost
		m.cursor = m.selectedHost
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.ports)-1 {
			m.cursor++
		}
	case "enter", " ":
		m.selectedPort = m.cursor
		// Start port forwarding
		return m, StartPortForwarding(m.hosts[m.selectedHost], m.ports[m.selectedPort])
	case "m":
		// Manual port forwarding
		m.state = StateManualPort
		m.manualPort = ""
		return m, nil
	}
	return m, nil
}

// updateManualPort handles manual port input state
func (m *Model) updateManualPort(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		if len(m.ports) > 0 {
			m.state = StateSelectPort
		} else {
			m.state = StateSelectHost
		}
		return m, nil
	case "enter":
		if m.manualPort != "" {
			// Parse and start manual port forwarding
			return m, StartManualPortForwarding(m.hosts[m.selectedHost], m.manualPort)
		}
	case "backspace":
		if len(m.manualPort) > 0 {
			m.manualPort = m.manualPort[:len(m.manualPort)-1]
		}
	default:
		// Add character to manual port
		if len(msg.String()) == 1 && msg.String() >= "0" && msg.String() <= "9" {
			m.manualPort += msg.String()
		}
	}
	return m, nil
}

// updateForwarding handles forwarding state
func (m *Model) updateForwarding(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.forwarder != nil {
			m.forwarder.Stop()
		}
		return m, tea.Quit
	case "esc":
		if m.forwarder != nil {
			m.forwarder.Stop()
		}
		m.state = StateSelectHost
		m.cursor = 0
		m.message = ""
		return m, nil
	}
	return m, nil
}

// View renders the TUI
func (m *Model) View() string {
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)
		
		return fmt.Sprintf("%s\n\n%s\n\nPress q to quit.", 
			errorStyle.Render("❌ Error"), m.err.Error())
	}

	var s strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	s.WriteString(headerStyle.Render("kport - SSH Port Forwarder"))
	s.WriteString("\n\n")

	switch m.state {
	case StateSelectHost:
		s.WriteString(m.renderHostSelection())
	case StateConnecting:
		s.WriteString(m.renderConnecting())
	case StateSelectPort:
		s.WriteString(m.renderPortSelection())
	case StateManualPort:
		s.WriteString(m.renderManualPort())
	case StateForwarding:
		s.WriteString(m.renderForwarding())
	}

	return s.String()
}

// renderHostSelection renders the host selection view
func (m *Model) renderHostSelection() string {
	var s strings.Builder
	
	s.WriteString("Select an SSH host:\n\n")

	for i, host := range m.hosts {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		hostInfo := fmt.Sprintf("%s@%s", host.User, host.Hostname)
		if host.User == "" {
			hostInfo = host.Hostname
		}

		style := lipgloss.NewStyle()
		if m.cursor == i {
			style = style.Foreground(lipgloss.Color("#FF75B7"))
		}

		s.WriteString(fmt.Sprintf("%s %s (%s)\n", cursor, 
			style.Render(host.Name), hostInfo))
	}

	s.WriteString("\n")
	s.WriteString("Controls:\n")
	s.WriteString("  ↑/↓: Navigate  Enter: Select  m: Manual port  q: Quit\n")

	return s.String()
}

// renderConnecting renders the connecting view
func (m *Model) renderConnecting() string {
	var s strings.Builder
	
	connectingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00BFFF")).
		Bold(true)
	
	s.WriteString(connectingStyle.Render("🔄 " + m.message))
	s.WriteString("\n\n")
	s.WriteString("Please wait while connecting to the remote host...\n\n")
	s.WriteString("Controls:\n")
	s.WriteString("  Esc: Cancel and go back  q: Quit\n")

	return s.String()
}

// renderPortSelection renders the port selection view
func (m *Model) renderPortSelection() string {
	var s strings.Builder
	
	host := m.hosts[m.selectedHost]
	s.WriteString(fmt.Sprintf("Detected ports on %s:\n\n", host.Name))

	if len(m.ports) == 0 {
		if m.message != "" {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
			s.WriteString(warningStyle.Render("⚠️  " + m.message))
			s.WriteString("\n\n")
		}
		s.WriteString("No open ports detected.\n\n")
		s.WriteString("Press 'm' for manual port forwarding or Esc to go back.\n")
		return s.String()
	}

	for i, port := range m.ports {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		style := lipgloss.NewStyle()
		if m.cursor == i {
			style = style.Foreground(lipgloss.Color("#FF75B7"))
		}

		s.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(fmt.Sprintf("Port %d", port))))
	}

	s.WriteString("\n")
	s.WriteString("Controls:\n")
	s.WriteString("  ↑/↓: Navigate  Enter: Forward  m: Manual port  Esc: Back  q: Quit\n")

	return s.String()
}

// renderManualPort renders the manual port input view
func (m *Model) renderManualPort() string {
	var s strings.Builder
	
	host := m.hosts[m.selectedHost]
	s.WriteString(fmt.Sprintf("Manual port forwarding for %s:\n\n", host.Name))
	
	s.WriteString("Enter remote port number: ")
	
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	
	s.WriteString(inputStyle.Render(m.manualPort))
	s.WriteString("\n\n")
	
	s.WriteString("Controls:\n")
	s.WriteString("  Enter: Start forwarding  Esc: Back  q: Quit\n")

	return s.String()
}

// renderForwarding renders the forwarding status view
func (m *Model) renderForwarding() string {
	var s strings.Builder
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)
	
	s.WriteString(successStyle.Render("✓ Port Forwarding Active"))
	s.WriteString("\n\n")
	s.WriteString(m.message)
	s.WriteString("\n\n")
	s.WriteString("Controls:\n")
	s.WriteString("  Esc: Stop forwarding and return  q: Quit\n")

	return s.String()
}