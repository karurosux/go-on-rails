// Package project locates a gor application root and reads its metadata.
package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Root walks up from the current directory looking for a gor app root,
// identified by go.mod + sqlc.yaml living side by side.
func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "sqlc.yaml")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a gor app (no go.mod + sqlc.yaml found walking up from the current directory)\nCreate one with: gor new <name>")
		}
		dir = parent
	}
}

// Module reads the module path from the app's go.mod.
func Module(root string) (string, error) {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module line in %s/go.mod", root)
}

// AppName is the binary/app name: the last element of the module path.
func AppName(root string) (string, error) {
	mod, err := Module(root)
	if err != nil {
		return "", err
	}
	return filepath.Base(mod), nil
}

// Port returns the app server's port: PORT from the environment or the
// app's .env, defaulting to 8080 (matching internal/config).
func Port(root string) string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	if env, err := godotenv.Read(filepath.Join(root, ".env")); err == nil && env["PORT"] != "" {
		return env["PORT"]
	}
	return "8080"
}

// DatabaseURL returns DATABASE_URL from the environment, falling back to the
// app's .env file.
func DatabaseURL(root string) (string, error) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url, nil
	}
	env, err := godotenv.Read(filepath.Join(root, ".env"))
	if err == nil && env["DATABASE_URL"] != "" {
		return env["DATABASE_URL"], nil
	}
	return "", fmt.Errorf("DATABASE_URL is not set\nCopy .env.example to .env and adjust it, or export DATABASE_URL")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
