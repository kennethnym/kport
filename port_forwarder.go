package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ForwardingStartedMsg is sent when port forwarding starts
type ForwardingStartedMsg struct {
	LocalPort  int
	RemotePort int
}

// PortForwarder manages SSH port forwarding
type PortForwarder struct {
	sshClient    *ssh.Client
	localPort    int
	remotePort   int
	listener     net.Listener
	stopChan     chan struct{}
	wg           sync.WaitGroup
	isRunning    bool
	mu           sync.Mutex
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(sshClient *ssh.Client, localPort, remotePort int) *PortForwarder {
	return &PortForwarder{
		sshClient:  sshClient,
		localPort:  localPort,
		remotePort: remotePort,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the port forwarding
func (pf *PortForwarder) Start() error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if pf.isRunning {
		return fmt.Errorf("port forwarding already running")
	}

	// Create local listener
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", pf.localPort))
	if err != nil {
		return fmt.Errorf("failed to create local listener: %w", err)
	}

	pf.listener = listener
	pf.isRunning = true

	// Start accepting connections
	pf.wg.Add(1)
	go pf.acceptConnections()

	return nil
}

// Stop stops the port forwarding
func (pf *PortForwarder) Stop() {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if !pf.isRunning {
		return
	}

	pf.isRunning = false
	close(pf.stopChan)

	if pf.listener != nil {
		pf.listener.Close()
	}

	pf.wg.Wait()
}

// acceptConnections accepts and handles incoming connections
func (pf *PortForwarder) acceptConnections() {
	defer pf.wg.Done()

	for {
		select {
		case <-pf.stopChan:
			return
		default:
			// Set a timeout for Accept to avoid blocking indefinitely
			if tcpListener, ok := pf.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			conn, err := pf.listener.Accept()
			if err != nil {
				// Check if it's a timeout error and continue
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// If we're stopping, ignore the error
				select {
				case <-pf.stopChan:
					return
				default:
					continue
				}
			}

			// Handle the connection in a separate goroutine
			pf.wg.Add(1)
			go pf.handleConnection(conn)
		}
	}
}

// handleConnection handles a single connection
func (pf *PortForwarder) handleConnection(localConn net.Conn) {
	defer pf.wg.Done()
	defer localConn.Close()

	// Create connection to remote host through SSH
	remoteConn, err := pf.sshClient.Dial("tcp", fmt.Sprintf("localhost:%d", pf.remotePort))
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Copy data between connections
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(localConn, remoteConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(remoteConn, localConn)
	}()

	wg.Wait()
}

// StartPortForwarding starts port forwarding for a specific port
func StartPortForwarding(host SSHHost, remotePort int) tea.Cmd {
	return func() tea.Msg {
		// Find an available local port
		localPort, err := findAvailablePort()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to find available local port: %w", err)}
		}

		// Create SSH client
		client, err := createSSHClient(host)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to create SSH client: %w", err)}
		}

		// Create and start port forwarder
		forwarder := NewPortForwarder(client, localPort, remotePort)
		if err := forwarder.Start(); err != nil {
			client.Close()
			return ErrorMsg{Error: fmt.Errorf("failed to start port forwarding: %w", err)}
		}

		return ForwardingStartedMsg{
			LocalPort:  localPort,
			RemotePort: remotePort,
		}
	}
}

// StartManualPortForwarding starts port forwarding for a manually entered port
func StartManualPortForwarding(host SSHHost, portStr string) tea.Cmd {
	return func() tea.Msg {
		remotePort, err := strconv.Atoi(portStr)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("invalid port number: %s", portStr)}
		}

		if remotePort <= 0 || remotePort > 65535 {
			return ErrorMsg{Error: fmt.Errorf("port number must be between 1 and 65535")}
		}

		// Find an available local port
		localPort, err := findAvailablePort()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to find available local port: %w", err)}
		}

		// Create SSH client
		client, err := createSSHClient(host)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to create SSH client: %w", err)}
		}

		// Create and start port forwarder
		forwarder := NewPortForwarder(client, localPort, remotePort)
		if err := forwarder.Start(); err != nil {
			client.Close()
			return ErrorMsg{Error: fmt.Errorf("failed to start port forwarding: %w", err)}
		}

		return ForwardingStartedMsg{
			LocalPort:  localPort,
			RemotePort: remotePort,
		}
	}
}

// createSSHClient creates an SSH client for the given host
func createSSHClient(host SSHHost) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         5 * time.Second, // Shorter timeout
	}

	// Add key-based authentication if identity file is specified
	if host.Identity != "" {
		key, err := loadPrivateKey(host.Identity)
		if err == nil {
			config.Auth = append(config.Auth, ssh.PublicKeys(key))
		}
	}

	// Add SSH agent authentication
	if agentAuth, err := sshAgentAuth(); err == nil {
		config.Auth = append(config.Auth, agentAuth)
	}

	// Connect to the remote host
	addr := net.JoinHostPort(host.Hostname, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", host.Name, err)
	}

	return client, nil
}

// sshAgentAuth returns SSH agent authentication method
func sshAgentAuth() (ssh.AuthMethod, error) {
	// Try to connect to SSH agent
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	sshAgent := agent.NewClient(agentConn)
	return ssh.PublicKeysCallback(sshAgent.Signers), nil
}

// findAvailablePort finds an available local port
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}