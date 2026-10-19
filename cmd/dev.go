package cmd

import (
	"fmt"
	"path/filepath"
	"time"

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
			port := project.Port(root)
			// A taken port would not fail `gor dev` itself: Air's server dies
			// on the bind while Vite proxies /api to whatever squats there,
			// and every request 404s with no hint why. Refuse up front instead.
			if err := run.CheckPortFree(port); err != nil {
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
			fmt.Printf("Starting dev servers — API :%s, web :5173%s (Ctrl+C to stop)\n", port, withWorker)
			// Air stays alive even when the server it launched exited at
			// startup, so warn when nothing binds the API port.
			run.WatchPort(port, 60*time.Second, func() {
				fmt.Printf("[gor] nothing is listening on :%s after 60s — the API server likely failed to start; check the [go] output above\n", port)
			})
			return run.Procs(procs)
		},
	}
	rootCmd.AddCommand(devCmd)
}
