package generator

import (
	"strings"
	"testing"

	"github.com/karurosux/go-on-rails/internal/render"
)

func TestMailerTemplatesRender(t *testing.T) {
	data := struct {
		AppName, Name, Pascal, Subject string
	}{"myapp", "welcome", "Welcome", "Welcome"}

	for _, tmpl := range []string{"mailer/mailer.go.tmpl", "mailer/mail.go.tmpl", "mailer/mail_test.go.tmpl"} {
		if _, err := render.File(tmpl, data); err != nil {
			t.Errorf("%s: %v", tmpl, err)
		}
	}
	for _, tmpl := range []string{"mailer/layout.html.tmpl", "mailer/mail.html.tmpl"} {
		out, err := render.FileDelims(tmpl, data)
		if err != nil {
			t.Errorf("%s: %v", tmpl, err)
			continue
		}
		// gor-time values substituted, runtime directives preserved
		if !strings.Contains(string(out), "myapp") && tmpl == "mailer/layout.html.tmpl" {
			t.Errorf("%s: AppName not substituted", tmpl)
		}
		if !strings.Contains(string(out), "{{") {
			t.Errorf("%s: runtime template directives were lost", tmpl)
		}
	}
	if _, err := render.File("mailer/snippets/compose_service.tmpl", struct{ SMTPPort, UIPort int }{2525, 8026}); err != nil {
		t.Errorf("compose snippet: %v", err)
	}
}

func TestMailPortsStable(t *testing.T) {
	s1, u1 := mailPorts("blog")
	s2, u2 := mailPorts("blog")
	if s1 != s2 || u1 != u2 {
		t.Error("mailPorts not deterministic")
	}
	s3, _ := mailPorts("shop")
	if s1 == s3 {
		t.Error("expected different apps to get different SMTP ports (hash collision in test names?)")
	}
}
