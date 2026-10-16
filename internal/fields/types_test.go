package fields

import (
	"strings"
	"testing"
)

func TestMappingTableComplete(t *testing.T) {
	all := []Type{
		TypeString, TypeText, TypeInt, TypeBigint, TypeBool,
		TypeDecimal, TypeTimestamp, TypeUUID, TypeReferences, TypeFile,
	}
	if len(mappings) != len(all) {
		t.Fatalf("mappings has %d entries, want %d", len(mappings), len(all))
	}
	for _, typ := range all {
		m, err := MappingFor(typ)
		if err != nil {
			t.Fatalf("MappingFor(%s): %v", typ, err)
		}
		if m.Postgres == "" || m.Go == "" || m.GoPtr == "" || m.TS == "" || m.Zod == "" || m.Input == "" {
			t.Errorf("mapping for %s has empty members: %+v", typ, m)
		}
		if !strings.HasPrefix(m.GoPtr, "*") {
			t.Errorf("mapping for %s: GoPtr %q is not a pointer type", typ, m.GoPtr)
		}
		if m.GoPtr != "*"+m.Go {
			t.Errorf("mapping for %s: GoPtr %q does not match Go type %q", typ, m.GoPtr, m.Go)
		}
	}
}

func TestMappingSpotChecks(t *testing.T) {
	cases := []struct {
		typ      Type
		postgres string
		goType   string
		ts       string
		input    Input
	}{
		{TypeString, "text", "string", "string", InputText},
		{TypeText, "text", "string", "string", InputTextarea},
		{TypeInt, "integer", "int32", "number", InputNumber},
		{TypeBigint, "bigint", "int64", "number", InputNumber},
		{TypeBool, "boolean", "bool", "boolean", InputCheckbox},
		{TypeDecimal, "numeric", "string", "string", InputText},
		{TypeTimestamp, "timestamptz", "time.Time", "string", InputDatetime},
		{TypeUUID, "uuid", "uuid.UUID", "string", InputText},
		{TypeReferences, "uuid", "uuid.UUID", "string", InputText},
		{TypeFile, "text", "string", "string", InputFile},
	}
	for _, tc := range cases {
		m, err := MappingFor(tc.typ)
		if err != nil {
			t.Fatalf("MappingFor(%s): %v", tc.typ, err)
		}
		if m.Postgres != tc.postgres {
			t.Errorf("%s: Postgres = %q, want %q", tc.typ, m.Postgres, tc.postgres)
		}
		if m.Go != tc.goType {
			t.Errorf("%s: Go = %q, want %q", tc.typ, m.Go, tc.goType)
		}
		if m.TS != tc.ts {
			t.Errorf("%s: TS = %q, want %q", tc.typ, m.TS, tc.ts)
		}
		if m.Input != tc.input {
			t.Errorf("%s: Input = %q, want %q", tc.typ, m.Input, tc.input)
		}
	}
}

func TestMappingForUnknown(t *testing.T) {
	_, err := MappingFor("varchar")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	// The error should list every valid type so the user can self-correct.
	for _, name := range TypeNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention valid type %q", err, name)
		}
	}
}
