package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// HasRealtime reports whether the app has the realtime package (used to
// validate --live).
func HasRealtime(root string) bool {
	return render.Exists(filepath.Join(root, "internal", "realtime", "realtime.go"))
}

// Realtime sets up server-push over SSE + Postgres NOTIFY: the realtime
// package (Publish + SSE handler) and the React hooks to consume it.
func Realtime(root, module string, apiOnly, force bool) error {
	protected := HasAuth(root)
	data := struct {
		Module    string
		Protected bool
	}{Module: module, Protected: protected}

	files := []plannedFile{
		{"realtime/realtime.go.tmpl", filepath.Join("internal", "realtime", "realtime.go")},
	}
	if !apiOnly {
		files = append(files,
			plannedFile{"realtime/web/realtime.ts.tmpl", filepath.Join("web", "src", "lib", "realtime.ts")},
		)
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

	routesGo := filepath.Join(root, "internal", "routes", "routes.go")
	importLine, err := render.File("realtime/snippets/routes_import.tmpl", data)
	if err != nil {
		return err
	}
	if _, err := inject.InsertBefore(routesGo, "// gor:imports", strings.TrimSpace(string(importLine))); err != nil {
		return err
	}
	registerLine, err := render.File("realtime/snippets/routes_register.tmpl", data)
	if err != nil {
		return err
	}
	changed, err := inject.InsertBefore(routesGo, "// gor:routes", strings.TrimSpace(string(registerLine)))
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  internal/routes/routes.go")
	}

	if err := tidy(root); err != nil {
		return err
	}

	visibility := "public — generate auth first (or wrap it yourself) to protect it"
	if protected {
		visibility = "registered behind auth.Require (sign-in needed to subscribe)"
	}
	fmt.Printf(`
Realtime set up: SSE + Postgres NOTIFY, no extra services.

  Endpoint:   GET /api/realtime/:channel (%s)
  Publish:    realtime.Publish(ctx, "posts", map[string]any{"event": "created"})
              (works from handlers and the jobs worker)
  Subscribe:  useRealtime("posts", (data) => ...)
              useRealtimeInvalidate("posts", ["posts"])  // live TanStack queries
`, visibility)
	return nil
}
