// Package render renders embedded templates and writes generated files,
// refusing to overwrite existing files unless forced.
package render

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/karurosux/go-on-rails/templates"
)

// File renders the embedded template at name (e.g. "resource/handler.go.tmpl")
// with data. Generated Go code is run through gofmt; a formatting failure
// means the template produced invalid Go and is reported as a bug.
func File(name string, data any) ([]byte, error) {
	raw, err := templates.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}
	t, err := template.New(filepath.Base(name)).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", name, err)
	}
	out := buf.Bytes()
	if strings.HasSuffix(strings.TrimSuffix(name, ".tmpl"), ".go") {
		formatted, err := format.Source(out)
		if err != nil {
			return nil, fmt.Errorf("template %s produced invalid Go (this is a gor bug): %w", name, err)
		}
		out = formatted
	}
	return out, nil
}

// FileDelims renders an embedded template using [[ ]] as the action
// delimiters, for generating files that themselves contain {{ }} runtime
// template syntax (e.g. email HTML templates).
func FileDelims(name string, data any) ([]byte, error) {
	raw, err := templates.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}
	t, err := template.New(filepath.Base(name)).Delims("[[", "]]").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// Write writes content to path, creating parent directories. It refuses to
// overwrite an existing file unless force is true.
func Write(path string, content []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
