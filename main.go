package main

import (
	"fmt"
	"os"

	commands "github.com/luisdavim/termux-docker/cmd"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Fatal error: %v\n", err)
		os.Exit(1)
	}

	rootCmd := commands.NewRootCmd(homeDir)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
}
