package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/karurosux/go-on-rails/internal/fields"
	"github.com/karurosux/go-on-rails/internal/render"
)

var (
	addRe    = regexp.MustCompile(`^add_[a-z0-9_]+_to_([a-z][a-z0-9_]*)$`)
	removeRe = regexp.MustCompile(`^remove_[a-z0-9_]+_from_([a-z][a-z0-9_]*)$`)
)

type migrationField struct {
	Column     string
	PgType     string
	IsRef      bool
	RefTable   string
	Nullable   bool   // added without NOT NULL (no safe default exists)
	DefaultSQL string // default used to backfill existing rows
}

type migrationData struct {
	Table  string
	Fields []migrationField
}

// GenerateMigration creates a timestamped goose migration. With fields, the
// name must follow add_<x>_to_<table> or remove_<x>_from_<table>: the SQL is
// generated and the owning resource's derived files are re-synced (module is
// the app module path, needed for that sync). Without fields it writes an
// empty skeleton.
func GenerateMigration(root, module, name string, fieldArgs []string) error {
	var tmpl, table string
	var add bool
	var data migrationData

	switch {
	case len(fieldArgs) == 0:
		tmpl = "migration/empty.sql.tmpl"
	case addRe.MatchString(name):
		tmpl = "migration/add.sql.tmpl"
		table = addRe.FindStringSubmatch(name)[1]
		add = true
	case removeRe.MatchString(name):
		tmpl = "migration/remove.sql.tmpl"
		table = removeRe.FindStringSubmatch(name)[1]
	default:
		return fmt.Errorf("to generate column changes, name the migration add_<x>_to_<table> or remove_<x>_from_<table> (e.g. add_slug_to_posts slug:string)\nWithout fields, any name creates an empty migration")
	}

	if table != "" {
		parsed, err := fields.Parse(fieldArgs)
		if err != nil {
			return err
		}
		data.Table = table
		for _, f := range parsed {
			data.Fields = append(data.Fields, buildMigrationField(f))
		}
	}

	path := uniqueMigrationPath(root, name)
	content, err := render.File(tmpl, data)
	if err != nil {
		return err
	}
	if err := render.Write(filepath.Join(root, path), content, false); err != nil {
		return err
	}
	fmt.Printf("  create  %s\n", path)
	if table == "" {
		return nil
	}
	if err := syncAfterMigration(root, module, table, add, fieldArgs); err != nil {
		return err
	}
	fmt.Println("\nNext: gor db migrate")
	return nil
}

// buildMigrationField decides how a column can be added to a table that may
// already hold rows: types with a sensible zero get NOT NULL DEFAULT, the
// rest are added nullable with a backfill TODO.
func buildMigrationField(f fields.Field) migrationField {
	mf := migrationField{
		Column:   f.Column(),
		PgType:   f.Mapping().Postgres,
		IsRef:    f.Type == fields.TypeReferences,
		RefTable: f.RefTable(),
	}
	switch f.Type {
	case fields.TypeString, fields.TypeText:
		mf.DefaultSQL = "''"
	case fields.TypeInt, fields.TypeBigint, fields.TypeDecimal:
		mf.DefaultSQL = "0"
	case fields.TypeBool:
		mf.DefaultSQL = "false"
	case fields.TypeTimestamp:
		mf.DefaultSQL = "now()"
	default: // uuid, references: no safe default for existing rows
		mf.Nullable = true
	}
	return mf
}

// uniqueMigrationPath returns migrations/<timestamp>_<base>.sql, bumping the
// timestamp until the goose version is unique within the migrations dir.
func uniqueMigrationPath(root, base string) string {
	entries, _ := os.ReadDir(filepath.Join(root, "migrations"))
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	ts := time.Now().UTC()
	for {
		version := ts.Format("20060102150405")
		taken := false
		for _, name := range names {
			if strings.HasPrefix(name, version+"_") {
				taken = true
				break
			}
		}
		if !taken {
			return filepath.Join("migrations", version+"_"+base+".sql")
		}
		ts = ts.Add(time.Second)
	}
}
