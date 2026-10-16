// Package generator implements `gor g resource`.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/deps"
	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
	"github.com/karurosux/go-on-rails/internal/run"
)

// Options configures a resource generation run.
type Options struct {
	Name      string // singular snake_case resource name
	Fields    []fields.Field
	Service   bool
	APIOnly   bool
	Protected bool // register behind auth.Require (needs `gor g auth` first)
	Live      bool // realtime updates (needs `gor g realtime` first)
	Force     bool
}

type plannedFile struct {
	template string
	path     string // relative to app root
}

// Generate creates all backend (and optionally frontend) files for a
// resource, wires up routes, and regenerates sqlc code.
func Generate(root, module string, opts Options) error {
	r := BuildResource(module, opts.Name, opts.Fields, opts.Service)
	if opts.Protected {
		if !HasAuth(root) {
			return fmt.Errorf("--protected needs authentication set up first: run `gor g auth`")
		}
		r.Protected = true
		r.RouteGroup = "protected"
	}
	if opts.Live {
		if !HasRealtime(root) {
			return fmt.Errorf("--live needs realtime set up first: run `gor g realtime`")
		}
		r.Live = true
	}
	if r.HasFile && !opts.APIOnly && !HasUploads(root) {
		return fmt.Errorf("`file` fields need uploads set up first: run `gor g uploads`")
	}

	migration := migrationPath(root, r.Table, opts.Force)
	files := planFiles(r, opts, migration)

	// Check every target up front so a conflict can't leave a half-generated
	// resource behind.
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

	manifest, err := loadManifest(root)
	if err != nil {
		return err
	}
	entry := &ManifestResource{
		Name:      opts.Name,
		Service:   opts.Service,
		APIOnly:   opts.APIOnly,
		Protected: opts.Protected,
		Live:      opts.Live,
		Files:     map[string]string{},
	}
	for _, f := range opts.Fields {
		entry.Fields = append(entry.Fields, f.Name+":"+string(f.Type))
	}

	for _, f := range files {
		content, err := render.File(f.template, r)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(root, f.path), content, opts.Force); err != nil {
			return err
		}
		if f.template != "resource/migration.sql.tmpl" {
			entry.Files[f.path] = hashContent(content)
		}
		fmt.Printf("  create  %s\n", f.path)
	}
	manifest.upsert(entry)
	if err := manifest.save(root); err != nil {
		return err
	}

	if err := wireBackendRoutes(root, r); err != nil {
		return err
	}
	if !opts.APIOnly {
		if err := wireFrontendRoutes(root, r); err != nil {
			return err
		}
	}

	if err := runCodegen(root); err != nil {
		return err
	}

	printSummary(r, opts)
	return nil
}

// planFiles lists every file a resource generates. An empty migration path
// means "skip the migration" (it already exists).
func planFiles(r Resource, opts Options, migration string) []plannedFile {
	files := []plannedFile{
		{"resource/queries.sql.tmpl", filepath.Join("queries", r.Plural+".sql")},
		{"resource/handler.go.tmpl", filepath.Join("internal", r.Name, "handler.go")},
		{"resource/handler_test.go.tmpl", filepath.Join("internal", r.Name, "handler_test.go")},
	}
	if migration != "" {
		files = append([]plannedFile{{"resource/migration.sql.tmpl", migration}}, files...)
	}
	if opts.Service {
		files = append(files, plannedFile{"resource/service.go.tmpl", filepath.Join("internal", r.Name, "service.go")})
	}
	if !opts.APIOnly {
		feat := filepath.Join("web", "src", "features", r.Plural)
		files = append(files,
			plannedFile{"resource/web/api.ts.tmpl", filepath.Join("web", "src", "api", r.Plural+".ts")},
			plannedFile{"resource/web/schema.ts.tmpl", filepath.Join(feat, "schema.ts")},
			plannedFile{"resource/web/List.tsx.tmpl", filepath.Join(feat, r.PluralPascal+"List.tsx")},
			plannedFile{"resource/web/Detail.tsx.tmpl", filepath.Join(feat, r.Pascal+"Detail.tsx")},
			plannedFile{"resource/web/Form.tsx.tmpl", filepath.Join(feat, r.Pascal+"Form.tsx")},
		)
	}
	return files
}

// tidy runs go mod tidy quietly.
func tidy(root string) error {
	if err := run.CommandQuiet(root, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}

// runCodegen re-runs sqlc generate and go mod tidy (skipped with a warning
// when sqlc is not installed).
func runCodegen(root string) error {
	if err := deps.Check("sqlc"); err != nil {
		fmt.Printf("\nwarning: skipped sqlc generate — %v\n", err)
		fmt.Println("Run `sqlc generate` after installing it.")
		return nil
	}
	fmt.Println("  run     sqlc generate")
	if err := run.CommandQuiet(root, "sqlc", "generate"); err != nil {
		return fmt.Errorf("sqlc generate failed: %w", err)
	}
	fmt.Println("  run     go mod tidy")
	if err := run.CommandQuiet(root, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}

// migrationPath returns the new migration's path, or "" if a create-table
// migration for this table already exists and force is set (re-running with
// --force regenerates code but must not duplicate schema history).
func migrationPath(root, table string, force bool) string {
	suffix := "_create_" + table + ".sql"
	entries, _ := os.ReadDir(filepath.Join(root, "migrations"))
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	for _, name := range names {
		if strings.HasSuffix(name, suffix) {
			if force {
				fmt.Printf("  skip    migrations/%s (already exists, keeping schema history)\n", name)
				return ""
			}
			// Without --force the conflict check below reports it.
			return filepath.Join("migrations", name)
		}
	}
	return uniqueMigrationPath(root, "create_"+table)
}

func wireBackendRoutes(root string, r Resource) error {
	routesGo := filepath.Join(root, "internal", "routes", "routes.go")

	importLine, err := render.File("resource/snippets/routes_import.tmpl", r)
	if err != nil {
		return err
	}
	if _, err := inject.InsertBefore(routesGo, "// gor:imports", strings.TrimSpace(string(importLine))); err != nil {
		return err
	}
	registerLine, err := render.File("resource/snippets/routes_register.tmpl", r)
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
	return nil
}

func wireFrontendRoutes(root string, r Resource) error {
	routesTsx := filepath.Join(root, "web", "src", "routes.tsx")
	appTsx := filepath.Join(root, "web", "src", "App.tsx")

	imports, err := render.File("resource/snippets/web_imports.tmpl", r)
	if err != nil {
		return err
	}
	if _, err := inject.InsertBefore(routesTsx, "// gor:imports", strings.TrimRight(string(imports), "\n")); err != nil {
		return err
	}
	routeBlock, err := render.File("resource/snippets/web_routes.tmpl", r)
	if err != nil {
		return err
	}
	routeMarker := "{/* gor:routes */}"
	if r.Protected {
		routeMarker = "{/* gor:protected-routes */}"
	}
	changed, err := inject.InsertBefore(routesTsx, routeMarker, strings.TrimRight(string(routeBlock), "\n"))
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  web/src/routes.tsx")
	}

	nav, err := render.File("resource/snippets/web_nav.tmpl", r)
	if err != nil {
		return err
	}
	changed, err = inject.InsertBefore(appTsx, "{/* gor:nav */}", strings.TrimRight(string(nav), "\n"))
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  web/src/App.tsx")
	}
	return nil
}

func printSummary(r Resource, opts Options) {
	fmt.Printf(`
Resource %s generated.

  API endpoints:
    GET    /api/%s
    POST   /api/%s
    GET    /api/%s/:id
    PATCH  /api/%s/:id
    DELETE /api/%s/:id
`, r.Name, r.Plural, r.Plural, r.Plural, r.Plural, r.Plural)
	if !opts.APIOnly {
		fmt.Printf("\n  Pages: /%s, /%s/new, /%s/:id, /%s/:id/edit\n", r.Plural, r.Plural, r.Plural, r.Plural)
	}
	fmt.Println("\nApply the migration with: gor db migrate")
}
