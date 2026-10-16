package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// Jobs sets up background jobs (River): the jobs package, an enqueue
// helper, and the worker process. Individual jobs come from `gor g job`.
func Jobs(root, module, appName string, force bool) error {
	data := struct{ Module, AppName string }{Module: module, AppName: appName}

	files := []plannedFile{
		{"jobs/jobs.go.tmpl", filepath.Join("internal", "jobs", "jobs.go")},
		{"jobs/client.go.tmpl", filepath.Join("internal", "jobs", "client.go")},
		{"jobs/worker_main.go.tmpl", filepath.Join("cmd", "worker", "main.go")},
		{"jobs/air.worker.toml.tmpl", ".air.worker.toml"},
	}

	if !force {
		var conflicts []string
		for _, f := range files {
			if render.Exists(filepath.Join(root, f.path)) {
				conflicts = append(conflicts, f.path)
			}
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("these files already exist (use --force to overwrite):\n  %s",
				strings.Join(conflicts, "\n  "))
		}
	}

	for _, f := range files {
		content, err := render.File(f.template, data)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(root, f.path), content, force); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", f.path)
	}

	if err := wireWorkerDockerfile(root, appName); err != nil {
		return err
	}

	fmt.Println("  run     go mod tidy")
	if err := tidy(root); err != nil {
		return err
	}

	fmt.Print(`
Background jobs set up (River, Postgres-backed — no extra services).

  Generate a job:   gor g job send_welcome_email
  Enqueue one:      jobs.Enqueue(ctx, jobs.SendWelcomeEmailArgs{...})
  Run the worker:   gor dev runs it automatically; production runs
                    bin/` + appName + `-worker (built by gor build)

River's queue tables are migrated automatically when the worker starts.
`)
	return nil
}

// wireWorkerDockerfile adds the worker binary to the production image, so
// one image runs both processes (the worker via --entrypoint).
func wireWorkerDockerfile(root, appName string) error {
	path := filepath.Join(root, "Dockerfile")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil // no Dockerfile — nothing to wire
	}
	if strings.Contains(string(raw), appName+"-worker") {
		return nil
	}
	if !strings.Contains(string(raw), "# gor:docker-build") {
		fmt.Printf(`
Your Dockerfile predates the # gor:docker-build marker — add the worker
binary yourself:

  (build stage)   RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /%[1]s-worker ./cmd/worker
  (runtime stage) COPY --from=build /%[1]s-worker /%[1]s-worker
`, appName)
		return nil
	}
	buildLine := fmt.Sprintf(`RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /%s-worker ./cmd/worker`, appName)
	if _, err := inject.InsertBefore(path, "# gor:docker-build", buildLine); err != nil {
		return err
	}
	copyLine := fmt.Sprintf("COPY --from=build /%[1]s-worker /%[1]s-worker", appName)
	changed, err := inject.InsertBefore(path, "# gor:docker-copy", copyLine)
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  Dockerfile (worker binary in the production image)")
	}
	return nil
}

// Job generates one background job and registers its worker. A non-empty
// every (a Go duration like "1h") or cronExpr (standard cron syntax) also
// schedules it periodically.
func Job(root, module, name, every, cronExpr string, live, force bool) error {
	if !render.Exists(filepath.Join(root, "internal", "jobs", "jobs.go")) {
		return fmt.Errorf("background jobs are not set up yet: run `gor g jobs` first")
	}
	if live && !HasRealtime(root) {
		return fmt.Errorf("--live needs realtime set up first: run `gor g realtime`")
	}
	if every != "" && cronExpr != "" {
		return fmt.Errorf("use either --every or --cron, not both")
	}
	if every != "" {
		if _, err := time.ParseDuration(every); err != nil {
			return fmt.Errorf("invalid --every interval %q: use a Go duration like 30m, 1h, or 24h", every)
		}
	}
	if cronExpr != "" {
		if _, err := cron.ParseStandard(cronExpr); err != nil {
			return fmt.Errorf("invalid --cron expression %q: %w (5-field syntax, e.g. \"0 9 * * *\")", cronExpr, err)
		}
	}

	data := struct {
		Module string
		Name   string
		Pascal string
		Every  string
		Cron   string
		Live   bool
	}{Module: module, Name: name, Pascal: fields.Pascal(name), Every: every, Cron: cronExpr, Live: live}

	files := []plannedFile{
		{"jobs/job.go.tmpl", filepath.Join("internal", "jobs", name+".go")},
		{"jobs/job_test.go.tmpl", filepath.Join("internal", "jobs", name+"_test.go")},
	}
	if !force {
		var conflicts []string
		for _, f := range files {
			if render.Exists(filepath.Join(root, f.path)) {
				conflicts = append(conflicts, f.path)
			}
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("these files already exist (use --force to overwrite):\n  %s",
				strings.Join(conflicts, "\n  "))
		}
	}

	for _, f := range files {
		content, err := render.File(f.template, data)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(root, f.path), content, force); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", f.path)
	}

	registerLine, err := render.File("jobs/snippets/worker_register.tmpl", data)
	if err != nil {
		return err
	}
	changed, err := inject.InsertBefore(
		filepath.Join(root, "internal", "jobs", "jobs.go"),
		"// gor:workers",
		strings.TrimSpace(string(registerLine)),
	)
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  internal/jobs/jobs.go")
	}

	scheduleSnippet := ""
	if every != "" {
		scheduleSnippet = "jobs/snippets/periodic_register.tmpl"
	} else if cronExpr != "" {
		scheduleSnippet = "jobs/snippets/cron_register.tmpl"
	}
	if scheduleSnippet != "" {
		periodicLine, err := render.File(scheduleSnippet, data)
		if err != nil {
			return err
		}
		if _, err := inject.InsertBefore(
			filepath.Join(root, "internal", "jobs", "jobs.go"),
			"// gor:periodic",
			strings.TrimSpace(string(periodicLine)),
		); err != nil {
			return fmt.Errorf("%w\nIf jobs were set up before periodic support, regenerate with `gor g jobs --force`", err)
		}
	}

	if err := tidy(root); err != nil {
		return err
	}

	if every != "" {
		fmt.Printf(`
Job %[1]s generated — runs every %[2]s (and once at worker startup).
It can also be enqueued on demand: jobs.Enqueue(ctx, jobs.%[3]sArgs{})
`, name, every, data.Pascal)
	} else if cronExpr != "" {
		fmt.Printf(`
Job %[1]s generated — runs on cron schedule %[2]q (worker's local time;
prefix with CRON_TZ=<zone> in internal/jobs/jobs.go to pin a timezone).
It can also be enqueued on demand: jobs.Enqueue(ctx, jobs.%[3]sArgs{})
`, name, cronExpr, data.Pascal)
	} else {
		fmt.Printf(`
Job %[1]s generated. Enqueue it with:

  jobs.Enqueue(ctx, jobs.%[2]sArgs{})
`, name, data.Pascal)
	}
	return nil
}
