package generator

import (
	"bytes"
	"go/format"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
	"github.com/karurosux/go-on-rails/internal/run"
)

// Plugins are installable generators: a directory holding plugin.json and
// templates. They use the same primitives as built-in generators — render
// files, inject at gor: markers, append env defaults, run allowlisted
// commands — declaratively, so installing one never executes plugin code.
//
// plugin.json:
//
//	{
//	  "name": "stripe",
//	  "description": "Stripe billing scaffolding",
//	  "files": [
//	    {"template": "templates/billing.go.tmpl", "path": "internal/billing/billing.go"},
//	    {"template": "templates/page.tsx.tmpl", "path": "web/src/features/billing/Page.tsx", "delims": "square"}
//	  ],
//	  "injections": [
//	    {"file": "internal/routes/routes.go", "marker": "// gor:imports", "snippet": "templates/import.snippet"},
//	    {"file": "internal/routes/routes.go", "marker": "// gor:routes", "snippet": "templates/routes.snippet"}
//	  ],
//	  "env": {"sentinel": "STRIPE_SECRET_KEY", "file": "templates/env.tmpl"},
//	  "postRun": ["tidy"],
//	  "message": "Next: set STRIPE_SECRET_KEY and run gor db migrate"
//	}
//
// Templates render with {{.Module}}, {{.AppName}}, and {{.Args}} (extra CLI
// args). "delims": "square" switches to [[ ]] for files that contain
// runtime {{ }} syntax. postRun entries must be in the allowlist:
// "tidy" (go mod tidy) or "sqlc" (sqlc generate + tidy).
//
// A file path may contain a literal {{timestamp}} token in its base name
// (e.g. "migrations/{{timestamp}}_create_billing.sql"). It resolves to a
// goose-style version (UTC now, bumped until unique in the directory) when
// the plugin runs, so shipped migrations always sort after the app's
// existing ones. If a file matching the rest of the name already exists,
// that file is reused — re-running the plugin overwrites it (with --force)
// instead of duplicating schema history.

// Plugin is one installed generator plugin.
type Plugin struct {
	Dir      string
	Manifest PluginManifest
}

// PluginManifest is the parsed plugin.json.
type PluginManifest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Files       []PluginFile      `json:"files"`
	Injections  []PluginInjection `json:"injections"`
	Env         *PluginEnv        `json:"env"`
	PostRun     []string          `json:"postRun"`
	Message     string            `json:"message"`
}

// PluginFile is one file the plugin renders into the app.
type PluginFile struct {
	Template string `json:"template"`
	Path     string `json:"path"`
	Delims   string `json:"delims"` // "" (default {{ }}) or "square" ([[ ]])
}

// PluginInjection inserts a rendered snippet before a marker in a file.
type PluginInjection struct {
	File    string `json:"file"`
	Marker  string `json:"marker"`
	Snippet string `json:"snippet"`
}

// PluginEnv appends env defaults, skipped when sentinel is present.
type PluginEnv struct {
	Sentinel string `json:"sentinel"`
	File     string `json:"file"`
}

var pluginNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// builtinGenerators are subcommand names plugins may not shadow.
var builtinGenerators = map[string]bool{
	"resource": true, "model": true, "migration": true, "sync": true,
	"auth": true, "auth-emails": true, "jobs": true, "job": true,
	"realtime": true, "mailer": true, "uploads": true,
}

// UserPluginsDir is the user-wide plugin directory.
func UserPluginsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gor", "plugins"), nil
}

// LoadPlugins returns installed plugins; projectRoot may be "" outside an
// app. Project-level plugins (.gor/plugins) shadow user-level ones.
func LoadPlugins(projectRoot string) ([]Plugin, error) {
	seen := map[string]bool{}
	var plugins []Plugin

	var dirs []string
	if projectRoot != "" {
		dirs = append(dirs, filepath.Join(projectRoot, ".gor", "plugins"))
	}
	if userDir, err := UserPluginsDir(); err == nil {
		dirs = append(dirs, userDir)
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			// A linked (dev) install is a symlink to the plugin's working
			// directory, so resolve through it.
			info, err := os.Stat(filepath.Join(dir, e.Name()))
			if err != nil || !info.IsDir() {
				continue
			}
			p, err := loadPlugin(filepath.Join(dir, e.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping plugin %s: %v\n", e.Name(), err)
				continue
			}
			if seen[p.Manifest.Name] {
				continue
			}
			seen[p.Manifest.Name] = true
			plugins = append(plugins, p)
		}
	}
	return plugins, nil
}

func loadPlugin(dir string) (Plugin, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return Plugin{}, err
	}
	var m PluginManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Plugin{}, fmt.Errorf("parse plugin.json: %w", err)
	}
	if !pluginNameRe.MatchString(m.Name) {
		return Plugin{}, fmt.Errorf("invalid plugin name %q", m.Name)
	}
	return Plugin{Dir: dir, Manifest: m}, nil
}

// InstallPlugin copies a plugin from a local path or clones it from a git
// URL into the user (or project) plugin directory. With link, the plugin
// directory is symlinked instead of copied — a dev install where edits are
// live, no reinstall needed.
func InstallPlugin(src string, project, link bool) (Plugin, error) {
	srcDir := src
	isGit := strings.Contains(src, "://") || strings.HasPrefix(src, "git@") || strings.HasSuffix(src, ".git")
	if isGit && link {
		return Plugin{}, fmt.Errorf("--link needs a local path (clone the repo first, then link the clone)")
	}
	if isGit {
		tmp, err := os.MkdirTemp("", "gor-plugin-*")
		if err != nil {
			return Plugin{}, err
		}
		defer os.RemoveAll(tmp)
		if err := run.CommandQuiet("", "git", "clone", "--depth", "1", src, tmp); err != nil {
			return Plugin{}, err
		}
		srcDir = tmp
	}

	p, err := loadPlugin(srcDir)
	if err != nil {
		return Plugin{}, fmt.Errorf("%s is not a gor plugin: %w", src, err)
	}
	if builtinGenerators[p.Manifest.Name] {
		return Plugin{}, fmt.Errorf("plugin name %q shadows a built-in generator", p.Manifest.Name)
	}

	var base string
	if project {
		base = filepath.Join(".gor", "plugins")
	} else {
		base, err = UserPluginsDir()
		if err != nil {
			return Plugin{}, err
		}
	}
	dest := filepath.Join(base, p.Manifest.Name)
	if err := os.RemoveAll(dest); err != nil {
		return Plugin{}, err
	}
	if link {
		abs, err := filepath.Abs(srcDir)
		if err != nil {
			return Plugin{}, err
		}
		if err := os.MkdirAll(base, 0o755); err != nil {
			return Plugin{}, err
		}
		if err := os.Symlink(abs, dest); err != nil {
			return Plugin{}, err
		}
		p.Dir = dest
		return p, nil
	}
	if err := copyDir(srcDir, dest); err != nil {
		return Plugin{}, err
	}
	p.Dir = dest
	return p, nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
}

// PluginData is the template data every plugin template receives.
type PluginData struct {
	Module  string
	AppName string
	Args    []string
}

// RunPlugin executes an installed plugin against the app at root.
func RunPlugin(p Plugin, root, module, appName string, args []string, force bool) error {
	data := PluginData{Module: module, AppName: appName, Args: args}

	paths := make([]string, len(p.Manifest.Files))
	for i, f := range p.Manifest.Files {
		paths[i] = resolvePluginPath(root, f.Path)
	}

	if !force {
		var conflicts []string
		for _, path := range paths {
			if render.Exists(filepath.Join(root, path)) {
				conflicts = append(conflicts, path)
			}
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("these files already exist (use --force to overwrite):\n  %s",
				strings.Join(conflicts, "\n  "))
		}
	}

	for i, f := range p.Manifest.Files {
		content, err := renderPluginTemplate(p.Dir, f.Template, f.Delims, data)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(root, paths[i]), content, force); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", paths[i])
	}

	for _, in := range p.Manifest.Injections {
		snippet, err := renderPluginTemplate(p.Dir, in.Snippet, "", data)
		if err != nil {
			return err
		}
		changed, err := inject.InsertBefore(filepath.Join(root, in.File), in.Marker,
			strings.TrimRight(string(snippet), "\n"))
		if err != nil {
			return err
		}
		if changed {
			fmt.Printf("  inject  %s\n", in.File)
		}
	}

	if p.Manifest.Env != nil {
		block, err := renderPluginTemplate(p.Dir, p.Manifest.Env.File, "", data)
		if err != nil {
			return err
		}
		if err := ensureEnvVars(root, p.Manifest.Env.Sentinel, string(block)); err != nil {
			return err
		}
	}

	for _, cmd := range p.Manifest.PostRun {
		switch cmd {
		case "tidy":
			fmt.Println("  run     go mod tidy")
			if err := tidy(root); err != nil {
				return err
			}
		case "sqlc":
			if err := runCodegen(root); err != nil {
				return err
			}
		default:
			return fmt.Errorf("plugin postRun %q is not allowed (allowlist: tidy, sqlc)", cmd)
		}
	}

	if p.Manifest.Message != "" {
		fmt.Println("\n" + p.Manifest.Message)
	}
	return nil
}

// resolvePluginPath expands a literal {{timestamp}} token in a plugin file
// path's base name into a goose-style version: UTC now, bumped by a second
// until no file in the target directory shares the version. When a file
// already matches the name around the token, its path is returned instead,
// so re-running a plugin targets the same migration rather than minting a
// new one. Paths without the token pass through untouched.
func resolvePluginPath(root, path string) string {
	const token = "{{timestamp}}"
	dir, base := filepath.Split(filepath.FromSlash(path))
	i := strings.Index(base, token)
	if i == -1 {
		return path
	}
	prefix, suffix := base[:i], base[i+len(token):]

	entries, _ := os.ReadDir(filepath.Join(root, dir))
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	for _, name := range names {
		if len(name) > len(prefix)+len(suffix) &&
			strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			return filepath.Join(dir, name)
		}
	}
	ts := time.Now().UTC()
	for {
		version := ts.Format("20060102150405")
		taken := false
		for _, name := range names {
			if strings.HasPrefix(name, prefix+version) {
				taken = true
				break
			}
		}
		if !taken {
			return filepath.Join(dir, prefix+version+suffix)
		}
		ts = ts.Add(time.Second)
	}
}

func renderPluginTemplate(dir, name, delims string, data PluginData) ([]byte, error) {
	path := filepath.Join(dir, name)
	if rel, err := filepath.Rel(dir, path); err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("plugin template %q escapes the plugin directory", name)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plugin template %s: %w", name, err)
	}
	t := template.New(filepath.Base(name))
	if delims == "square" {
		t = t.Delims("[[", "]]")
	}
	t, err = t.Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse plugin template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render plugin template %s: %w", name, err)
	}
	out := buf.Bytes()
	if strings.HasSuffix(name, ".go.tmpl") {
		formatted, err := format.Source(out)
		if err != nil {
			return nil, fmt.Errorf("plugin template %s produced invalid Go: %w", name, err)
		}
		out = formatted
	}
	return out, nil
}
