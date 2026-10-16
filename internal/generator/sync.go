package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/render"
)

// Sync regenerates every derived file of a resource from its manifest entry.
// Files still matching the hash gor last wrote (i.e. untouched by the user)
// are rewritten; files the user has edited are skipped with a warning so
// manual changes are never clobbered.
func Sync(root, module, name string) error {
	manifest, err := loadManifest(root)
	if err != nil {
		return err
	}
	entry := manifest.find(name)
	if entry == nil {
		return fmt.Errorf("no manifest entry for resource %q (generated before gor tracked resources?)\nRe-run `gor g resource %s <fields...> --force` once to start tracking it", name, name)
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
	opts := Options{Name: name, Service: entry.Service, APIOnly: entry.APIOnly, Protected: entry.Protected}

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

	if err := wireBackendRoutes(root, r); err != nil {
		return err
	}
	if !entry.APIOnly {
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
	return Sync(root, module, entry.Name)
}
