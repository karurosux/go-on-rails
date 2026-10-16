package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/deps"
	"github.com/karurosux/go-on-rails/internal/generator"
	"github.com/karurosux/go-on-rails/internal/project"
	"github.com/karurosux/go-on-rails/internal/render"
	"github.com/karurosux/go-on-rails/internal/run"
)

// goose runs a goose subcommand against the app's migrations and DATABASE_URL.
func goose(root string, args ...string) error {
	if err := deps.Check("goose"); err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "up" && !hasMigrations(root) {
		fmt.Println("No migrations yet — nothing to apply. Generate one with `gor g resource` or `gor g migration`.")
		return nil
	}
	url, err := project.DatabaseURL(root)
	if err != nil {
		return err
	}
	full := append([]string{"-dir", "migrations", "postgres", url}, args...)
	return run.Command(root, "goose", full...)
}

func init() {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Database commands (goose + Docker)",
	}

	dbCmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Apply pending migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			return goose(root, "up")
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "rollback",
		Short: "Roll back the most recent migration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			return goose(root, "down")
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			return goose(root, "status")
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "new <name>",
		Short: "Create an empty migration (see also: gor g migration)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			return generator.GenerateMigration(root, "", args[0], nil)
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "seed",
		Short: "Run the app's seeds (cmd/seed)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			seedMain := root + "/cmd/seed/main.go"
			if !render.Exists(seedMain) {
				// Apps generated before seeds existed get the file on demand.
				module, err := project.Module(root)
				if err != nil {
					return err
				}
				content, err := render.File("project/cmd/seed/main.go.tmpl", struct{ Module string }{module})
				if err != nil {
					return err
				}
				if err := render.Write(seedMain, content, false); err != nil {
					return err
				}
				fmt.Println("  create  cmd/seed/main.go (add your seed data there)")
			}
			return run.Command(root, "go", "run", "./cmd/seed")
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "gen",
		Short: "Regenerate the Go db code (sqlc) from migrations/ and queries/",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			if err := deps.Check("sqlc"); err != nil {
				return err
			}
			fmt.Println("  run     sqlc generate")
			if err := run.CommandQuiet(root, "sqlc", "generate"); err != nil {
				return fmt.Errorf("sqlc generate failed: %w", err)
			}
			fmt.Println("  run     go mod tidy")
			if err := run.CommandQuiet(root, "go", "mod", "tidy"); err != nil {
				return fmt.Errorf("go mod tidy failed: %w", err)
			}
			fmt.Println("internal/db/gen is up to date.")
			return nil
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "setup",
		Short: "Start Postgres via Docker (if available) and run migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			return dbSetup(root)
		},
	})

	dbCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Destroy the dev database (docker compose down -v) and set it up fresh",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			if !render.Exists(root+"/docker-compose.yml") || !deps.Have("docker") {
				return fmt.Errorf("gor db reset needs the app's docker-compose.yml and Docker\nWithout Docker, drop and recreate the database yourself, then run `gor db migrate`")
			}
			fmt.Println("Destroying the dev database (docker compose down -v)...")
			if err := run.Command(root, "docker", "compose", "down", "-v"); err != nil {
				return err
			}
			return dbSetup(root)
		},
	})

	rootCmd.AddCommand(dbCmd)
}

// hasMigrations reports whether the app has any .sql migrations.
func hasMigrations(root string) bool {
	entries, err := os.ReadDir(root + "/migrations")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			return true
		}
	}
	return false
}

// dbSetup starts the Docker Postgres when available and migrates.
func dbSetup(root string) error {
	if render.Exists(root+"/docker-compose.yml") && deps.Have("docker") {
		fmt.Println("Starting Postgres (docker compose up -d)...")
		if err := run.Command(root, "docker", "compose", "up", "-d", "--wait"); err != nil {
			return fmt.Errorf("docker compose failed: %w\nIf you don't use Docker, start Postgres yourself and run `gor db migrate`", err)
		}
		if err := waitForPostgres(root); err != nil {
			return err
		}
	} else {
		fmt.Println("No Docker found (or no docker-compose.yml) — assuming Postgres is already running.")
	}
	if err := goose(root, "up"); err != nil {
		return fmt.Errorf("%w\n\nIf the database holds tables from an older version of this app (a stale\nDocker volume, e.g. \"relation already exists\"), wipe it with: gor db reset", err)
	}
	return nil
}

// waitForPostgres polls pg_isready inside the db container. `compose up
// --wait` normally covers this; the loop guards older Docker versions.
func waitForPostgres(root string) error {
	for i := 0; i < 30; i++ {
		cmd := exec.Command("docker", "compose", "exec", "-T", "db", "pg_isready", "-U", "postgres")
		cmd.Dir = root
		if cmd.Run() == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("Postgres did not become ready — check `docker compose logs db`")
}
