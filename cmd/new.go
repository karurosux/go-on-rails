package cmd

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/scaffold"
)

var appNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

func init() {
	var module string
	var skipInstall bool

	newCmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new full-stack app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !appNameRe.MatchString(name) {
				return fmt.Errorf("app name %q must be lowercase letters, digits, - or _, starting with a letter", name)
			}
			return scaffold.New(scaffold.Options{
				Name:        name,
				Module:      module,
				SkipInstall: skipInstall,
			})
		},
	}
	newCmd.Flags().StringVar(&module, "module", "", "Go module path (default: the app name)")
	newCmd.Flags().BoolVar(&skipInstall, "skip-install", false, "skip go mod tidy, npm install, and git init")
	rootCmd.AddCommand(newCmd)
}
