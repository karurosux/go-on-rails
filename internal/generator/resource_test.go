package generator

import (
	"strings"
	"testing"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/render"
)

func sampleFields(t *testing.T) []fields.Field {
	t.Helper()
	fs, err := fields.Parse([]string{
		"title:string", "body:text", "views:int", "big:bigint",
		"published:bool", "price:decimal", "published_at:timestamp",
		"token:uuid", "author:references",
	})
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

// TestTemplatesRenderValidGo renders every Go-producing template with every
// flag combination; render.File gofmt-checks the output, so a template that
// produces invalid Go fails here instead of in a user's project.
func TestTemplatesRenderValidGo(t *testing.T) {
	for _, service := range []bool{false, true} {
		for _, single := range []bool{false, true} {
			fs := sampleFields(t)
			if single {
				fs = fs[:1]
			}
			r := BuildResource("myapp", "post", fs, service)
			for _, tmpl := range []string{
				"resource/handler.go.tmpl",
				"resource/handler_test.go.tmpl",
				"resource/service.go.tmpl",
			} {
				if _, err := render.File(tmpl, r); err != nil {
					t.Errorf("service=%v single=%v %s: %v", service, single, tmpl, err)
				}
			}
		}
	}
}

func TestTemplatesRenderNonGo(t *testing.T) {
	r := BuildResource("myapp", "blog_post", sampleFields(t), false)
	for _, tmpl := range []string{
		"resource/migration.sql.tmpl",
		"resource/queries.sql.tmpl",
		"resource/web/api.ts.tmpl",
		"resource/web/schema.ts.tmpl",
		"resource/web/List.tsx.tmpl",
		"resource/web/Detail.tsx.tmpl",
		"resource/web/Form.tsx.tmpl",
		"resource/snippets/routes_import.tmpl",
		"resource/snippets/routes_register.tmpl",
		"resource/snippets/web_imports.tmpl",
		"resource/snippets/web_routes.tmpl",
		"resource/snippets/web_nav.tmpl",
	} {
		out, err := render.File(tmpl, r)
		if err != nil {
			t.Errorf("%s: %v", tmpl, err)
			continue
		}
		if strings.Contains(string(out), "<no value>") {
			t.Errorf("%s: rendered a missing template variable (<no value>)", tmpl)
		}
	}
}

func TestHeadlessFormRenders(t *testing.T) {
	r := BuildResource("myapp", "post", sampleFields(t), false)
	r.Headless = true
	out, err := render.File("resource/web/Form.tsx.tmpl", r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "useNavigate") || strings.Contains(s, "useParams") || strings.Contains(s, "react-router") {
		t.Error("headless Form must not depend on the router")
	}
	if !strings.Contains(s, "onSuccess?: (item: Post) => void") {
		t.Error("headless Form should expose an onSuccess prop")
	}
}

func TestBuildResource(t *testing.T) {
	r := BuildResource("myapp", "blog_post", sampleFields(t), true)

	if r.PkgName != "blogpost" || !r.NeedsAlias {
		t.Errorf("PkgName/NeedsAlias wrong: %q %v", r.PkgName, r.NeedsAlias)
	}
	if r.Pascal != "BlogPost" || r.PluralPascal != "BlogPosts" || r.Plural != "blog_posts" {
		t.Errorf("naming wrong: %q %q %q", r.Pascal, r.PluralPascal, r.Plural)
	}
	if !r.HasRequired || !r.HasTimestamp || r.SingleField {
		t.Errorf("flags wrong: %+v", r)
	}
	if r.CreateCols != "title, body, views, big, published, price, published_at, token, author_id" {
		t.Errorf("CreateCols = %q", r.CreateCols)
	}
	if r.CreatePlaceholders != "$1, $2, $3, $4, $5, $6, $7, $8, $9" {
		t.Errorf("CreatePlaceholders = %q", r.CreatePlaceholders)
	}

	author := r.Fields[8]
	if author.Column != "author_id" || author.GoField != "AuthorID" || author.TSName != "authorId" {
		t.Errorf("references field mapped wrong: %+v", author)
	}
	if !author.IsRef || author.RefTable != "authors" {
		t.Errorf("references metadata wrong: %+v", author)
	}

	ts := r.Fields[6]
	if ts.SubmitExpr != "new Date(values.publishedAt).toISOString()" {
		t.Errorf("timestamp SubmitExpr = %q", ts.SubmitExpr)
	}
	if ts.EditExpr != "existing.data.publishedAt.slice(0, 16)" {
		t.Errorf("timestamp EditExpr = %q", ts.EditExpr)
	}
}

// TestProjectTemplatesRender renders every project template (gofmt-checking
// the Go ones) so `gor new` can't fail on a template bug at runtime.
func TestProjectTemplatesRender(t *testing.T) {
	data := struct {
		AppName, Module string
		DBPort          int
		GoVersion       string
		NodeMajor       int
	}{"myapp", "myapp", 5433, "1.25", 22}
	for _, tmpl := range []string{
		"project/cmd/server/main.go.tmpl",
		"project/embed.go.tmpl",
		"project/internal/config/config.go.tmpl",
		"project/internal/db/db.go.tmpl",
		"project/internal/db/gen/db.go.tmpl",
		"project/internal/httpx/httpx.go.tmpl",
		"project/internal/middleware/middleware.go.tmpl",
		"project/internal/routes/routes.go.tmpl",
		"project/go.mod.tmpl",
		"project/sqlc.yaml.tmpl",
		"project/.air.toml.tmpl",
		"project/.nvmrc.tmpl",
		"project/.env.example.tmpl",
		"project/docker-compose.yml.tmpl",
		"project/Dockerfile.tmpl",
		"project/.dockerignore.tmpl",
		"project/Makefile.tmpl",
		"project/Procfile.dev.tmpl",
		"project/.gitignore.tmpl",
		"project/README.md.tmpl",
		"project/AGENTS.md.tmpl",
		"project/CLAUDE.md.tmpl",
		"project/cmd/seed/main.go.tmpl",
		"project/web/package.json.tmpl",
		"project/web/.npmrc.tmpl",
		"project/web/components.json.tmpl",
		"project/web/vite.config.ts.tmpl",
		"project/web/index.html.tmpl",
		"project/web/src/vite-env.d.ts.tmpl",
		"project/web/src/main.tsx.tmpl",
		"project/web/src/App.tsx.tmpl",
		"project/web/src/routes.tsx.tmpl",
		"project/web/src/api/client.ts.tmpl",
		"project/web/src/lib/utils.ts.tmpl",
		"project/web/src/components/ui/button.tsx.tmpl",
		"project/web/src/components/ui/input.tsx.tmpl",
		"project/web/src/components/ui/textarea.tsx.tmpl",
		"project/web/src/components/ui/label.tsx.tmpl",
		"project/web/src/components/ui/checkbox.tsx.tmpl",
		"project/web/src/components/ui/table.tsx.tmpl",
		"project/web/src/components/ui/card.tsx.tmpl",
	} {
		if _, err := render.File(tmpl, data); err != nil {
			t.Errorf("%s: %v", tmpl, err)
		}
	}
}
