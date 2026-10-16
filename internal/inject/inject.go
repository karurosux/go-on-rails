// Package inject inserts generated code blocks into existing files at
// marker comments (e.g. // gor:routes), idempotently.
package inject

import (
	"fmt"
	"go/format"
	"os"
	"strings"
)

// InsertBefore inserts block into the file at path, immediately before the
// line containing marker, indented to match the marker line. It is
// idempotent: if the file already contains the block's first non-empty line
// (trimmed), nothing is written and it returns false.
func InsertBefore(path, marker, block string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	content := string(raw)

	key := firstNonEmptyLine(block)
	if key == "" {
		return false, fmt.Errorf("inject: empty block")
	}
	if containsLine(content, key) {
		return false, nil // already injected
	}

	lines := strings.Split(content, "\n")
	idx := -1
	indent := ""
	for i, line := range lines {
		if strings.Contains(line, marker) {
			idx = i
			indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			break
		}
	}
	if idx == -1 {
		return false, fmt.Errorf("marker %q not found in %s — was it removed? Re-add it where generated code should be inserted", marker, path)
	}

	var blockLines []string
	for _, bl := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		if strings.TrimSpace(bl) == "" {
			blockLines = append(blockLines, "")
			continue
		}
		blockLines = append(blockLines, indent+bl)
	}

	out := make([]string, 0, len(lines)+len(blockLines))
	out = append(out, lines[:idx]...)
	out = append(out, blockLines...)
	out = append(out, lines[idx:]...)
	result := strings.Join(out, "\n")

	if strings.HasSuffix(path, ".go") {
		formatted, err := format.Source([]byte(result))
		if err != nil {
			return false, fmt.Errorf("injection into %s produced invalid Go (this is a gor bug): %w", path, err)
		}
		result = string(formatted)
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(result), info.Mode().Perm())
}

func firstNonEmptyLine(block string) string {
	for _, l := range strings.Split(block, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

func containsLine(content, key string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == key {
			return true
		}
	}
	return false
}
