// Package deps checks for the external tools gor shells out to and
// produces actionable install instructions when they are missing.
package deps

import (
	"fmt"
	"os/exec"
	"strings"
)

// Tool describes an external dependency.
type Tool struct {
	Name    string
	Why     string
	Install string // one-line install command
	URL     string
}

var tools = map[string]Tool{
	"go": {
		Name:    "go",
		Why:     "compiles the backend",
		Install: "brew install go",
		URL:     "https://go.dev/dl/",
	},
	"node": {
		Name:    "node",
		Why:     "runs the Vite frontend",
		Install: "brew install node",
		URL:     "https://nodejs.org",
	},
	"npm": {
		Name:    "npm",
		Why:     "installs frontend packages (ships with Node)",
		Install: "brew install node",
		URL:     "https://nodejs.org",
	},
	"sqlc": {
		Name:    "sqlc",
		Why:     "generates type-safe Go from SQL queries",
		Install: "go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest",
		URL:     "https://docs.sqlc.dev/en/latest/overview/install.html",
	},
	"goose": {
		Name:    "goose",
		Why:     "runs database migrations",
		Install: "go install github.com/pressly/goose/v3/cmd/goose@latest",
		URL:     "https://github.com/pressly/goose",
	},
	"air": {
		Name:    "air",
		Why:     "hot-reloads the Go server in development",
		Install: "go install github.com/air-verse/air@latest",
		URL:     "https://github.com/air-verse/air",
	},
	"git": {
		Name:    "git",
		Why:     "initializes the project repository",
		Install: "brew install git",
		URL:     "https://git-scm.com",
	},
	"docker": {
		Name:    "docker",
		Why:     "runs the development Postgres (optional)",
		Install: "brew install --cask docker",
		URL:     "https://docs.docker.com/get-docker/",
	},
}

// Have reports whether the named tool is on PATH.
func Have(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Check returns an error with install instructions if any named tool is missing.
func Check(names ...string) error {
	var missing []Tool
	for _, n := range names {
		if !Have(n) {
			t, ok := tools[n]
			if !ok {
				t = Tool{Name: n}
			}
			missing = append(missing, t)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("missing required tools:\n")
	for _, t := range missing {
		fmt.Fprintf(&b, "\n  %s — %s\n    install: %s\n    docs:    %s\n", t.Name, t.Why, t.Install, t.URL)
	}
	if hasGoInstall(missing) {
		b.WriteString("\nNote: `go install` puts binaries in $(go env GOPATH)/bin — make sure it is on your PATH.")
	}
	return fmt.Errorf("%s", b.String())
}

func hasGoInstall(ts []Tool) bool {
	for _, t := range ts {
		if strings.HasPrefix(t.Install, "go install") {
			return true
		}
	}
	return false
}

// Known returns the tool metadata for `gor doctor`.
func Known(name string) (Tool, bool) {
	t, ok := tools[name]
	return t, ok
}
