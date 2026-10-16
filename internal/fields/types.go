package fields

import (
	"fmt"
	"sort"
	"strings"
)

// Type is a gor field type as written on the command line (title:string).
type Type string

const (
	TypeString     Type = "string"
	TypeText       Type = "text"
	TypeInt        Type = "int"
	TypeBigint     Type = "bigint"
	TypeBool       Type = "bool"
	TypeDecimal    Type = "decimal"
	TypeTimestamp  Type = "timestamp"
	TypeUUID       Type = "uuid"
	TypeReferences Type = "references"
)

// Input identifies which form widget the frontend templates render.
type Input string

const (
	InputText     Input = "text"
	InputTextarea Input = "textarea"
	InputNumber   Input = "number"
	InputCheckbox Input = "checkbox"
	InputDatetime Input = "datetime-local"
)

// Mapping is the single source of truth for how a gor type maps onto
// every layer of the stack.
type Mapping struct {
	Postgres string // column type in the migration
	Go       string // Go type as emitted by sqlc (non-null column)
	GoPtr    string // pointer Go type, used for PATCH inputs (sqlc narg params)
	TS       string // TypeScript type in the API client
	Zod      string // zod schema expression for the form
	Input    Input  // form widget
	Required bool   // whether the create input gets validate:"required"
}

var mappings = map[Type]Mapping{
	TypeString: {
		Postgres: "text",
		Go:       "string",
		GoPtr:    "*string",
		TS:       "string",
		Zod:      `z.string().min(1, "Required")`,
		Input:    InputText,
		Required: true,
	},
	TypeText: {
		Postgres: "text",
		Go:       "string",
		GoPtr:    "*string",
		TS:       "string",
		Zod:      `z.string()`,
		Input:    InputTextarea,
		Required: false,
	},
	TypeInt: {
		Postgres: "integer",
		Go:       "int32",
		GoPtr:    "*int32",
		TS:       "number",
		Zod:      `z.number({ invalid_type_error: "Must be a number" }).int()`,
		Input:    InputNumber,
		Required: false,
	},
	TypeBigint: {
		Postgres: "bigint",
		Go:       "int64",
		GoPtr:    "*int64",
		TS:       "number",
		Zod:      `z.number({ invalid_type_error: "Must be a number" }).int()`,
		Input:    InputNumber,
		Required: false,
	},
	TypeBool: {
		Postgres: "boolean",
		Go:       "bool",
		GoPtr:    "*bool",
		TS:       "boolean",
		Zod:      `z.boolean()`,
		Input:    InputCheckbox,
		Required: false,
	},
	TypeDecimal: {
		Postgres: "numeric",
		Go:       "string", // sqlc override: numeric -> string, lossless
		GoPtr:    "*string",
		TS:       "string",
		Zod:      `z.string().regex(/^-?\d+(\.\d+)?$/, "Must be a number")`,
		Input:    InputText,
		Required: true,
	},
	TypeTimestamp: {
		Postgres: "timestamptz",
		Go:       "time.Time",
		GoPtr:    "*time.Time",
		TS:       "string",
		Zod:      `z.string().min(1, "Required")`,
		Input:    InputDatetime,
		Required: true,
	},
	TypeUUID: {
		Postgres: "uuid",
		Go:       "uuid.UUID",
		GoPtr:    "*uuid.UUID",
		TS:       "string",
		Zod:      `z.string().uuid("Must be a UUID")`,
		Input:    InputText,
		Required: true,
	},
	TypeReferences: {
		Postgres: "uuid",
		Go:       "uuid.UUID",
		GoPtr:    "*uuid.UUID",
		TS:       "string",
		Zod:      `z.string().uuid("Must be a UUID")`,
		Input:    InputText,
		Required: true,
	},
}

// MappingFor returns the mapping for t, or an error naming the valid types.
func MappingFor(t Type) (Mapping, error) {
	m, ok := mappings[t]
	if !ok {
		return Mapping{}, fmt.Errorf("unknown field type %q (valid types: %s)", t, strings.Join(TypeNames(), ", "))
	}
	return m, nil
}

// TypeNames returns all valid type names, sorted.
func TypeNames() []string {
	names := make([]string, 0, len(mappings))
	for t := range mappings {
		names = append(names, string(t))
	}
	sort.Strings(names)
	return names
}
