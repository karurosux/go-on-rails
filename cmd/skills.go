package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/templates"
)

func init() {
	var project bool

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the gor skills for AI coding agents (Claude Code)",
		Long: "Install the gor skills so AI coding agents know how to use gor's\n" +
			"generators (gor) and how to author generator plugins (gor-plugins).\n\n" +
			"By default they install user-wide (~/.claude/skills); --project installs\n" +
			"into the current project's .claude/skills instead, so they ship with the repo.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var base string
			if project {
				base = filepath.Join(".claude", "skills")
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				base = filepath.Join(home, ".claude", "skills")
			}

			skills := []struct{ source, name string }{
				{"skill/SKILL.md", "gor"},
				{"skill/PLUGIN_SKILL.md", "gor-plugins"},
			}
			for _, s := range skills {
				content, err := templates.FS.ReadFile(s.source)
				if err != nil {
					return err
				}
				dir := filepath.Join(base, s.name)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
				path := filepath.Join(dir, "SKILL.md")
				if err := os.WriteFile(path, content, 0o644); err != nil {
					return err
				}
				fmt.Printf("Installed skill %q at %s\n", s.name, path)
			}
			fmt.Println("\nAI agents with skill support (e.g. Claude Code) can now use gor's")
			fmt.Println("generators and author gor plugins. Re-run after upgrading gor.")
			return nil
		},
	}
	installCmd.Flags().BoolVar(&project, "project", false, "install into ./.claude/skills (ships with the repo)")

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "AI agent integration (install the gor skills)",
	}
	skillsCmd.AddCommand(installCmd)
	rootCmd.AddCommand(skillsCmd)
}
