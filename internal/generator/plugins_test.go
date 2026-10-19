package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestResolvePluginPathPassthrough(t *testing.T) {
	root := t.TempDir()
	got := resolvePluginPath(root, "internal/billing/billing.go")
	if got != "internal/billing/billing.go" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestResolvePluginPathMintsVersion(t *testing.T) {
	root := t.TempDir()
	got := resolvePluginPath(root, "migrations/{{timestamp}}_create_billing.sql")
	want := regexp.MustCompile(`^migrations/\d{14}_create_billing\.sql$`)
	if !want.MatchString(filepath.ToSlash(got)) {
		t.Fatalf("expected timestamped path, got %q", got)
	}
}

func TestResolvePluginPathReusesExisting(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "20240101000000_create_billing.sql"
	if err := os.WriteFile(filepath.Join(dir, existing), []byte("-- +goose Up\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolvePluginPath(root, "migrations/{{timestamp}}_create_billing.sql")
	if filepath.ToSlash(got) != "migrations/"+existing {
		t.Fatalf("expected reuse of %s, got %q", existing, got)
	}
}

func TestResolvePluginPathBumpsTakenVersion(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A different migration occupying some version must not be reused.
	if err := os.WriteFile(filepath.Join(dir, "20240101000000_create_posts.sql"), []byte("-- +goose Up\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolvePluginPath(root, "migrations/{{timestamp}}_create_billing.sql")
	want := regexp.MustCompile(`^migrations/\d{14}_create_billing\.sql$`)
	if !want.MatchString(filepath.ToSlash(got)) {
		t.Fatalf("expected timestamped path, got %q", got)
	}
}
