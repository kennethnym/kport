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
}