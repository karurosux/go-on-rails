// Package fields parses `name:type` field definitions and owns the
// type-mapping table used by every generator (SQL, Go, TypeScript, Zod).
package fields

import (
	"fmt"
	"regexp"
	"strings"
)

// Field is one parsed `name:type` argument.
type Field struct {
	Name string // as given, snake_case: "title", or "author" for author:references
	Type Type
}

// Column is the database column name. A references field named "author"
// becomes the column "author_id".
func (f Field) Column() string {
	if f.Type == TypeReferences {
		return f.Name + "_id"
	}
	return f.Name
}

// RefTable is the table a references field points at ("author" -> "authors").
func (f Field) RefTable() string {
	return Plural(f.Name)
}

// Mapping returns the type mapping for this field.
func (f Field) Mapping() Mapping {
	return mappings[f.Type]
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// reserved columns are generated automatically on every resource.
var reserved = map[string]bool{"id": true, "created_at": true, "updated_at": true}

// Parse turns CLI args like ["title:string", "author:references"] into Fields.
func Parse(args []string) ([]Field, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("at least one field is required (e.g. title:string)")
	}
	seen := map[string]bool{}
	out := make([]Field, 0, len(args))
	for _, arg := range args {
		name, typ, ok := strings.Cut(arg, ":")
		if !ok || name == "" || typ == "" {
			return nil, fmt.Errorf("invalid field %q: expected name:type (e.g. title:string)", arg)
		}
		if !nameRe.MatchString(name) {
			return nil, fmt.Errorf("invalid field name %q: use lower_snake_case starting with a letter", name)
		}
		if reserved[name] {
			return nil, fmt.Errorf("field %q is added automatically to every resource", name)
		}
		f := Field{Name: name, Type: Type(typ)}
		if _, err := MappingFor(f.Type); err != nil {
			return nil, fmt.Errorf("field %q: %w", arg, err)
		}
		col := f.Column()
		if seen[col] {
			return nil, fmt.Errorf("duplicate field %q", col)
		}
		seen[col] = true
		out = append(out, f)
	}
	return out, nil
}
