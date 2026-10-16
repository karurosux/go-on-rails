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
	var image bool
	var tag string

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build one production binary with the frontend embedded (--image for a Docker image)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			app, err := project.AppName(root)
			if err != nil {
				return err
			}

			if tag != "" {
				image = true
			}
			if image {
				if err := deps.Check("docker"); err != nil {
					return err
				}
				if tag == "" {
					tag = app + ":latest"
				}
				fmt.Printf("Building image %s (docker build)...\n", tag)
				if err := run.Command(root, "docker", "build", "-t", tag, "."); err != nil {
					return err
				}
				fmt.Printf(`
Done: image %[1]s — run it with:
  docker run -p 8080:8080 -e DATABASE_URL=postgres://... %[1]s
(for a Postgres on this machine, use host.docker.internal as the host)
`, tag)
				return nil
			}

			if err := deps.Check("go", "node", "npm"); err != nil {
				return err
			}

			fmt.Println("Building frontend (npm run build)...")
			if err := run.Command(filepath.Join(root, "web"), "npm", "run", "build"); err != nil {
				return err
			}

			out := filepath.Join("bin", app)
			fmt.Printf("Building backend (go build -o %s)...\n", out)
			if err := run.Command(root, "go", "build", "-o", out, "./cmd/server"); err != nil {
				return err
			}
			if render.Exists(filepath.Join(root, "cmd", "worker")) {
				workerOut := filepath.Join("bin", app+"-worker")
				fmt.Printf("Building jobs worker (go build -o %s)...\n", workerOut)
				if err := run.Command(root, "go", "build", "-o", workerOut, "./cmd/worker"); err != nil {
					return err
				}
			}

			fmt.Printf("\nDone: %s/%s — run it with:\n  APP_ENV=production DATABASE_URL=... ./%s\n", root, out, out)
			return nil
		},
	}
	buildCmd.Flags().BoolVar(&image, "image", false, "build a Docker image instead of a local binary")
	buildCmd.Flags().StringVar(&tag, "tag", "", "image tag (default <app>:latest, implies --image)")
	rootCmd.AddCommand(buildCmd)
}
