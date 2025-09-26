package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// ForwardingStartedMsg is sent when port forwarding starts
type ForwardingStartedMsg struct {
	LocalPort  int
	RemotePort int
}

// PortForwarder manages SSH port forwarding using ssh command
type PortForwarder struct {
	hostName     string
	localPort    int
	remotePort   int
	sshCmd       *exec.Cmd
	stopChan     chan struct{}
	wg           sync.WaitGroup
	isRunning    bool
	mu           sync.Mutex
}

// NewPortForwarder creates a new port forwarder using ssh command
func NewPortForwarder(hostName string, localPort, remotePort int) *PortForwarder {
	return &PortForwarder{
		hostName:   hostName,
		localPort:  localPort,
		remotePort: remotePort,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the port forwarding using ssh command
func (pf *PortForwarder) Start() error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if pf.isRunning {
		return fmt.Errorf("port forwarding already running")
	}

	// Use ssh command with -L flag for local port forwarding
	// Format: ssh -L localport:localhost:remoteport hostname
	pf.sshCmd = exec.Command("ssh", 
		"-L", fmt.Sprintf("%d:localhost:%d", pf.localPort, pf.remotePort),
		"-N", // Don't execute remote command, just forward ports
		"-o", "ExitOnForwardFailure=yes", // Exit if port forwarding fails
		"-o", "ServerAliveInterval=30", // Keep connection alive
		"-o", "ServerAliveCountMax=3",
		pf.hostName)

	fmt.Fprintf(os.Stderr, "Debug: Starting SSH command: %s\n", pf.sshCmd.String())

	// Start the SSH command
	if err := pf.sshCmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH port forwarding: %w", err)
	}

	pf.isRunning = true

	// Monitor the SSH process
	pf.wg.Add(1)
	go pf.monitorSSH()

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

	// Kill the SSH process
	if pf.sshCmd != nil && pf.sshCmd.Process != nil {
		fmt.Fprintf(os.Stderr, "Debug: Stopping SSH port forwarding\n")
		pf.sshCmd.Process.Kill()
	}

	pf.wg.Wait()
}

// monitorSSH monitors the SSH process
func (pf *PortForwarder) monitorSSH() {
	defer pf.wg.Done()

	// Wait for the SSH command to finish or be stopped
	select {
	case <-pf.stopChan:
		// We were asked to stop
		return
	default:
		// Wait for SSH command to finish
		if err := pf.sshCmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Debug: SSH command finished with error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Debug: SSH command finished successfully\n")
		}
	}
}

// StartPortForwarding starts port forwarding for a specific port
func StartPortForwarding(host SSHHost, remotePort int) tea.Cmd {
	return func() tea.Msg {
		fmt.Fprintf(os.Stderr, "Debug: Starting port forwarding for %s:%d\n", host.Name, remotePort)
		
		// Try to use the same port locally, fallback to random if unavailable
		localPort, samePort, err := findPreferredLocalPort(remotePort)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Debug: Failed to find available port: %v\n", err)
			return ErrorMsg{Error: fmt.Errorf("failed to find available local port: %w", err)}
		}
		if samePort {
			fmt.Fprintf(os.Stderr, "Debug: Using same port locally: %d\n", localPort)
		} else {
			fmt.Fprintf(os.Stderr, "Debug: Port %d unavailable, using alternative: %d\n", remotePort, localPort)
		}

		// Create and start port forwarder using ssh command
		forwarder := NewPortForwarder(host.Name, localPort, remotePort)
		if err := forwarder.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Debug: Failed to start port forwarder: %v\n", err)
			return ErrorMsg{Error: fmt.Errorf("failed to start port forwarding: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "Debug: Port forwarder started successfully\n")

		return ForwardingStartedMsg{
			LocalPort:  localPort,
			RemotePort: remotePort,
		}
	}
}

// StartManualPortForwarding starts port forwarding for a manually entered port
func StartManualPortForwarding(host SSHHost, portStr string) tea.Cmd {
	return func() tea.Msg {
		fmt.Fprintf(os.Stderr, "Debug: Manual port forwarding requested for %s:%s\n", host.Name, portStr)
		
		remotePort, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Debug: Invalid port number: %s\n", portStr)
			return ErrorMsg{Error: fmt.Errorf("invalid port number: %s", portStr)}
		}

		if remotePort <= 0 || remotePort > 65535 {
			fmt.Fprintf(os.Stderr, "Debug: Port number out of range: %d\n", remotePort)
			return ErrorMsg{Error: fmt.Errorf("port number must be between 1 and 65535")}
		}

		// Try to use the same port locally, fallback to random if unavailable
		localPort, samePort, err := findPreferredLocalPort(remotePort)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Debug: Failed to find available port: %v\n", err)
			return ErrorMsg{Error: fmt.Errorf("failed to find available local port: %w", err)}
		}
		if samePort {
			fmt.Fprintf(os.Stderr, "Debug: Using same port locally: %d\n", localPort)
		} else {
			fmt.Fprintf(os.Stderr, "Debug: Port %d unavailable, using alternative: %d\n", remotePort, localPort)
		}

		// Create and start port forwarder using ssh command
		forwarder := NewPortForwarder(host.Name, localPort, remotePort)
		if err := forwarder.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Debug: Failed to start port forwarder: %v\n", err)
			return ErrorMsg{Error: fmt.Errorf("failed to start port forwarding: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "Debug: Port forwarder started successfully\n")

		return ForwardingStartedMsg{
			LocalPort:  localPort,
			RemotePort: remotePort,
		}
	}
}



// findPreferredLocalPort tries to use the same port as remote, fallback to random
func findPreferredLocalPort(remotePort int) (localPort int, samePort bool, err error) {
	// First try to use the same port as the remote port
	if isPortAvailable(remotePort) {
		return remotePort, true, nil
	}
	
	// If same port is not available, find any available port
	availablePort, err := findAvailablePort()
	if err != nil {
		return 0, false, err
	}
	
	return availablePort, false, nil
}

// isPortAvailable checks if a specific port is available locally
func isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
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