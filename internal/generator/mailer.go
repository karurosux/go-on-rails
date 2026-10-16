package generator

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// mailPorts derives stable host ports for the Mailpit container from the
// app name, so several gor apps can run their mail catchers side by side.
func mailPorts(appName string) (smtp, ui int) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(appName))
	n := int(h.Sum32() % 500)
	return 2525 + n, 8026 + n
}

// tmplFile is a template to render plus where it lands; delims selects the
// [[ ]] gor-time delimiters for files containing runtime {{ }} syntax.
type tmplFile struct {
	template string
	path     string
	delims   bool
	data     any
}

// HasMailer reports whether the app has the mailer package.
func HasMailer(root string) bool {
	return render.Exists(filepath.Join(root, "internal", "mailer", "mailer.go"))
}

// ensureMailerSetup creates the mailer package core (client + layout) if
// missing and wires Mailpit + SMTP env defaults, idempotently.
func ensureMailerSetup(root, appName string) error {
	smtpPort, uiPort := mailPorts(appName)
	if !HasMailer(root) {
		data := struct{ AppName string }{appName}
		for _, f := range []tmplFile{
			{"mailer/mailer.go.tmpl", filepath.Join("internal", "mailer", "mailer.go"), false, data},
			{"mailer/layout.html.tmpl", filepath.Join("internal", "mailer", "templates", "layout.html.tmpl"), true, data},
		} {
			content, err := renderTmplFile(f)
			if err != nil {
				return err
			}
			if err := render.Write(filepath.Join(root, f.path), content, false); err != nil {
				return err
			}
			fmt.Printf("  create  %s\n", f.path)
		}
	}
	return wireMailpit(root, smtpPort, uiPort)
}

func renderTmplFile(f tmplFile) ([]byte, error) {
	if f.delims {
		return render.FileDelims(f.template, f.data)
	}
	return render.File(f.template, f.data)
}

// Mailer generates one transactional email (and the mailer package itself
// on first use): a Send<Name> function, an HTML template rendered in a
// shared layout, and Mailpit in docker-compose as the dev mail catcher.
func Mailer(root, module, appName, name string, force bool) error {
	_, uiPort := mailPorts(appName)

	mailData := struct {
		AppName string
		Name    string
		Pascal  string
		Subject string
	}{AppName: appName, Name: name, Pascal: fields.Pascal(name), Subject: fields.Humanize(name)}

	var files []tmplFile
	files = append(files,
		tmplFile{"mailer/mail.go.tmpl", filepath.Join("internal", "mailer", name+".go"), false, mailData},
		tmplFile{"mailer/mail_test.go.tmpl", filepath.Join("internal", "mailer", name+"_test.go"), false, mailData},
		tmplFile{"mailer/mail.html.tmpl", filepath.Join("internal", "mailer", "templates", name+".html.tmpl"), true, mailData},
		tmplFile{"mailer/mail.txt.tmpl", filepath.Join("internal", "mailer", "templates", name+".txt.tmpl"), true, mailData},
	)

	if !force {
		var conflicts []string
		for _, f := range files {
			if render.Exists(filepath.Join(root, f.path)) {
				conflicts = append(conflicts, f.path)
			}
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("these files already exist (use --force to overwrite):\n  %s",
				strings.Join(conflicts, "\n  "))
		}
	}

	if err := ensureMailerSetup(root, appName); err != nil {
		return err
	}
	for _, f := range files {
		content, err := renderTmplFile(f)
		if err != nil {
			return err
		}
		if err := render.Write(filepath.Join(root, f.path), content, force); err != nil {
			return err
		}
		fmt.Printf("  create  %s\n", f.path)
	}
	if err := tidy(root); err != nil {
		return err
	}

	fmt.Printf(`
Mailer %[1]s generated. Send it with:

  mailer.Send%[2]s(ctx, user.Email, mailer.%[2]sData{Name: user.Name})

For emails that must not get lost, call it from a background job (River
retries failed deliveries).

Dev emails are caught by Mailpit: docker compose up -d, then open
http://localhost:%[3]d — nothing is really delivered in development.
`, name, mailData.Pascal, uiPort)
	return nil
}

// wireMailpit adds the Mailpit service to docker-compose.yml and SMTP
// defaults to the env files, idempotently.
func wireMailpit(root string, smtpPort, uiPort int) error {
	portData := struct{ SMTPPort, UIPort int }{smtpPort, uiPort}

	composePath := filepath.Join(root, "docker-compose.yml")
	if raw, err := os.ReadFile(composePath); err == nil {
		content := string(raw)
		switch {
		case strings.Contains(content, "mailpit"):
			// already wired
		case strings.Contains(content, "# gor:services"):
			block, err := render.File("mailer/snippets/compose_service.tmpl", portData)
			if err != nil {
				return err
			}
			changed, err := inject.InsertBefore(composePath, "# gor:services", strings.TrimRight(string(block), "\n"))
			if err != nil {
				return err
			}
			if changed {
				fmt.Println("  inject  docker-compose.yml (Mailpit dev mail catcher)")
			}
		default:
			fmt.Printf(`
Your docker-compose.yml predates the # gor:services marker — add Mailpit
yourself to catch dev email:

  mail:
    image: axllent/mailpit:latest
    ports:
      - "%d:1025"
      - "%d:8025"
`, smtpPort, uiPort)
		}
	}

	return ensureEnvVars(root, "SMTP_ADDR", fmt.Sprintf(`
# Mail — dev defaults point at Mailpit (docker compose up -d, UI at http://localhost:%d)
SMTP_ADDR=localhost:%d
MAIL_FROM=no-reply@localhost
# SMTP_USERNAME=
# SMTP_PASSWORD=
`, uiPort, smtpPort))
}
