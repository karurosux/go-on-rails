package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/karurosux/go-on-rails/internal/fields"
)

// Manifest records what gor generated, so later schema changes can
// regenerate derived files automatically — without clobbering files the
// user has edited (detected via content hashes).
type Manifest struct {
	Version   int                 `json:"version"`
	Resources []*ManifestResource `json:"resources"`
}

// ManifestResource is one generated resource: its field definitions (raw
// name:type strings) and the hash each generated file had when gor wrote it.
type ManifestResource struct {
	Name      string            `json:"name"`
	Service   bool              `json:"service,omitempty"`
	APIOnly   bool              `json:"apiOnly,omitempty"`
	Protected bool              `json:"protected,omitempty"`
	Fields    []string          `json:"fields"`
	Files     map[string]string `json:"files"`
}

const manifestPath = ".gor/manifest.json"

func loadManifest(root string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, manifestPath))
	if os.IsNotExist(err) {
		return &Manifest{Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	return &m, nil
}

func (m *Manifest) save(root string) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(root, manifestPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func (m *Manifest) find(name string) *ManifestResource {
	for _, r := range m.Resources {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// findByTable matches a resource by its table name (plural).
func (m *Manifest) findByTable(table string) *ManifestResource {
	for _, r := range m.Resources {
		if fields.Plural(r.Name) == table {
			return r
		}
	}
	return nil
}

// upsert replaces or appends the entry for res.Name.
func (m *Manifest) upsert(res *ManifestResource) {
	for i, r := range m.Resources {
		if r.Name == res.Name {
			m.Resources[i] = res
			return
		}
	}
	m.Resources = append(m.Resources, res)
}

func hashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
