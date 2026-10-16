package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEnvVars(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte("PORT=8080\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PORT=9999\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	block := "# Mail\nSMTP_ADDR=localhost:1025\n"
	if err := ensureEnvVars(root, "SMTP_ADDR", block); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{".env.example", ".env"} {
		raw, _ := os.ReadFile(filepath.Join(root, f))
		if !strings.Contains(string(raw), "SMTP_ADDR=localhost:1025") {
			t.Errorf("%s missing appended vars:\n%s", f, raw)
		}
		if !strings.Contains(string(raw), "PORT=") {
			t.Errorf("%s lost existing content", f)
		}
	}

	// Idempotent: a second call appends nothing.
	before, _ := os.ReadFile(filepath.Join(root, ".env"))
	if err := ensureEnvVars(root, "SMTP_ADDR", block); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(root, ".env"))
	if string(before) != string(after) {
		t.Error("second call changed .env")
	}

	// A user who already set the var keeps their value untouched.
	root2 := t.TempDir()
	custom := "SMTP_ADDR=smtp.resend.com:587\n"
	os.WriteFile(filepath.Join(root2, ".env.example"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(root2, ".env"), []byte(custom), 0o644)
	if err := ensureEnvVars(root2, "SMTP_ADDR", block); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(root2, ".env"))
	if strings.Contains(string(raw), "localhost:1025") {
		t.Error("overrode user's existing SMTP_ADDR in .env")
	}

	// No .env at all: only .env.example is written, .env is not created.
	root3 := t.TempDir()
	if err := ensureEnvVars(root3, "SMTP_ADDR", block); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root3, ".env")); !os.IsNotExist(err) {
		t.Error(".env was created for the user")
	}
	if raw, err := os.ReadFile(filepath.Join(root3, ".env.example")); err != nil || !strings.Contains(string(raw), "SMTP_ADDR") {
		t.Error(".env.example was not created/filled")
	}
}
