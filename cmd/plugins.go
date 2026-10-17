package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/generator"
	"github.com/karurosux/go-on-rails/internal/project"
)

// bestEffortRoot returns the app root, or "" when not inside a gor app.
func bestEffortRoot() string {
	root, err := project.Root()
	if err != nil {
		return ""
	}
	return root
}

func init() {
	var installProject bool
	installCmd := &cobra.Command{
		Use:   "install <git-url|path>",
		Short: "Install a generator plugin (user-wide, or --project to ship it with the repo)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := generator.InstallPlugin(args[0], installProject)
			if err != nil {
				return err
			}
			fmt.Printf("Installed plugin %q at %s\nRun it with: gor g %s\n",
				p.Manifest.Name, p.Dir, p.Manifest.Name)
			fmt.Println("\nNote: a plugin writes files into your apps when run — install only sources you trust.")
			return nil
		},
	}
	installCmd.Flags().BoolVar(&installProject, "project", false, "install into ./.gor/plugins (ships with the repo)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed generator plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			plugins, err := generator.LoadPlugins(bestEffortRoot())
			if err != nil {
				return err
			}
			if len(plugins) == 0 {
				fmt.Println("No plugins installed. Install one with: gor plugins install <git-url|path>")
				return nil
			}
			for _, p := range plugins {
				fmt.Printf("  %-16s %s\n      %s\n", p.Manifest.Name, p.Manifest.Description, p.Dir)
			}
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var dirs []string
			if root := bestEffortRoot(); root != "" {
				dirs = append(dirs, filepath.Join(root, ".gor", "plugins", name))
			}
			if userDir, err := generator.UserPluginsDir(); err == nil {
				dirs = append(dirs, filepath.Join(userDir, name))
			}
			for _, dir := range dirs {
				if _, err := os.Stat(dir); err == nil {
					if err := os.RemoveAll(dir); err != nil {
						return err
					}
					fmt.Printf("Removed plugin %q (%s)\n", name, dir)
					return nil
				}
			}
			return fmt.Errorf("plugin %q is not installed", name)
		},
	}

	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Install and manage generator plugins",
	}
	pluginsCmd.AddCommand(installCmd)
	pluginsCmd.AddCommand(listCmd)
	pluginsCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(pluginsCmd)
}
