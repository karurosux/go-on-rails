package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// AuthOptions configures `gor g auth`.
type AuthOptions struct {
	APIOnly bool
	Force   bool
}

// Auth generates cookie-session authentication: users + sessions tables,
// the auth package (middleware, role guards, endpoints), login/register
// pages, and wires everything into the app's routes.
func Auth(root, module string, opts AuthOptions) error {
	data := struct{ Module string }{Module: module}

	migration := migrationPath(root, "users", opts.Force)
	files := []plannedFile{
		{"auth/users.sql.tmpl", filepath.Join("queries", "users.sql")},
		{"auth/sessions.sql.tmpl", filepath.Join("queries", "sessions.sql")},
		{"auth/auth.go.tmpl", filepath.Join("internal", "auth", "auth.go")},
		{"auth/middleware.go.tmpl", filepath.Join("internal", "auth", "middleware.go")},
		{"auth/handler.go.tmpl", filepath.Join("internal", "auth", "handler.go")},
		{"auth/password.go.tmpl", filepath.Join("internal", "auth", "password.go")},
		{"auth/handler_test.go.tmpl", filepath.Join("internal", "auth", "handler_test.go")},
	}
	if migration != "" {
		files = append([]plannedFile{{"auth/migration.sql.tmpl", migration}}, files...)
	}
	if !opts.APIOnly {
		feat := filepath.Join("web", "src", "features", "auth")
		files = append(files,
			plannedFile{"auth/web/api.ts.tmpl", filepath.Join("web", "src", "api", "auth.ts")},
			plannedFile{"auth/web/schema.ts.tmpl", filepath.Join(feat, "schema.ts")},
			plannedFile{"auth/web/LoginPage.tsx.tmpl", filepath.Join(feat, "LoginPage.tsx")},
			plannedFile{"auth/web/RegisterPage.tsx.tmpl", filepath.Join(feat, "RegisterPage.tsx")},
			plannedFile{"auth/web/RequireAuth.tsx.tmpl", filepath.Join(feat, "RequireAuth.tsx")},
			plannedFile{"auth/web/gates.tsx.tmpl", filepath.Join(feat, "gates.tsx")},
			plannedFile{"auth/web/AuthNav.tsx.tmpl", filepath.Join(feat, "AuthNav.tsx")},
		)
	}

	if !opts.Force {
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
		if err := render.Write(filepath.Join(root, f.path), content, opts.Force); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", f.path)
	}

	if err := wireAuthBackend(root, data); err != nil {
		return err
	}
	if err := protectGeneratedRoutes(root); err != nil {
		return err
	}
	if !opts.APIOnly {
		if err := wireAuthFrontend(root); err != nil {
			return err
		}
	}

	if err := runCodegen(root); err != nil {
		return err
	}

	printAuthSummary(opts)
	return nil
}

func wireAuthBackend(root string, data any) error {
	routesGo := filepath.Join(root, "internal", "routes", "routes.go")

	importLine, err := render.File("auth/snippets/routes_import.tmpl", data)
	if err != nil {
		return err
	}
	if _, err := inject.InsertBefore(routesGo, "// gor:imports", strings.TrimSpace(string(importLine))); err != nil {
		return err
	}

	block, err := render.File("auth/snippets/routes_block.tmpl", data)
	if err != nil {
		return err
	}
	// Insert right before the healthz route so the middleware and the
	// protected group sit above every resource registration; fall back to
	// the gor:routes marker if healthz was removed.
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
	return nil
}

// protectGeneratedRoutes moves endpoints generated before auth (realtime,
// uploads) behind auth.Require, so generation order doesn't matter.
// Customized lines are left alone.
func protectGeneratedRoutes(root string) error {
	path := filepath.Join(root, "internal", "routes", "routes.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)
	swaps := map[string]string{
		`api.GET("/realtime/:channel", realtime.Handler())`:               `protected.GET("/realtime/:channel", realtime.Handler())`,
		`api.POST("/uploads", storage.Handler(), middleware.BodyLimit("10M"))`: `protected.POST("/uploads", storage.Handler(), middleware.BodyLimit("10M"))`,
	}
	changed := false
	for public, protected := range swaps {
		if strings.Contains(content, public) {
			content = strings.Replace(content, public, protected, 1)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("protecting generated routes produced invalid Go (this is a gor bug): %w", err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return err
	}
	fmt.Println("  update  internal/routes/routes.go (generated endpoints now require auth)")
	return nil
}

func wireAuthFrontend(root string) error {
	routesTsx := filepath.Join(root, "web", "src", "routes.tsx")
	appTsx := filepath.Join(root, "web", "src", "App.tsx")

	imports, err := render.File("auth/snippets/web_imports.tmpl", nil)
	if err != nil {
		return err
	}
	if _, err := inject.InsertBefore(routesTsx, "// gor:imports", strings.TrimRight(string(imports), "\n")); err != nil {
		return err
	}
	// The route block changed shape over time (login/register are now
	// wrapped in <RequireGuest />); if auth routes already exist from an
	// earlier generation, leave them alone instead of injecting duplicates.
	if raw, err := os.ReadFile(routesTsx); err == nil && strings.Contains(string(raw), `path="login"`) &&
		!strings.Contains(string(raw), "<RequireGuest />") {
		fmt.Println("  note    web/src/routes.tsx already has auth routes — wrap /login and /register in <Route element={<RequireGuest />}> yourself for the guest-only guard")
	} else {
		routeBlock, err := render.File("auth/snippets/web_routes.tmpl", nil)
		if err != nil {
			return err
		}
		changed, err := inject.InsertBefore(routesTsx, "{/* gor:routes */}", strings.TrimRight(string(routeBlock), "\n"))
		if err != nil {
			return err
		}
		if changed {
			fmt.Println("  inject  web/src/routes.tsx")
		}
	}

	// App.tsx: nav import + <AuthNav /> at the end of the nav. These are
	// cosmetic, so a customized App.tsx downgrades to a warning.
	if _, err := inject.InsertBefore(appTsx, `from "react-router"`,
		`import { AuthNav } from "./features/auth/AuthNav";`); err != nil {
		fmt.Printf("  warning: could not add AuthNav import to App.tsx (%v) — add it yourself\n", err)
		return nil
	}
	navChanged, err := inject.InsertBefore(appTsx, "</nav>", "<AuthNav />")
	if err != nil {
		fmt.Printf("  warning: could not add <AuthNav /> to App.tsx (%v) — add it yourself\n", err)
		return nil
	}
	if navChanged {
		fmt.Println("  inject  web/src/App.tsx")
	}
	return nil
}

func printAuthSummary(opts AuthOptions) {
	fmt.Print(`
Authentication generated.

  Endpoints: POST /api/auth/register | login | logout, GET /api/auth/me
`)
	if !opts.APIOnly {
		fmt.Println("  Pages:     /login, /register (nav shows session state)")
	}
	fmt.Print(`
  Protect a new resource:    gor g resource note title:string --protected
  Protect an existing one:   in internal/routes/routes.go, change
                             <name>.Register(api, q) to <name>.Register(protected, q)
`)
	if !opts.APIOnly {
		fmt.Println(`                             and in web/src/routes.tsx, move its <Route> block
                             inside the <RequireAuth /> wrapper`)
	}
	fmt.Print(`  Role-gate routes:          api.Group("/admin", auth.RequireRole("admin"))
  Promote an admin:          UPDATE users SET role = 'admin' WHERE email = '...';

Apply the migration with: gor db migrate
`)
}

// HasAuth reports whether the app's routes are wired for auth (used to
// validate --protected).
func HasAuth(root string) bool {
	raw, err := os.ReadFile(filepath.Join(root, "internal", "routes", "routes.go"))
	return err == nil && strings.Contains(string(raw), "protected :=")
}
