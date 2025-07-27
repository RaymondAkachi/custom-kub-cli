package main

import (
	"fmt"
	"os"

	"github.com/RaymondAkachi/custom-kub-cli/internal/config"
	"github.com/RaymondAkachi/custom-kub-cli/internal/system"
	"github.com/RaymondAkachi/custom-kub-cli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Check system dependencies first
	dc := system.NewDependencyChecker()

	// Check all dependencies
	if err := dc.CheckAll(); err != nil {
		fmt.Println(err)
	}

	// Get version of kubectl
	version, err := dc.GetVersion("kubectl")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("kubectl version:", version)
	}


	// Initialize configuration
	cfg, err := config.Initialize()
	if err != nil {
		fmt.Printf("❌ Failed to initialize configuration: %v\n", err)
		os.Exit(1)
	}

	// Create and run the TUI application
	app, err := ui.NewApplication(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to create application: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🚀 Starting Kubernetes Orchestrator...")
	fmt.Println("✅ All dependencies verified")
	
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Application error: %v\n", err)
		os.Exit(1)
	}
}