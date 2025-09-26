package main

import (
	"fmt"
	"os"
)

func main() {
	// Check for test mode
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		testMode()
		return
	}
	
	// Check for connection test mode
	if len(os.Args) > 2 && os.Args[1] == "--test-connect" {
		testConnection(os.Args[2])
		return
	}
	
	// Initialize the application
	app := NewApp()
	if err := app.Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}

// testMode runs a simple test without TUI
func testMode() {
	fmt.Println("kport - SSH Port Forwarder - Test Mode")
	fmt.Println("======================================")
	
	// Test SSH config loading
	config := NewSSHConfig()
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("❌ Failed to load SSH config: %v\n", err)
		return
	}
	
	hosts := config.GetHosts()
	fmt.Printf("✅ Successfully loaded SSH config with %d hosts:\n\n", len(hosts))
	
	for i, host := range hosts {
		fmt.Printf("%d. %s\n", i+1, host.Name)
		fmt.Printf("   Host: %s\n", host.Hostname)
		fmt.Printf("   User: %s\n", host.User)
		fmt.Printf("   Port: %s\n", host.Port)
		if host.Identity != "" {
			fmt.Printf("   Identity: %s\n", host.Identity)
		}
		fmt.Println()
	}
	
	fmt.Println("📝 Note: The example hosts above are not real servers.")
	fmt.Println("   Replace them in ~/.ssh/config with your actual SSH hosts.")
	fmt.Println("")
	fmt.Println("To run the interactive TUI, use: ./kport")
	fmt.Println("Note: TUI requires a proper terminal environment")
	fmt.Println("")
	fmt.Println("To test connection to a specific host: ./kport --test-connect <hostname>")
}

// testConnection tests connecting to a specific host
func testConnection(hostName string) {
	fmt.Printf("Testing connection to host: %s\n", hostName)
	fmt.Println("=====================================")
	
	// Load SSH config
	config := NewSSHConfig()
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("❌ Failed to load SSH config: %v\n", err)
		return
	}
	
	// Find the host
	host, err := config.GetHostByName(hostName)
	if err != nil {
		fmt.Printf("❌ Host not found: %v\n", err)
		return
	}
	
	fmt.Printf("Found host configuration:\n")
	fmt.Printf("  Name: %s\n", host.Name)
	fmt.Printf("  Hostname: %s\n", host.Hostname)
	fmt.Printf("  User: %s\n", host.User)
	fmt.Printf("  Port: %s\n", host.Port)
	if host.Identity != "" {
		fmt.Printf("  Identity: %s\n", host.Identity)
	}
	fmt.Println("")
	
	// Test port detection
	fmt.Println("Testing port detection...")
	ports, err := detectRemotePorts(*host)
	if err != nil {
		fmt.Printf("❌ Port detection failed: %v\n", err)
		fmt.Println("")
		fmt.Println("This is expected if:")
		fmt.Println("- The host is not reachable")
		fmt.Println("- SSH keys are not set up")
		fmt.Println("- SSH agent is not running")
		fmt.Println("- The host doesn't exist")
	} else {
		fmt.Printf("✅ Port detection successful! Found %d ports: %v\n", len(ports), ports)
	}
	
	fmt.Println("")
	fmt.Println("You can still use manual port forwarding in the TUI even if port detection fails.")
}