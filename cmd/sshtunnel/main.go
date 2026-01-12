package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/tunnel"
	"github.com/labbs/sshtunnel/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	initConfig := flag.Bool("init", false, "Create a default configuration file")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("SSH Tunnel Manager %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		os.Exit(0)
	}

	// Determine config path
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	// Initialize config
	if *initConfig {
		if err := config.CreateDefaultConfig(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration file created at: %s\n", cfgPath)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRun with --init to create a default configuration file.\n")
		os.Exit(1)
	}

	// Create tunnel manager
	manager := tunnel.NewManager(cfg)
	if err := manager.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tunnels: %v\n", err)
		os.Exit(1)
	}

	// Start auto-start tunnels
	if err := manager.StartAutoStart(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting auto-start tunnels: %v\n", err)
	}

	// Run cleanup on exit
	defer func() {
		fmt.Println("\nShutting down...")
		manager.Shutdown(10 * time.Second)
	}()

	// Run the TUI (handles signals internally via BubbleTea)
	if err := ui.Run(manager, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}
}
