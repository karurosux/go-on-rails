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
		Short: "Install the gor skill for AI coding agents (Claude Code)",
		Long: "Install the gor skill so AI coding agents know to use gor's generators\n" +
			"instead of hand-writing boilerplate.\n\n" +
			"By default it installs user-wide (~/.claude/skills/gor); --project\n" +
			"installs into the current project's .claude/skills/gor instead, so the\n" +
			"skill ships with the repo.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := templates.FS.ReadFile("skill/SKILL.md")
			if err != nil {
				return err
			}

			var dir string
			if project {
				dir = filepath.Join(".claude", "skills", "gor")
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				dir = filepath.Join(home, ".claude", "skills", "gor")
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			path := filepath.Join(dir, "SKILL.md")
			if err := os.WriteFile(path, content, 0o644); err != nil {
				return err
			}

			fmt.Printf("Installed the gor skill at %s\n", path)
			fmt.Println("AI agents with skill support (e.g. Claude Code) will now reach for")
			fmt.Println("gor's generators when working on gor apps. Re-run after upgrading gor.")
			return nil
		},
	}
	installCmd.Flags().BoolVar(&project, "project", false, "install into ./.claude/skills/gor (ships with the repo)")

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "AI agent integration (install the gor skill)",
	}
	skillsCmd.AddCommand(installCmd)
	rootCmd.AddCommand(skillsCmd)
}
