package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SSHHost represents an SSH host configuration
type SSHHost struct {
	Name     string
	Hostname string
	User     string
	Port     string
	Identity string
}

// SSHConfig handles parsing SSH configuration
type SSHConfig struct {
	Hosts []SSHHost
}

// NewSSHConfig creates a new SSH config parser
func NewSSHConfig() *SSHConfig {
	return &SSHConfig{
		Hosts: make([]SSHHost, 0),
	}
}

// LoadConfig loads SSH configuration from the default location
func (sc *SSHConfig) LoadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")
	return sc.LoadConfigFromFile(configPath)
}

// LoadConfigFromFile loads SSH configuration from a specific file
func (sc *SSHConfig) LoadConfigFromFile(path string) error {
	return sc.loadConfigFromFileRecursive(path, make(map[string]bool))
}

// loadConfigFromFileRecursive loads SSH config with include support and cycle detection
func (sc *SSHConfig) loadConfigFromFileRecursive(path string, visited map[string]bool) error {
	// Resolve absolute path to detect cycles
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	// Check for cycles
	if visited[absPath] {
		return fmt.Errorf("circular include detected: %s", absPath)
	}
	visited[absPath] = true
	defer delete(visited, absPath)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open SSH config file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentHost *SSHHost

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		switch key {
		case "include":
			// Handle include directive
			if err := sc.processInclude(value, visited); err != nil {
				// Log error but continue processing
				fmt.Fprintf(os.Stderr, "Warning: failed to process include %s: %v\n", value, err)
			}
		case "host":
			// Save previous host if exists
			if currentHost != nil {
				sc.Hosts = append(sc.Hosts, *currentHost)
			}
			// Start new host
			currentHost = &SSHHost{
				Name: value,
				Port: "22", // default port
			}
		case "hostname":
			if currentHost != nil {
				currentHost.Hostname = value
			}
		case "user":
			if currentHost != nil {
				currentHost.User = value
			}
		case "port":
			if currentHost != nil {
				currentHost.Port = value
			}
		case "identityfile":
			if currentHost != nil {
				currentHost.Identity = value
			}
		}
	}

	// Add the last host
	if currentHost != nil {
		sc.Hosts = append(sc.Hosts, *currentHost)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading SSH config file %s: %w", path, err)
	}

	return nil
}

// processInclude handles SSH config include directives
func (sc *SSHConfig) processInclude(pattern string, visited map[string]bool) error {
	// Expand tilde to home directory
	if strings.HasPrefix(pattern, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		pattern = filepath.Join(homeDir, pattern[2:])
	}

	// Handle glob patterns
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
	}

	// Process each matching file
	for _, match := range matches {
		// Skip directories
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			continue
		}

		// Recursively load the included file
		if err := sc.loadConfigFromFileRecursive(match, visited); err != nil {
			return fmt.Errorf("failed to load included file %s: %w", match, err)
		}
	}

	return nil
}

// GetHosts returns all configured SSH hosts
func (sc *SSHConfig) GetHosts() []SSHHost {
	return sc.Hosts
}

// GetHostByName returns a specific host by name
func (sc *SSHConfig) GetHostByName(name string) (*SSHHost, error) {
	for _, host := range sc.Hosts {
		if host.Name == name {
			return &host, nil
		}
	}
	return nil, fmt.Errorf("host '%s' not found", name)
}