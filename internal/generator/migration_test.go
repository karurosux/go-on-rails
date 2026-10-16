package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Two resources generated within the same second must not share a goose
// version — goose panics on duplicate versions.
func TestMigrationPathAvoidsDuplicateVersions(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	version := time.Now().UTC().Format("20060102150405")
	if err := os.WriteFile(filepath.Join(dir, version+"_create_posts.sql"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got := migrationPath(root, "authors", false)
	base := filepath.Base(got)
	if !strings.HasSuffix(base, "_create_authors.sql") {
		t.Fatalf("unexpected migration name %q", base)
	}
	if strings.HasPrefix(base, version+"_") {
		t.Errorf("new migration %q reuses existing version %s", base, version)
	}
}

func TestMigrationPathReturnsExistingWithoutForce(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "20240101000000_create_posts.sql"
	if err := os.WriteFile(filepath.Join(dir, existing), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := migrationPath(root, "posts", false); filepath.Base(got) != existing {
		t.Errorf("expected existing migration to be reported as conflict, got %q", got)
	}
	if got := migrationPath(root, "posts", true); got != "" {
		t.Errorf("with force, expected skip (empty path), got %q", got)
	}
}
