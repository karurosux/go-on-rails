// Package cmd defines the gor CLI. Commands stay thin; logic lives in
// internal packages.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "gor",
	Short:         "Go on Rails — scaffold full-stack Go + React apps",
	Long:          "gor scaffolds and generates full-stack apps: a Go API (Echo, pgx, sqlc, goose)\nwith an embedded React + Vite + TypeScript frontend, built into one binary.",
	Version:       "0.3.0",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
