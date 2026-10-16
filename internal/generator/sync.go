package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/render"
)

// FlagChanges toggles a resource's situational flags during Sync; nil
// pointers leave the current setting alone.
type FlagChanges struct {
	Live      *bool
	Protected *bool
}

// Sync regenerates every derived file of a resource from its manifest entry,
// optionally toggling flags first. Files still matching the hash gor last
// wrote (i.e. untouched by the user) are rewritten; files the user has
// edited are skipped with a warning so manual changes are never clobbered.
func Sync(root, module, name string, changes *FlagChanges) error {
	manifest, err := loadManifest(root)
	if err != nil {
		return err
	}
	entry := manifest.find(name)
	if entry == nil {
		return fmt.Errorf("no manifest entry for resource %q (generated before gor tracked resources?)\nRe-run `gor g resource %s <fields...> --force` once to start tracking it", name, name)
	}

	if changes != nil {
		if changes.Live != nil {
			if *changes.Live && !HasRealtime(root) {
				return fmt.Errorf("--live needs realtime set up first: run `gor g realtime`")
			}
			entry.Live = *changes.Live
		}
		if changes.Protected != nil {
			if *changes.Protected && !HasAuth(root) {
				return fmt.Errorf("--protected needs authentication set up first: run `gor g auth`")
			}
			entry.Protected = *changes.Protected
		}
	}

	parsed, err := fields.Parse(entry.Fields)
	if err != nil {
		return fmt.Errorf("manifest fields for %s are invalid: %w", name, err)
	}
	r := BuildResource(module, name, parsed, entry.Service)
	if entry.Protected {
		r.Protected = true
		r.RouteGroup = "protected"
	}
	r.Live = entry.Live
	opts := Options{Name: name, Service: entry.Service, APIOnly: entry.APIOnly, Protected: entry.Protected, Live: entry.Live}

	var skipped []string
	for _, f := range planFiles(r, opts, "") {
		content, err := render.File(f.template, r)
		if err != nil {
			return err
		}
		abs := filepath.Join(root, f.path)
		current, readErr := os.ReadFile(abs)
		switch {
		case readErr == nil && hashContent(current) == hashContent(content):
			// already up to date
		case readErr == nil && hashContent(current) != entry.Files[f.path]:
			skipped = append(skipped, f.path)
			continue
		default:
			if err := render.Write(abs, content, true); err != nil {
				return err
			}
			fmt.Printf("  update  %s\n", f.path)
		}
		entry.Files[f.path] = hashContent(content)
	}

	if err := moveBackendRegistration(root, r); err != nil {
		return err
	}
	if err := wireBackendRoutes(root, r); err != nil {
		return err
	}
	if !entry.APIOnly {
		if err := moveFrontendBlock(root, r); err != nil {
			return err
		}
		if err := wireFrontendRoutes(root, r); err != nil {
			return err
		}
	}
	if err := manifest.save(root); err != nil {
		return err
	}
	if err := runCodegen(root); err != nil {
		return err
	}

	if len(skipped) > 0 {
		fmt.Printf("\nSkipped files you have edited — apply the field change there yourself:\n  %s\n",
			strings.Join(skipped, "\n  "))
	}
	return nil
}

// moveBackendRegistration swaps the resource's route registration between
// the api and protected groups when its Protected flag changed.
func moveBackendRegistration(root string, r Resource) error {
	path := filepath.Join(root, "internal", "routes", "routes.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	otherGroup := "protected"
	if r.RouteGroup == "protected" {
		otherGroup = "api"
	}
	old := fmt.Sprintf("%s.Register(%s, q)", r.PkgName, otherGroup)
	content := string(raw)
	if !strings.Contains(content, old) {
		return nil // already on the right group, or not registered yet
	}
	content = strings.Replace(content, old, fmt.Sprintf("%s.Register(%s, q)", r.PkgName, r.RouteGroup), 1)
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("moving %s registration produced invalid Go (this is a gor bug): %w", r.Name, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return err
	}
	fmt.Printf("  update  internal/routes/routes.go (%s now on %s)\n", r.Name, r.RouteGroup)
	return nil
}

// moveFrontendBlock relocates the resource's <Route> block between the
// public and protected regions of routes.tsx when its Protected flag
// changed. A block that looks hand-customized is left alone with a warning.
func moveFrontendBlock(root string, r Resource) error {
	path := filepath.Join(root, "web", "src", "routes.tsx")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == fmt.Sprintf(`<Route path="%s">`, r.Plural) {
			start = i
			break
		}
	}
	protectedMarker := -1
	for i, line := range lines {
		if strings.Contains(line, "{/* gor:protected-routes */}") {
			protectedMarker = i
			break
		}
	}
	if start == -1 || protectedMarker == -1 {
		return nil // block or marker missing; wireFrontendRoutes handles fresh insertion
	}

	// Blocks sit directly above their marker, and the protected marker
	// precedes the public region — so position tells us where it lives now.
	isProtectedNow := start < protectedMarker
	if isProtectedNow == r.Protected {
		return nil
	}

	end := -1
	for i := start + 1; i < len(lines) && i <= start+8; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "</Route>" {
			end = i
			break
		}
		if !strings.HasPrefix(trimmed, "<Route ") {
			break // customized content inside the block — don't touch it
		}
	}
	if end == -1 {
		fmt.Printf("  warning: %s's route block in routes.tsx looks customized — move it between the public and protected regions yourself\n", r.Name)
		return nil
	}

	rest := append(append([]string{}, lines[:start]...), lines[end+1:]...)
	if err := os.WriteFile(path, []byte(strings.Join(rest, "\n")), 0o644); err != nil {
		return err
	}
	// wireFrontendRoutes (called next) re-injects it under the right marker.
	fmt.Println("  update  web/src/routes.tsx (route block moved)")
	return nil
}

// syncAfterMigration updates the manifest for a column add/remove on table
// and regenerates the resource's derived files. A table gor doesn't track
// just gets a hint instead.
func syncAfterMigration(root, module, table string, add bool, fieldArgs []string) error {
	manifest, err := loadManifest(root)
	if err != nil {
		return err
	}
	entry := manifest.findByTable(table)
	if entry == nil {
		fmt.Printf("\nNo gor-generated resource tracks table %q — update handlers/frontend manually if needed.\n", table)
		return nil
	}

	if add {
		entry.Fields = append(entry.Fields, fieldArgs...)
	} else {
		removed := map[string]bool{}
		for _, arg := range fieldArgs {
			n, _, _ := strings.Cut(arg, ":")
			removed[n] = true
		}
		kept := entry.Fields[:0]
		for _, f := range entry.Fields {
			n, _, _ := strings.Cut(f, ":")
			if !removed[n] {
				kept = append(kept, f)
			}
		}
		entry.Fields = kept
	}
	if err := manifest.save(root); err != nil {
		return err
	}

	fmt.Printf("\nSyncing resource %s...\n", entry.Name)
	return Sync(root, module, entry.Name, nil)
}
