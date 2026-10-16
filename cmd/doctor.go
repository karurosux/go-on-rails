package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/deps"
)

func init() {
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check that every tool gor relies on is installed",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			required := []string{"go", "node", "npm", "sqlc", "goose", "air", "git"}
			optional := []string{"docker"}

			allGood := true
			for _, name := range required {
				allGood = report(name, false) && allGood
			}
			for _, name := range optional {
				report(name, true)
			}

			if allGood {
				fmt.Println("\nEverything gor needs is installed.")
			} else {
				fmt.Println("\nInstall the missing tools above, then re-run `gor doctor`.")
			}
		},
	}
	rootCmd.AddCommand(doctorCmd)
}

// report prints one tool's status and returns whether it is present.
func report(name string, optional bool) bool {
	tool, _ := deps.Known(name)
	if deps.Have(name) {
		fmt.Printf("  ok       %-7s %s\n", name, tool.Why)
		return true
	}
	label := "MISSING"
	if optional {
		label = "absent "
	}
	fmt.Printf("  %s  %-7s %s\n           install: %s\n", label, name, tool.Why, tool.Install)
	return false
}
