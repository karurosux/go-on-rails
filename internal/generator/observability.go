package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

var promSubsystemRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Observability sets up vendor-neutral monitoring: 5xx error reports to any
// Sentry-compatible service (SENTRY_DSN) and Prometheus metrics at /metrics.
func Observability(root, module, appName string, force bool) error {
	data := struct {
		Module           string
		MetricsSubsystem string
	}{
		Module: module,
		// Prometheus identifiers can't contain dashes etc.
		MetricsSubsystem: promSubsystemRe.ReplaceAllString(appName, "_"),
	}

	path := filepath.Join("internal", "obs", "obs.go")
	if !force && render.Exists(filepath.Join(root, path)) {
		return fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}
	content, err := render.File("observability/obs.go.tmpl", data)
	if err != nil {
		return err
	}
	if err := render.Write(filepath.Join(root, path), content, force); err != nil {
		return err
	}
	fmt.Printf("  create  %s\n", path)

	routesGo := filepath.Join(root, "internal", "routes", "routes.go")
	imports, err := render.File("observability/snippets/routes_import.tmpl", data)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(imports)), "\n") {
		if _, err := inject.InsertBefore(routesGo, "// gor:imports", line); err != nil {
			return err
		}
	}
	block, err := render.File("observability/snippets/routes_block.tmpl", data)
	if err != nil {
		return err
	}
	// Before healthz so every subsequently registered route is measured;
	// fall back to the routes marker if healthz was removed.
	changed, err := inject.InsertBefore(routesGo, `api.GET("/healthz"`, strings.TrimRight(string(block), "\n"))
	if err != nil {
		changed, err = inject.InsertBefore(routesGo, "// gor:routes", strings.TrimRight(string(block), "\n"))
	}
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  internal/routes/routes.go")
	}

	if err := ensureEnvVars(root, "SENTRY_DSN", `
# Observability (gor g observability)
# Error reports go to any Sentry-compatible service (Sentry, GlitchTip, Bugsink).
# Unset = reporting disabled. Metrics are always served at /metrics.
# SENTRY_DSN=
`); err != nil {
		return err
	}
	if err := tidy(root); err != nil {
		return err
	}

	fmt.Print(`
Observability set up — emission is vendor-neutral, storage is external:

  Errors:   5xx responses report to SENTRY_DSN (Sentry/GlitchTip/Bugsink);
            unset = disabled. From jobs/goroutines: obs.CaptureError(err)
  Metrics:  GET /metrics (Prometheus format) — scrape it with Prometheus,
            Grafana Cloud, or any vendor agent. Keep it firewalled or
            protect it; it is not meant to be public.
  Logs:     already structured JSON on stdout — point your collector there.
`)
	return nil
}
