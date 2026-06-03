package main

import (
	"fmt"
	"os"

	commands "github.com/luisdavim/termux-docker/cmd"
)

func main() {
	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
}
