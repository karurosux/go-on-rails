package fields

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	got, err := Parse([]string{"title:string", "body:text", "published:bool", "author:references"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Field{
		{Name: "title", Type: TypeString},
		{Name: "body", Type: TypeText},
		{Name: "published", Type: TypeBool},
		{Name: "author", Type: TypeReferences},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		msg  string // substring the error must contain
	}{
		{"empty", nil, "at least one field"},
		{"no colon", []string{"title"}, "expected name:type"},
		{"empty name", []string{":string"}, "expected name:type"},
		{"empty type", []string{"title:"}, "expected name:type"},
		{"unknown type", []string{"title:varchar"}, "unknown field type"},
		{"bad name", []string{"Title:string"}, "lower_snake_case"},
		{"bad name dash", []string{"my-field:string"}, "lower_snake_case"},
		{"reserved id", []string{"id:uuid"}, "automatically"},
		{"reserved created_at", []string{"created_at:timestamp"}, "automatically"},
		{"duplicate", []string{"title:string", "title:text"}, "duplicate"},
		{"duplicate via references", []string{"author_id:uuid", "author:references"}, "duplicate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.args)
			if err == nil {
				t.Fatalf("expected error for %v", tc.args)
			}
			if !strings.Contains(err.Error(), tc.msg) {
				t.Errorf("error %q does not contain %q", err, tc.msg)
			}
		})
	}
}

func TestColumn(t *testing.T) {
	if got := (Field{Name: "title", Type: TypeString}).Column(); got != "title" {
		t.Errorf("Column() = %q, want title", got)
	}
	if got := (Field{Name: "author", Type: TypeReferences}).Column(); got != "author_id" {
		t.Errorf("Column() = %q, want author_id", got)
	}
}

func TestRefTable(t *testing.T) {
	f := Field{Name: "category", Type: TypeReferences}
	if got := f.RefTable(); got != "categories" {
		t.Errorf("RefTable() = %q, want categories", got)
	}
}
