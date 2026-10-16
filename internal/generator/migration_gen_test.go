package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readMigration(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected exactly one migration, got %v (err %v)", entries, err)
	}
	b, err := os.ReadFile(filepath.Join(root, "migrations", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGenerateMigrationAdd(t *testing.T) {
	root := t.TempDir()
	err := GenerateMigration(root, "myapp", "add_slug_to_posts", []string{"slug:string", "author:references"})
	if err != nil {
		t.Fatal(err)
	}
	got := readMigration(t, root)
	for _, want := range []string{
		"ALTER TABLE posts ADD COLUMN slug text NOT NULL DEFAULT ''",
		"ALTER TABLE posts ADD COLUMN author_id uuid REFERENCES authors (id);",
		"CREATE INDEX posts_author_id_idx ON posts (author_id);",
		"TODO: backfill author_id",
		"ALTER TABLE posts DROP COLUMN slug;",
		"ALTER TABLE posts DROP COLUMN author_id;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("migration missing %q:\n%s", want, got)
		}
	}
}

func TestGenerateMigrationRemove(t *testing.T) {
	root := t.TempDir()
	if err := GenerateMigration(root, "myapp", "remove_views_from_posts", []string{"views:int"}); err != nil {
		t.Fatal(err)
	}
	got := readMigration(t, root)
	if !strings.Contains(got, "-- +goose Up\nALTER TABLE posts DROP COLUMN views;") {
		t.Errorf("up should drop the column:\n%s", got)
	}
	if !strings.Contains(got, "ALTER TABLE posts ADD COLUMN views integer;") {
		t.Errorf("down should restore the column nullable:\n%s", got)
	}
}

func TestGenerateMigrationEmpty(t *testing.T) {
	root := t.TempDir()
	if err := GenerateMigration(root, "myapp", "backfill_things", nil); err != nil {
		t.Fatal(err)
	}
	got := readMigration(t, root)
	if !strings.Contains(got, "-- +goose Up") || !strings.Contains(got, "-- +goose Down") {
		t.Errorf("skeleton missing goose sections:\n%s", got)
	}
}

func TestGenerateMigrationBadNameWithFields(t *testing.T) {
	root := t.TempDir()
	err := GenerateMigration(root, "myapp", "tweak_posts", []string{"slug:string"})
	if err == nil {
		t.Fatal("expected error for unconventional name with fields")
	}
	if !strings.Contains(err.Error(), "add_<x>_to_<table>") {
		t.Errorf("error should explain the naming convention: %v", err)
	}
}
