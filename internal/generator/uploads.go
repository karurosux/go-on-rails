package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// HasUploads reports whether the app has the storage package (used to
// validate `file` resource fields).
func HasUploads(root string) bool {
	return render.Exists(filepath.Join(root, "internal", "storage", "storage.go"))
}

// Uploads sets up file uploads: a disk-backed storage package with an
// upload endpoint, static serving at /uploads/*, and the frontend upload
// control that `file` resource fields plug into.
func Uploads(root, module string, apiOnly, force bool) error {
	protected := HasAuth(root)
	data := struct {
		Module    string
		Protected bool
	}{Module: module, Protected: protected}

	files := []plannedFile{
		{"uploads/storage.go.tmpl", filepath.Join("internal", "storage", "storage.go")},
	}
	if !apiOnly {
		files = append(files,
			plannedFile{"uploads/web/uploads.ts.tmpl", filepath.Join("web", "src", "api", "uploads.ts")},
			plannedFile{"uploads/web/upload-input.tsx.tmpl", filepath.Join("web", "src", "components", "upload-input.tsx")},
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
	for _, importLine := range []string{
		`"github.com/labstack/echo/v4/middleware"`,
		fmt.Sprintf("%q", data.Module+"/internal/storage"),
	} {
		if _, err := inject.InsertBefore(routesGo, "// gor:imports", importLine); err != nil {
			return err
		}
	}
	registerBlock, err := render.File("uploads/snippets/routes_register.tmpl", data)
	if err != nil {
		return err
	}
	changed, err := inject.InsertBefore(routesGo, "// gor:routes", strings.TrimRight(string(registerBlock), "\n"))
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  internal/routes/routes.go")
	}

	if err := ensureEnvVars(root, "STORAGE_DIR", `
# File uploads (gor g uploads)
# STORAGE_DIR=./storage/uploads
`); err != nil {
		return err
	}
	if err := ignoreStorageDir(root); err != nil {
		return err
	}
	if err := tidy(root); err != nil {
		return err
	}

	visibility := "public — generate auth first (or wrap it yourself) to protect it"
	if protected {
		visibility = "behind auth.Require (sign-in needed to upload)"
	}
	fmt.Printf(`
File uploads set up (local disk; STORAGE_DIR overrides the location).

  Endpoint:   POST /api/uploads, 10MB limit (%s)
  Serving:    GET /uploads/<key>
  Resources:  gor g resource doc title:string attachment:file
              — forms get an upload control, the column stores the URL

In production, mount a persistent volume at the storage directory.
`, visibility)
	return nil
}

// ignoreStorageDir keeps uploaded files out of git.
func ignoreStorageDir(root string) error {
	path := filepath.Join(root, ".gitignore")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil // no .gitignore — leave it be
	}
	if strings.Contains(string(raw), "/storage/") {
		return nil
	}
	entry := []byte("\n# uploaded files\n/storage/\n")
	if err := os.WriteFile(path, append(raw, entry...), 0o644); err != nil {
		return err
	}
	fmt.Println("  append  .gitignore (/storage/)")
	return nil
}
