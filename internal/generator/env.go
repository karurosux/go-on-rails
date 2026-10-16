package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureEnvVars appends an env block to the app's .env.example and — when
// the file exists — .env, so generated code is configured out of the box.
// The block is skipped wherever sentinel (the first key) already appears.
//
// Every generator that introduces configuration must declare it through
// this helper; apps should never need hand-edited env files to run what
// gor generated.
func ensureEnvVars(root, sentinel, block string) error {
	if !strings.HasPrefix(block, "\n") {
		block = "\n" + block
	}
	for _, envFile := range []string{".env.example", ".env"} {
		path := filepath.Join(root, envFile)
		raw, err := os.ReadFile(path)
		if err != nil {
			if envFile == ".env" {
				continue // .env is the user's copy; don't create it for them
			}
			raw = nil // .env.example should always exist — (re)create it
		}
		if strings.Contains(string(raw), sentinel) {
			continue
		}
		if err := os.WriteFile(path, append(raw, []byte(block)...), 0o644); err != nil {
			return err
		}
		fmt.Printf("  append  %s\n", envFile)
	}
	return nil
}
