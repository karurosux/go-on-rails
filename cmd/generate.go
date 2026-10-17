package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/generator"
	"github.com/karurosux/go-on-rails/internal/project"
)

var resourceNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func init() {
	generateCmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"g"},
		Short:   "Generate code (resource)",
	}

	var service, apiOnly, protected, live, headless, force bool
	resourceCmd := &cobra.Command{
		Use:   "resource <name> <field:type>...",
		Short: "Generate a full CRUD resource (migration, queries, handlers, React pages)",
		Long: "Generate a full CRUD resource.\n\nField types: " + strings.Join(fields.TypeNames(), ", ") +
			"\n\nExample:\n  gor g resource post title:string body:text published:bool author:references",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !resourceNameRe.MatchString(name) {
				return fmt.Errorf("resource name %q must be lower_snake_case, starting with a letter", name)
			}
			if strings.HasSuffix(name, "s") {
				fmt.Printf("note: resource names should be singular (got %q) — gor pluralizes for you\n", name)
			}
			parsed, err := fields.Parse(args[1:])
			if err != nil {
				return err
			}
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			// Per-app defaults (.gor/config.json); explicit flags win.
			cfg, err := generator.LoadConfig(root)
			if err != nil {
				return err
			}
			flagOr := func(flag string, value, fallback bool) bool {
				if cmd.Flags().Changed(flag) {
					return value
				}
				return fallback
			}
			return generator.Generate(root, module, generator.Options{
				Name:      name,
				Fields:    parsed,
				Service:   flagOr("service", service, cfg.Resource.Service),
				APIOnly:   flagOr("api-only", apiOnly, cfg.Resource.APIOnly),
				Protected: flagOr("protected", protected, cfg.Resource.Protected),
				Live:      flagOr("live", live, cfg.Resource.Live),
				Headless:  flagOr("headless", headless, cfg.Resource.Headless),
				Force:     force,
			})
		},
	}
	resourceCmd.Flags().BoolVar(&service, "service", false, "generate a service layer between handlers and the database")
	resourceCmd.Flags().BoolVar(&apiOnly, "api-only", false, "skip frontend generation")
	resourceCmd.Flags().BoolVar(&protected, "protected", false, "require a signed-in user (needs `gor g auth` first)")
	resourceCmd.Flags().BoolVar(&live, "live", false, "publish mutations + live-updating pages (needs `gor g realtime` first)")
	resourceCmd.Flags().BoolVar(&headless, "headless", false, "frontend building blocks only: schema, hooks, props-driven Form — no pages or routes")
	resourceCmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")

	migrationCmd := &cobra.Command{
		Use:   "migration <name> [field:type...]",
		Short: "Generate a goose migration (add_x_to_posts / remove_x_from_posts generate the SQL)",
		Long: "Generate a timestamped goose migration.\n\n" +
			"With fields and a conventional name, the column SQL is written for you:\n" +
			"  gor g migration add_slug_to_posts slug:string\n" +
			"  gor g migration remove_views_from_posts views:int\n\n" +
			"Any other name creates an empty migration skeleton:\n" +
			"  gor g migration backfill_post_slugs",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.GenerateMigration(root, module, args[0], args[1:])
		},
	}

	var syncLive, syncNoLive, syncProtected, syncNoProtected bool
	syncCmd := &cobra.Command{
		Use:   "sync <resource>",
		Short: "Regenerate a resource's derived files (and toggle --live/--protected)",
		Long: "Regenerate queries, handlers, and frontend files for a resource from the\n" +
			"field list recorded in .gor/manifest.json. Files you have edited by hand\n" +
			"are left alone and listed instead.\n\n" +
			"Toggle situational flags on an existing resource:\n" +
			"  gor g sync post --live          # make it realtime\n" +
			"  gor g sync post --no-protected  # make it public again",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (syncLive && syncNoLive) || (syncProtected && syncNoProtected) {
				return fmt.Errorf("pick one of --live/--no-live and --protected/--no-protected")
			}
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			changes := &generator.FlagChanges{}
			if syncLive || syncNoLive {
				v := syncLive
				changes.Live = &v
			}
			if syncProtected || syncNoProtected {
				v := syncProtected
				changes.Protected = &v
			}
			return generator.Sync(root, module, args[0], changes)
		},
	}
	syncCmd.Flags().BoolVar(&syncLive, "live", false, "turn on realtime updates (needs `gor g realtime`)")
	syncCmd.Flags().BoolVar(&syncNoLive, "no-live", false, "turn off realtime updates")
	syncCmd.Flags().BoolVar(&syncProtected, "protected", false, "move behind auth.Require (needs `gor g auth`)")
	syncCmd.Flags().BoolVar(&syncNoProtected, "no-protected", false, "make public again")

	var authAPIOnly, authForce bool
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Generate cookie-session authentication (users, sessions, middleware, roles, login UI)",
		Long: "Generate authentication: users + sessions tables, bcrypt passwords, an auth\n" +
			"package with middleware (auth.Require, auth.RequireRole), REST endpoints\n" +
			"(/api/auth/register|login|logout|me), and login/register pages.\n\n" +
			"Afterwards, generate protected resources with `gor g resource ... --protected`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.Auth(root, module, generator.AuthOptions{
				APIOnly: authAPIOnly,
				Force:   authForce,
			})
		},
	}
	authCmd.Flags().BoolVar(&authAPIOnly, "api-only", false, "skip frontend generation")
	authCmd.Flags().BoolVar(&authForce, "force", false, "overwrite existing files")

	var jobsForce bool
	jobsCmd := &cobra.Command{
		Use:   "jobs",
		Short: "Set up background jobs (River, Postgres-backed) with a worker process",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			app, err := project.AppName(root)
			if err != nil {
				return err
			}
			return generator.Jobs(root, module, app, jobsForce)
		},
	}
	jobsCmd.Flags().BoolVar(&jobsForce, "force", false, "overwrite existing files")

	var jobForce, jobLive bool
	var jobEvery, jobCron string
	jobCmd := &cobra.Command{
		Use:   "job <name>",
		Short: "Generate a background job (needs `gor g jobs` once)",
		Long: "Generate a background job worker, e.g.:\n  gor g job send_welcome_email\n\n" +
			"With --every it also runs on a schedule:\n  gor g job cleanup_sessions --every 1h",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !resourceNameRe.MatchString(name) {
				return fmt.Errorf("job name %q must be lower_snake_case, starting with a letter", name)
			}
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.Job(root, module, name, jobEvery, jobCron, jobLive, jobForce)
		},
	}
	jobCmd.Flags().StringVar(&jobEvery, "every", "", "also run on a schedule (Go duration: 30m, 1h, 24h)")
	jobCmd.Flags().StringVar(&jobCron, "cron", "", `also run on a cron schedule, e.g. "0 9 * * *" (daily 9am)`)
	jobCmd.Flags().BoolVar(&jobLive, "live", false, `stream started/completed/failed to the browser on the "jobs" channel (needs gor g realtime)`)
	jobCmd.Flags().BoolVar(&jobForce, "force", false, "overwrite existing files")

	var realtimeAPIOnly, realtimeForce bool
	realtimeCmd := &cobra.Command{
		Use:   "realtime",
		Short: "Set up realtime server push (SSE + Postgres NOTIFY) with React hooks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.Realtime(root, module, realtimeAPIOnly, realtimeForce)
		},
	}
	realtimeCmd.Flags().BoolVar(&realtimeAPIOnly, "api-only", false, "skip frontend generation")
	realtimeCmd.Flags().BoolVar(&realtimeForce, "force", false, "overwrite existing files")

	var mailerForce bool
	mailerCmd := &cobra.Command{
		Use:   "mailer <name>",
		Short: "Generate a transactional email (SMTP + HTML template + Mailpit dev catcher)",
		Long:  "Generate a transactional email, e.g.:\n  gor g mailer welcome\n\nThe first mailer also sets up internal/mailer and adds Mailpit to\ndocker-compose so dev email is caught locally instead of delivered.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !resourceNameRe.MatchString(name) {
				return fmt.Errorf("mailer name %q must be lower_snake_case, starting with a letter", name)
			}
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			app, err := project.AppName(root)
			if err != nil {
				return err
			}
			return generator.Mailer(root, module, app, name, mailerForce)
		},
	}
	mailerCmd.Flags().BoolVar(&mailerForce, "force", false, "overwrite existing files")

	var uploadsAPIOnly, uploadsForce bool
	uploadsCmd := &cobra.Command{
		Use:   "uploads",
		Short: "Set up file uploads (disk storage, endpoint, frontend upload control)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.Uploads(root, module, uploadsAPIOnly, uploadsForce)
		},
	}
	uploadsCmd.Flags().BoolVar(&uploadsAPIOnly, "api-only", false, "skip frontend generation")
	uploadsCmd.Flags().BoolVar(&uploadsForce, "force", false, "overwrite existing files")

	var authEmailsAPIOnly, authEmailsForce bool
	authEmailsCmd := &cobra.Command{
		Use:   "auth-emails",
		Short: "Generate password reset + email verification (needs auth; sets up mailer)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			app, err := project.AppName(root)
			if err != nil {
				return err
			}
			return generator.AuthEmails(root, module, app, authEmailsAPIOnly, authEmailsForce)
		},
	}
	authEmailsCmd.Flags().BoolVar(&authEmailsAPIOnly, "api-only", false, "skip frontend generation")
	authEmailsCmd.Flags().BoolVar(&authEmailsForce, "force", false, "overwrite existing files")

	var modelForce bool
	modelCmd := &cobra.Command{
		Use:   "model <name> <field:type>...",
		Short: "Generate the data layer only: migration + sqlc queries (no API, no frontend)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !resourceNameRe.MatchString(name) {
				return fmt.Errorf("model name %q must be lower_snake_case, starting with a letter", name)
			}
			parsed, err := fields.Parse(args[1:])
			if err != nil {
				return err
			}
			root, err := project.Root()
			if err != nil {
				return err
			}
			module, err := project.Module(root)
			if err != nil {
				return err
			}
			return generator.Generate(root, module, generator.Options{
				Name:      name,
				Fields:    parsed,
				ModelOnly: true,
				Force:     modelForce,
			})
		},
	}
	modelCmd.Flags().BoolVar(&modelForce, "force", false, "overwrite existing files")

	generateCmd.AddCommand(resourceCmd)
	generateCmd.AddCommand(modelCmd)
	generateCmd.AddCommand(migrationCmd)
	generateCmd.AddCommand(authCmd)
	generateCmd.AddCommand(authEmailsCmd)
	generateCmd.AddCommand(jobsCmd)
	generateCmd.AddCommand(jobCmd)
	generateCmd.AddCommand(realtimeCmd)
	generateCmd.AddCommand(mailerCmd)
	generateCmd.AddCommand(uploadsCmd)
	generateCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(generateCmd)
}
