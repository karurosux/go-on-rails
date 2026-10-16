package fields

import (
	"strings"
	"unicode"
)

// commonInitialisms mirrors the list sqlc uses when turning column names
// into Go struct field names, so generated handler code matches sqlc output.
var commonInitialisms = map[string]bool{
	"api": true, "db": true, "dns": true, "http": true, "https": true,
	"id": true, "ip": true, "json": true, "sql": true, "ssh": true,
	"tls": true, "ttl": true, "ui": true, "uid": true, "uuid": true,
	"uri": true, "url": true, "xml": true,
}

// Pascal converts snake_case to PascalCase using sqlc-compatible
// initialisms: author_id -> AuthorID, api_key -> APIKey.
func Pascal(snake string) string {
	var b strings.Builder
	for _, part := range strings.Split(snake, "_") {
		if part == "" {
			continue
		}
		if commonInitialisms[part] {
			b.WriteString(strings.ToUpper(part))
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}

// Camel converts snake_case to camelCase: author_id -> authorId.
// This matches sqlc's json_tags_case_style: camel.
func Camel(snake string) string {
	parts := strings.Split(snake, "_")
	var b strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			b.WriteString(part)
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}

// Plural pluralizes a snake_case word with simple English rules:
// post -> posts, category -> categories, box -> boxes.
func Plural(s string) string {
	switch {
	case s == "":
		return s
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

func isVowel(r rune) bool {
	return strings.ContainsRune("aeiou", unicode.ToLower(r))
}

// Humanize turns snake_case into a readable label: author_id -> "Author id",
// blog_post -> "Blog post".
func Humanize(snake string) string {
	s := strings.ReplaceAll(snake, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// PkgName turns a resource name into a Go package name: blog_post -> blogpost.
func PkgName(snake string) string {
	return strings.ReplaceAll(snake, "_", "")
}
