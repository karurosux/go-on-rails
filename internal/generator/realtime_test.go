package generator

import (
	"testing"

	"github.com/karurosux/go-on-rails/internal/render"
)

func TestRealtimeTemplatesRender(t *testing.T) {
	for _, protected := range []bool{false, true} {
		data := struct {
			Module    string
			Protected bool
		}{"myapp", protected}
		for _, tmpl := range []string{
			"realtime/realtime.go.tmpl",
			"realtime/web/realtime.ts.tmpl",
			"realtime/snippets/routes_register.tmpl",
			"realtime/snippets/routes_import.tmpl",
		} {
			if _, err := render.File(tmpl, data); err != nil {
				t.Errorf("protected=%v %s: %v", protected, tmpl, err)
			}
		}
	}
}
