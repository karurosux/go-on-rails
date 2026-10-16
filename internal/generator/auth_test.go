package generator

import (
	"testing"

	"github.com/karurosux/go-on-rails/internal/render"
)

// Every auth template must render, and the Go ones must be valid Go
// (render.File gofmt-checks them).
func TestAuthTemplatesRender(t *testing.T) {
	data := struct{ Module string }{"myapp"}
	for _, tmpl := range []string{
		"auth/migration.sql.tmpl",
		"auth/users.sql.tmpl",
		"auth/sessions.sql.tmpl",
		"auth/auth.go.tmpl",
		"auth/middleware.go.tmpl",
		"auth/handler.go.tmpl",
		"auth/password.go.tmpl",
		"auth/handler_test.go.tmpl",
		"auth/web/api.ts.tmpl",
		"auth/web/schema.ts.tmpl",
		"auth/web/LoginPage.tsx.tmpl",
		"auth/web/RegisterPage.tsx.tmpl",
		"auth/web/RequireAuth.tsx.tmpl",
		"auth/web/gates.tsx.tmpl",
		"auth/web/AuthNav.tsx.tmpl",
		"auth/snippets/routes_block.tmpl",
		"auth/snippets/routes_import.tmpl",
		"auth/snippets/web_imports.tmpl",
		"auth/snippets/web_routes.tmpl",
	} {
		if _, err := render.File(tmpl, data); err != nil {
			t.Errorf("%s: %v", tmpl, err)
		}
	}
}

func TestProtectedResourceSnippets(t *testing.T) {
	r := BuildResource("myapp", "note", sampleFields(t), false)
	r.Protected = true
	r.RouteGroup = "protected"
	out, err := render.File("resource/snippets/routes_register.tmpl", r)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "note.Register(protected, q)\n" {
		t.Errorf("unexpected register line: %q", out)
	}
}
