package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/deps"
	"github.com/karurosux/go-on-rails/internal/project"
	"github.com/karurosux/go-on-rails/internal/render"
	"github.com/karurosux/go-on-rails/internal/run"
)

func init() {
	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Run the Go server (Air) and the Vite dev server together",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			if err := deps.Check("air", "sqlc", "node", "npm"); err != nil {
				return err
			}
			procs := []run.Proc{
				{
					Name:  "go",
					Color: run.ColorCyan,
					Dir:   root,
					Cmd:   "air",
				},
				{
					Name:  "web",
					Color: run.ColorMagenta,
					Dir:   filepath.Join(root, "web"),
					Env:   []string{"FORCE_COLOR=1"},
					Cmd:   "npm",
					Args:  []string{"run", "dev"},
				},
			}
			withWorker := ""
			if render.Exists(filepath.Join(root, "cmd", "worker")) {
				procs = append(procs, run.Proc{
					Name:  "job",
					Color: run.ColorYellow,
					Dir:   root,
					Cmd:   "air",
					Args:  []string{"-c", ".air.worker.toml"},
				})
				withWorker = ", jobs worker"
			}
			fmt.Printf("Starting dev servers — API :%s, web :5173%s (Ctrl+C to stop)\n", project.Port(root), withWorker)
			return run.Procs(procs)
		},
	}
	rootCmd.AddCommand(devCmd)
}
