package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karurosux/go-on-rails/internal/inject"
	"github.com/karurosux/go-on-rails/internal/render"
)

// AuthEmails generates password reset and email verification on top of
// auth: tokens table, endpoints, the two emails, and the frontend pages.
// A password reset signs the user out everywhere.
func AuthEmails(root, module, appName string, apiOnly, force bool) error {
	if !HasAuth(root) {
		return fmt.Errorf("authentication is not set up yet: run `gor g auth` first")
	}
	if err := ensureMailerSetup(root, appName); err != nil {
		return err
	}

	data := struct {
		Module  string
		AppName string
	}{Module: module, AppName: appName}

	// The schema migration is created once; --force regenerates code but
	// keeps migration history.
	migration := ""
	already := false
	entries, _ := os.ReadDir(filepath.Join(root, "migrations"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_add_email_flows.sql") {
			already = true
			if !force {
				migration = filepath.Join("migrations", e.Name())
			}
			break
		}
	}
	if !already {
		migration = uniqueMigrationPath(root, "add_email_flows")
	}

	files := []tmplFile{
		{"authemails/queries.sql.tmpl", filepath.Join("queries", "auth_tokens.sql"), false, data},
		{"authemails/emails.go.tmpl", filepath.Join("internal", "auth", "emails.go"), false, data},
		{"authemails/password_reset.go.tmpl", filepath.Join("internal", "mailer", "password_reset.go"), false, data},
		{"authemails/password_reset.html.tmpl", filepath.Join("internal", "mailer", "templates", "password_reset.html.tmpl"), true, data},
		{"authemails/password_reset.txt.tmpl", filepath.Join("internal", "mailer", "templates", "password_reset.txt.tmpl"), true, data},
		{"authemails/verify_email.go.tmpl", filepath.Join("internal", "mailer", "verify_email.go"), false, data},
		{"authemails/verify_email.html.tmpl", filepath.Join("internal", "mailer", "templates", "verify_email.html.tmpl"), true, data},
		{"authemails/verify_email.txt.tmpl", filepath.Join("internal", "mailer", "templates", "verify_email.txt.tmpl"), true, data},
	}
	if migration != "" && !already {
		files = append([]tmplFile{{"authemails/migration.sql.tmpl", migration, false, data}}, files...)
	}
	if !apiOnly {
		feat := filepath.Join("web", "src", "features", "auth")
		files = append(files,
			tmplFile{"authemails/web/account.ts.tmpl", filepath.Join("web", "src", "api", "account.ts"), false, data},
			tmplFile{"authemails/web/ForgotPasswordPage.tsx.tmpl", filepath.Join(feat, "ForgotPasswordPage.tsx"), false, data},
			tmplFile{"authemails/web/ResetPasswordPage.tsx.tmpl", filepath.Join(feat, "ResetPasswordPage.tsx"), false, data},
			tmplFile{"authemails/web/VerifyEmailPage.tsx.tmpl", filepath.Join(feat, "VerifyEmailPage.tsx"), false, data},
		)
	}

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

	routesGo := filepath.Join(root, "internal", "routes", "routes.go")
	changed, err := inject.InsertBefore(routesGo, "// gor:routes", "auth.RegisterEmailRoutes(api, q)")
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("  inject  internal/routes/routes.go")
	}

	if !apiOnly {
		routesTsx := filepath.Join(root, "web", "src", "routes.tsx")
		imports, err := render.File("authemails/snippets/web_imports.tmpl", nil)
		if err != nil {
			return err
		}
		if _, err := inject.InsertBefore(routesTsx, "// gor:imports", strings.TrimRight(string(imports), "\n")); err != nil {
			return err
		}
		routeBlock, err := render.File("authemails/snippets/web_routes.tmpl", nil)
		if err != nil {
			return err
		}
		changed, err := inject.InsertBefore(routesTsx, "{/* gor:routes */}", strings.TrimRight(string(routeBlock), "\n"))
		if err != nil {
			return err
		}
		if changed {
			fmt.Println("  inject  web/src/routes.tsx")
		}
	}

	if err := ensureEnvVars(root, "APP_URL", `
# Public base URL used in email links (auth flows)
APP_URL=http://localhost:5173
`); err != nil {
		return err
	}
	if err := runCodegen(root); err != nil {
		return err
	}

	fmt.Print(`
Password reset + email verification generated.

  Endpoints: POST /api/auth/forgot-password | reset-password | verify-email
  Pages:     /forgot-password, /reset-password, /verify-email
  A password reset revokes every session ("sign out everywhere").

Two one-line integrations are yours to add (gor never edits your code):

  1. internal/auth/handler.go, in register(), after the user is created:
       _ = SendVerificationEmail(c.Request().Context(), h.q, user)
  2. web/src/features/auth/LoginPage.tsx, under the password field:
       <Link to="/forgot-password" className="text-sm text-muted-foreground underline underline-offset-4">Forgot password?</Link>

Apply the migration with: gor db migrate
`)
	return nil
}
