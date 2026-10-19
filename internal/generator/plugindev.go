package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/render"
)

// PluginNew scaffolds a new generator plugin in ./gor-<name>: a working
// example (file + route injection + env) ready for `gor plugins install`.
func PluginNew(name string) error {
	if !pluginNameRe.MatchString(name) {
		return fmt.Errorf("plugin name %q must be lowercase letters, digits, or dashes, starting with a letter", name)
	}
	if builtinGenerators[name] {
		return fmt.Errorf("plugin name %q shadows a built-in generator", name)
	}

	dir := "gor-" + name
	if render.Exists(dir) {
		return fmt.Errorf("%s already exists", dir)
	}

	data := struct {
		Name      string
		Pkg       string // valid Go identifier: dashes removed
		EnvPrefix string // SCREAMING_SNAKE for env keys
	}{
		Name:      name,
		Pkg:       strings.ReplaceAll(name, "-", ""),
		EnvPrefix: strings.ToUpper(strings.ReplaceAll(name, "-", "_")),
	}

	files := []struct{ template, path string }{
		{"plugindev/plugin.json.tmpl", "plugin.json"},
		{"plugindev/README.md.tmpl", "README.md"},
		{"plugindev/example.go.tmpl", filepath.Join("templates", data.Pkg+".go.tmpl")},
		{"plugindev/routes_import.snippet.tmpl", filepath.Join("templates", "routes_import.snippet")},
		{"plugindev/routes_register.snippet.tmpl", filepath.Join("templates", "routes_register.snippet")},
		{"plugindev/env.tmpl.tmpl", filepath.Join("templates", "env.tmpl")},
	}
	for _, f := range files {
		content, err := render.FileDelims(f.template, data)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(dir, f.path), content, false); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", filepath.Join(dir, f.path))
	}

	fmt.Printf(`
Plugin scaffold ready. Develop it against a throwaway app:

  gor new testapp && cd testapp
  gor plugins install ../%s --project
  gor g %s

Edit %s/plugin.json and templates/ to make it real — the README inside
documents the manifest schema and available markers.
`, dir, name, dir)
	return nil
}
