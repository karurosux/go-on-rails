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
	var installProject, installLink bool
	installCmd := &cobra.Command{
		Use:   "install <git-url|path>",
		Short: "Install a generator plugin (user-wide, or --project to ship it with the repo)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := generator.InstallPlugin(args[0], installProject, installLink)
			if err != nil {
				return err
			}
			mode := "Installed"
			if installLink {
				mode = "Linked (dev install — edits are live)"
			}
			fmt.Printf("%s plugin %q at %s\nRun it with: gor g %s\n",
				mode, p.Manifest.Name, p.Dir, p.Manifest.Name)
			fmt.Println("\nNote: a plugin writes files into your apps when run — install only sources you trust.")
			return nil
		},
	}
	installCmd.Flags().BoolVar(&installProject, "project", false, "install into ./.gor/plugins (ships with the repo)")
	installCmd.Flags().BoolVar(&installLink, "link", false, "symlink a local plugin for development — edits apply without reinstalling")

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
				location := p.Dir
				if target, err := os.Readlink(p.Dir); err == nil {
					location = p.Dir + " -> " + target + " (linked)"
				}
				fmt.Printf("  %-16s %s\n      %s\n", p.Manifest.Name, p.Manifest.Description, location)
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

	newCmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Scaffold a new generator plugin in ./gor-<name>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.PluginNew(args[0])
		},
	}

	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Install and manage generator plugins",
	}
	pluginsCmd.AddCommand(newCmd)
	pluginsCmd.AddCommand(installCmd)
	pluginsCmd.AddCommand(listCmd)
	pluginsCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(pluginsCmd)
}
