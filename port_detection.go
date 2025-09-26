package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"
)

// PortsDetectedMsg is sent when ports are detected
type PortsDetectedMsg struct {
	Ports []int
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Error error
}

// DetectPorts detects open ports on the remote host
func DetectPorts(host SSHHost) tea.Cmd {
	return func() tea.Msg {
		ports, err := detectRemotePorts(host)
		if err != nil {
			// Log the error for debugging but don't quit the app
			fmt.Fprintf(os.Stderr, "Debug: Port detection failed for %s: %v\n", host.Name, err)
			// Return empty ports list so user can still use manual port forwarding
			return PortsDetectedMsg{Ports: []int{}}
		}
		fmt.Fprintf(os.Stderr, "Debug: Detected %d ports on %s: %v\n", len(ports), host.Name, ports)
		return PortsDetectedMsg{Ports: ports}
	}
}

// detectRemotePorts connects to the remote host and detects open ports
func detectRemotePorts(host SSHHost) ([]int, error) {
	// Create SSH client configuration
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

	// Add SSH agent authentication if available
	if agentAuth, err := sshAgentAuth(); err == nil {
		config.Auth = append(config.Auth, agentAuth)
	}

	// If no auth methods available, add a dummy one to avoid empty auth
	if len(config.Auth) == 0 {
		config.Auth = append(config.Auth, ssh.PasswordCallback(func() (string, error) {
			return "", fmt.Errorf("no authentication methods available")
		}))
	}

	// Connect to the remote host
	addr := net.JoinHostPort(host.Hostname, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s (%s): %w", host.Name, addr, err)
	}
	defer client.Close()

	// Run netstat command to detect listening ports
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Try different commands to detect listening ports
	commands := []string{
		"netstat -tlnp 2>/dev/null | grep LISTEN | awk '{print $4}' | cut -d: -f2 | sort -n | uniq",
		"ss -tlnp 2>/dev/null | grep LISTEN | awk '{print $4}' | cut -d: -f2 | sort -n | uniq",
		"lsof -i -P -n 2>/dev/null | grep LISTEN | awk '{print $9}' | cut -d: -f2 | sort -n | uniq",
	}

	var output []byte
	for _, cmd := range commands {
		session, err = client.NewSession()
		if err != nil {
			continue
		}
		
		output, err = session.Output(cmd)
		session.Close()
		
		if err == nil && len(output) > 0 {
			break
		}
	}

	if err != nil || len(output) == 0 {
		// Fallback: try common ports
		return detectCommonPorts(client), nil
	}

	// Parse the output to extract port numbers
	ports := make([]int, 0)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		port, err := strconv.Atoi(line)
		if err == nil && port > 0 && port < 65536 {
			ports = append(ports, port)
		}
	}

	// Remove duplicates and sort
	ports = removeDuplicates(ports)
	sort.Ints(ports)

	return ports, nil
}

// detectCommonPorts tries to detect common ports by attempting connections
func detectCommonPorts(client *ssh.Client) []int {
	commonPorts := []int{80, 443, 3000, 3001, 4000, 5000, 8000, 8080, 8443, 9000}
	var openPorts []int

	for _, port := range commonPorts {
		// Try to create a connection to the port through the SSH tunnel
		conn, err := client.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			conn.Close()
			openPorts = append(openPorts, port)
		}
	}

	return openPorts
}

// loadPrivateKey loads a private key from file
func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	// Expand tilde to home directory
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := getHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = strings.Replace(keyPath, "~", homeDir, 1)
	}

	keyBytes, err := readFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}

// getHomeDir returns the user's home directory
func getHomeDir() (string, error) {
	return os.UserHomeDir()
}

// readFile reads a file and returns its contents
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// removeDuplicates removes duplicate integers from a slice
func removeDuplicates(slice []int) []int {
	keys := make(map[int]bool)
	result := make([]int, 0)
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}