package generator

import (
	"testing"

	"github.com/karurosux/go-on-rails/internal/render"
)

func TestJobsTemplatesRender(t *testing.T) {
	setup := struct{ Module, AppName string }{"myapp", "myapp"}
	for _, tmpl := range []string{
		"jobs/jobs.go.tmpl",
		"jobs/client.go.tmpl",
		"jobs/worker_main.go.tmpl",
		"jobs/air.worker.toml.tmpl",
	} {
		if _, err := render.File(tmpl, setup); err != nil {
			t.Errorf("%s: %v", tmpl, err)
		}
	}

	job := struct{ Module, Name, Pascal, Every, Cron string }{"myapp", "send_welcome_email", "SendWelcomeEmail", "1h", "0 9 * * *"}
	for _, tmpl := range []string{
		"jobs/job.go.tmpl",
		"jobs/job_test.go.tmpl",
		"jobs/snippets/worker_register.tmpl",
		"jobs/snippets/periodic_register.tmpl",
		"jobs/snippets/cron_register.tmpl",
	} {
		if _, err := render.File(tmpl, job); err != nil {
			t.Errorf("%s: %v", tmpl, err)
		}
	}
}
