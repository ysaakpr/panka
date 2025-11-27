package main

import (
	"fmt"
	"os"
)

var (
	// Version is set during build
	Version = "dev"
	// BuildTime is set during build
	BuildTime = "unknown"
	// GitCommit is set during build
	GitCommit = "unknown"
)

func main() {
	fmt.Printf("Panka v%s\n", Version)
	fmt.Printf("Build: %s (%s)\n", BuildTime, GitCommit)
	fmt.Println("\n🚀 Multi-tenant AWS deployment CLI tool")
	fmt.Println("\nUsage: panka <command> [options]")
	fmt.Println("\nAvailable commands will be added as we implement them...")
	
	os.Exit(0)
}

