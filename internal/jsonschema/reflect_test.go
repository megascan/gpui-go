package jsonschema

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"
	"time"
)

type schemaInner struct {
	Level int `json:"level"`
}

type schemaPlain struct {
	Name    string   `json:"name" doc:"Display name."`
	Count   int      `json:"count,omitempty"`
	Tags    []string `json:"tags"`
	Note    *string  `json:"note"`
	Skip    int      `json:"-"`
	private int
}

type schemaOuter struct {
	Inner schemaInner  `json:"inner"`
	Ptr   *schemaInner `json:"ptr,omitempty"`
}

type schemaBase struct {
	ID int `json:"id"`
}

type schemaDerived struct {
	schemaBase
	Name string `json:"name"`
}

type schemaNode struct {
	Children []schemaNode `json:"children"`
}

type schemaStrict struct {
	_ struct{} `jsonschema:"deny_unknown_fields"`
	X bool     `json:"x"`
}

type schemaCustom struct {
	Raw string
}

func (schemaCustom) JSONSchema(*Generator) *Schema {
	return &Schema{Type: Types{"string"}, Format: "uri"}
}

type schemaMode string

func (schemaMode) JSONSchemaEnum() []string {
	return []string{"fast", "slow"}
}

type schemaHolder struct {
	C    schemaCustom `json:"c"`
	Mode schemaMode   `json:"mode"`
	When time.Time    `json:"when"`
	Any  any          `json:"any"`
}

func typeOf[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

const (
	rootHead = `{"$schema":"https://json-schema.org/draft/2020-12/schema",`
	int64Min = `"format":"int64"`
)

func TestReflect_Inline(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"bool", typeOf[bool](), `{"type":"boolean"}`},
		{"string", typeOf[string](), `{"type":"string"}`},
		{"int8", typeOf[int8](), `{"type":"integer","format":"int8","minimum":-128,"maximum":127}`},
		{"uint16", typeOf[uint16](), `{"type":"integer","format":"uint16","minimum":0,"maximum":65535}`},
		{"int32", typeOf[int32](), `{"type":"integer","format":"int32","minimum":-2147483648,"maximum":2147483647}`},
		{"uint64", typeOf[uint64](), `{"type":"integer","format":"uint64","minimum":0,"maximum":18446744073709551615}`},
		{"float32", typeOf[float32](), `{"type":"number","format":"float"}`},
		{"float64", typeOf[float64](), `{"type":"number","format":"double"}`},
		{"slice of string", typeOf[[]string](), `{"type":"array","items":{"type":"string"}}`},
		{"fixed array", typeOf[[2]int8](), `{"type":"array","items":{"type":"integer","format":"int8","minimum":-128,"maximum":127},"minItems":2,"maxItems":2}`},
		{"byte slice is base64", typeOf[[]byte](), `{"type":"string","contentEncoding":"base64"}`},
		{"string map", typeOf[map[string]uint8](), `{"type":"object","additionalProperties":{"type":"integer","format":"uint8","minimum":0,"maximum":255}}`},
		{"pointer is nullable", typeOf[*string](), `{"type":["string","null"]}`},
		{"interface accepts all", typeOf[any](), `{}`},
		{"time", typeOf[time.Time](), `{"type":"string","format":"date-time"}`},
		{"raw json value", typeOf[jsontext.Value](), `{}`},
		{"anonymous struct", typeOf[struct {
			A bool `json:"a"`
		}](), `{"type":"object","properties":{"a":{"type":"boolean"}},"required":["a"]}`},
		{"named struct is a ref", typeOf[schemaInner](), `{"$ref":"#/$defs/schemaInner"}`},
		{"enum named string is a ref", typeOf[schemaMode](), `{"$ref":"#/$defs/schemaMode"}`},
		{"schemaer named type is a ref", typeOf[schemaCustom](), `{"$ref":"#/$defs/schemaCustom"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshal(t, NewGenerator().SchemaFor(tt.typ))
			if got != tt.want {
				t.Fatalf("SchemaFor(%v)\n got: %s\nwant: %s", tt.typ, got, tt.want)
			}
		})
	}
}

func TestReflect_Root(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{
			name: "fields tags docs and optionality",
			typ:  typeOf[schemaPlain](),
			want: rootHead + `"title":"schemaPlain","type":"object",` +
				`"properties":{` +
				`"count":{"type":"integer",` + int64Min + `,"minimum":-9223372036854775808,"maximum":9223372036854775807},` +
				`"name":{"description":"Display name.","type":"string"},` +
				`"note":{"type":["string","null"]},` +
				`"tags":{"type":"array","items":{"type":"string"}}},` +
				`"required":["name","tags"]}`,
		},
		{
			name: "named nested struct and optional pointer",
			typ:  typeOf[schemaOuter](),
			want: rootHead + `"$defs":{"schemaInner":{"type":"object","properties":{"level":{"type":"integer",` + int64Min + `,"minimum":-9223372036854775808,"maximum":9223372036854775807}},"required":["level"]}},` +
				`"title":"schemaOuter","type":"object",` +
				`"properties":{` +
				`"inner":{"$ref":"#/$defs/schemaInner"},` +
				`"ptr":{"anyOf":[{"$ref":"#/$defs/schemaInner"},{"type":"null"}]}},` +
				`"required":["inner"]}`,
		},
		{
			name: "embedded struct is flattened",
			typ:  typeOf[schemaDerived](),
			want: rootHead + `"title":"schemaDerived","type":"object",` +
				`"properties":{"id":{"type":"integer",` + int64Min + `,"minimum":-9223372036854775808,"maximum":9223372036854775807},"name":{"type":"string"}},` +
				`"required":["name","id"]}`,
		},
		{
			name: "recursive type refers to itself",
			typ:  typeOf[schemaNode](),
			want: rootHead + `"$defs":{"schemaNode":{"title":"schemaNode","type":"object",` +
				`"properties":{"children":{"type":"array","items":{"$ref":"#/$defs/schemaNode"}}},"required":["children"]}},` +
				`"title":"schemaNode","type":"object",` +
				`"properties":{"children":{"type":"array","items":{"$ref":"#/$defs/schemaNode"}}},"required":["children"]}`,
		},
		{
			name: "deny unknown fields",
			typ:  typeOf[schemaStrict](),
			want: rootHead + `"title":"schemaStrict","type":"object",` +
				`"properties":{"x":{"type":"boolean"}},"required":["x"],"additionalProperties":false}`,
		},
		{
			name: "enum named string",
			typ:  typeOf[schemaMode](),
			want: rootHead + `"title":"schemaMode","type":"string","enum":["fast","slow"]}`,
		},
		{
			name: "field schemaer enum time and interface",
			typ:  typeOf[schemaHolder](),
			want: rootHead + `"$defs":{` +
				`"schemaCustom":{"type":"string","format":"uri"},` +
				`"schemaMode":{"type":"string","enum":["fast","slow"]}},` +
				`"title":"schemaHolder","type":"object",` +
				`"properties":{` +
				`"any":{},` +
				`"c":{"$ref":"#/$defs/schemaCustom"},` +
				`"mode":{"$ref":"#/$defs/schemaMode"},` +
				`"when":{"type":"string","format":"date-time"}},` +
				`"required":["c","mode","when","any"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshal(t, NewGenerator().Reflect(tt.typ))
			if got != tt.want {
				t.Fatalf("Reflect(%v)\n got: %s\nwant: %s", tt.typ, got, tt.want)
			}
		})
	}
}

func TestReflect_OutputIsValidJSON(t *testing.T) {
	for _, typ := range []reflect.Type{typeOf[schemaPlain](), typeOf[schemaOuter](), typeOf[schemaNode](), typeOf[schemaHolder](), typeOf[map[string][]*schemaInner]()} {
		b, err := json.Marshal(NewGenerator().Reflect(typ))
		if err != nil {
			t.Fatalf("Marshal(%v): %v", typ, err)
		}
		if !jsontext.Value(b).IsValid() {
			t.Fatalf("Marshal(%v) produced invalid JSON: %s", typ, b)
		}
	}
}

func TestReflect_SharedGeneratorDedupesDefs(t *testing.T) {
	g := NewGenerator()
	first := marshal(t, g.Reflect(typeOf[schemaOuter]()))
	second := marshal(t, g.Reflect(typeOf[schemaOuter]()))
	if first != second {
		t.Fatalf("second Reflect differs\nfirst:  %s\nsecond: %s", first, second)
	}
	if n := strings.Count(second, `"schemaInner":{`); n != 1 {
		t.Fatalf("schemaInner defined %d times in %s", n, second)
	}
}

func TestReflect_NameCollisionQualifies(t *testing.T) {
	type Dup struct {
		A int `json:"a"`
	}
	local := func() reflect.Type {
		type Dup struct {
			B string `json:"b"`
		}
		return typeOf[Dup]()
	}
	type holder struct {
		First  Dup `json:"first"`
		Second any `json:"second"`
	}
	g := NewGenerator()
	g.SchemaFor(typeOf[Dup]())
	g.SchemaFor(local())
	root := g.Reflect(typeOf[holder]())
	if len(root.Defs) != 2 {
		t.Fatalf("expected 2 defs, got %d: %v", len(root.Defs), root.Defs)
	}
	if _, ok := root.Defs["Dup"]; !ok {
		t.Fatalf("missing plain Dup def: %v", root.Defs)
	}
	var qualified bool
	for name := range root.Defs {
		if name != "Dup" && strings.HasPrefix(name, "github.com.megascan.gpui-go.internal.jsonschema.Dup") {
			qualified = true
		}
	}
	if !qualified {
		t.Fatalf("second Dup not qualified by package path: %v", root.Defs)
	}
}

func marshal(t *testing.T, s *Schema) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(b)
}
