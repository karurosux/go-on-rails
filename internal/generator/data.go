package generator

import (
	"fmt"
	"strings"

	"github.com/karurosux/go-on-rails/internal/fields"
)

// Resource is the data passed to every resource template.
type Resource struct {
	Module           string // app module path
	Name             string // singular snake: "post", "blog_post"
	PkgName          string // Go package: "post", "blogpost"
	NeedsAlias       bool   // import alias needed (name contains "_")
	Pascal           string // Post, BlogPost
	Camel            string // post, blogPost
	Plural           string // posts, blog_posts
	PluralPascal     string // Posts, BlogPosts
	Table            string // == Plural
	Human            string // "Blog post"
	HumanLower       string // "blog post"
	PluralHuman      string // "Blog posts"
	PluralHumanLower string // "blog posts"
	Service          bool
	Protected        bool
	RouteGroup       string // "api", or "protected" for --protected resources
	HasRequired      bool
	HasTimestamp     bool
	HasBool          bool
	HasTextarea      bool
	SingleField      bool
	Fields           []FieldData

	CreateCols         string // "title, body, published"
	CreatePlaceholders string // "$1, $2, $3"
}

// FieldData is one field, pre-mapped for every template.
type FieldData struct {
	Column    string // author_id
	GoField   string // AuthorID (matches sqlc)
	GoType    string
	GoPtrType string
	JSONName  string // authorId (matches sqlc json_tags_case_style: camel)
	TSName    string // authorId
	TSType    string
	Zod       string
	PgType    string
	Input     fields.Input
	Required  bool
	IsRef     bool
	RefTable  string
	Label     string // "Author id"

	// Pre-built TSX expressions, so templates stay declarative.
	DefaultTS  string // defaultValues entry
	EditExpr   string // reset() value when editing
	SubmitExpr string // API payload value
	CellExpr   string // list table cell
	DetailExpr string // detail page value
}

// BuildResource maps a parsed resource definition onto template data.
func BuildResource(module, name string, fs []fields.Field, service bool) Resource {
	r := Resource{
		Module:       module,
		Name:         name,
		PkgName:      fields.PkgName(name),
		NeedsAlias:   strings.Contains(name, "_"),
		Pascal:       fields.Pascal(name),
		Camel:        fields.Camel(name),
		Plural:       fields.Plural(name),
		PluralPascal: fields.Pascal(fields.Plural(name)),
		Service:      service,
		RouteGroup:   "api",
		SingleField:  len(fs) == 1,
	}
	r.Table = r.Plural
	r.Human = fields.Humanize(name)
	r.HumanLower = strings.ToLower(r.Human)
	r.PluralHuman = fields.Humanize(r.Plural)
	r.PluralHumanLower = strings.ToLower(r.PluralHuman)

	cols := make([]string, 0, len(fs))
	phs := make([]string, 0, len(fs))
	for i, f := range fs {
		m := f.Mapping()
		fd := FieldData{
			Column:    f.Column(),
			GoField:   fields.Pascal(f.Column()),
			GoType:    m.Go,
			GoPtrType: m.GoPtr,
			JSONName:  fields.Camel(f.Column()),
			TSName:    fields.Camel(f.Column()),
			TSType:    m.TS,
			Zod:       m.Zod,
			PgType:    m.Postgres,
			Input:     m.Input,
			Required:  m.Required,
			IsRef:     f.Type == fields.TypeReferences,
			RefTable:  f.RefTable(),
			Label:     fields.Humanize(f.Column()),
		}
		fillTSX(&fd, f.Type)

		if m.Required {
			r.HasRequired = true
		}
		if f.Type == fields.TypeTimestamp {
			r.HasTimestamp = true
		}
		if f.Type == fields.TypeBool {
			r.HasBool = true
		}
		if m.Input == fields.InputTextarea {
			r.HasTextarea = true
		}
		cols = append(cols, fd.Column)
		phs = append(phs, fmt.Sprintf("$%d", i+1))
		r.Fields = append(r.Fields, fd)
	}
	r.CreateCols = strings.Join(cols, ", ")
	r.CreatePlaceholders = strings.Join(phs, ", ")
	return r
}

// fillTSX precomputes the TSX expressions each frontend template drops in.
func fillTSX(fd *FieldData, t fields.Type) {
	item := "item." + fd.TSName
	data := "existing.data." + fd.TSName
	val := "values." + fd.TSName

	switch t {
	case fields.TypeBool:
		fd.DefaultTS = "false"
		fd.CellExpr = fmt.Sprintf(`{%s ? "Yes" : "No"}`, item)
	case fields.TypeInt, fields.TypeBigint:
		fd.DefaultTS = "0"
		fd.CellExpr = fmt.Sprintf("{%s}", item)
	case fields.TypeTimestamp:
		fd.DefaultTS = `""`
		fd.CellExpr = fmt.Sprintf("{new Date(%s).toLocaleString()}", item)
	default:
		fd.DefaultTS = `""`
		fd.CellExpr = fmt.Sprintf("{%s}", item)
	}
	fd.DetailExpr = fd.CellExpr

	if t == fields.TypeTimestamp {
		// datetime-local inputs want "YYYY-MM-DDTHH:mm"; the API wants RFC 3339.
		fd.EditExpr = data + ".slice(0, 16)"
		fd.SubmitExpr = fmt.Sprintf("new Date(%s).toISOString()", val)
	} else {
		fd.EditExpr = data
		fd.SubmitExpr = val
	}
}
