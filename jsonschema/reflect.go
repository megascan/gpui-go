package jsonschema

import (
	"encoding/json/jsontext"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Schemaer lets a type supply its own schema, the analogue of schemars' JsonSchema impl.
// A named type that implements it is emitted once under `$defs`.
type Schemaer interface {
	JSONSchema(g *Generator) *Schema
}

// Enumer marks a named string type whose values are the listed variants.
type Enumer interface {
	JSONSchemaEnum() []string
}

// Generator builds schemas for Go types and collects named types under `$defs`.
// A struct field tagged `jsonschema:"deny_unknown_fields"` on a blank `_ struct{}`
// field sets `additionalProperties: false` on that struct.
// A struct field's `doc` tag becomes its description.
// Named struct, Schemaer and Enumer types are emitted under `$defs` and referenced by `$ref`.
type Generator struct {
	defs     map[string]*Schema
	names    map[reflect.Type]string
	used     map[string]reflect.Type
	refs     map[string]bool
	building map[reflect.Type]bool
}

const denyUnknown = "deny_unknown_fields"

var (
	timeType     = reflect.TypeFor[time.Time]()
	rawValueType = reflect.TypeFor[jsontext.Value]()
	schemaerType = reflect.TypeFor[Schemaer]()
	enumerType   = reflect.TypeFor[Enumer]()
)

// NewGenerator returns an empty Generator.
func NewGenerator() *Generator {
	return &Generator{
		defs:     map[string]*Schema{},
		names:    map[reflect.Type]string{},
		used:     map[string]reflect.Type{},
		refs:     map[string]bool{},
		building: map[reflect.Type]bool{},
	}
}

// Reflect returns the root schema for t with `$schema` set and every collected named type under `$defs`.
// A named root type is inlined and titled with its Go name.
func (g *Generator) Reflect(t reflect.Type) *Schema {
	root := g.rootSchema(t)
	root.Schema = Draft
	if len(g.defs) > 0 {
		root.Defs = maps.Clone(g.defs)
	}
	return root
}

// SchemaFor returns the schema for t inline, or a `$ref` when t is a named type.
func (g *Generator) SchemaFor(t reflect.Type) *Schema {
	return g.schemaFor(t)
}

func (g *Generator) rootSchema(t reflect.Type) *Schema {
	if !definable(t) {
		return g.schemaFor(t)
	}
	name := g.nameOf(t)
	g.building[t] = true
	body := g.build(t)
	delete(g.building, t)
	body.Title = t.Name()
	if g.refs[name] {
		inner := *body
		g.defs[name] = &inner
	}
	return body
}

func (g *Generator) schemaFor(t reflect.Type) *Schema {
	switch {
	case t == timeType:
		return &Schema{Type: Types{"string"}, Format: "date-time"}
	case t.Kind() == reflect.Pointer:
		return nullable(g.schemaFor(t.Elem()))
	case definable(t):
		return g.ref(t)
	default:
		return g.build(t)
	}
}

func (g *Generator) ref(t reflect.Type) *Schema {
	name := g.nameOf(t)
	if _, ok := g.defs[name]; !ok && !g.building[t] {
		g.building[t] = true
		g.defs[name] = g.build(t)
		delete(g.building, t)
	}
	g.refs[name] = true
	return &Schema{Ref: "#/$defs/" + name}
}

func (g *Generator) nameOf(t reflect.Type) string {
	if name, ok := g.names[t]; ok {
		return name
	}
	name := g.assignName(t)
	g.names[t] = name
	return name
}

func (g *Generator) build(t reflect.Type) *Schema {
	if t == rawValueType {
		return &Schema{}
	}
	if reflect.PointerTo(t).Implements(schemaerType) {
		return reflect.New(t).Interface().(Schemaer).JSONSchema(g)
	}
	if reflect.PointerTo(t).Implements(enumerType) && t.Kind() == reflect.String {
		variants := reflect.New(t).Interface().(Enumer).JSONSchemaEnum()
		enum := make([]any, len(variants))
		for i, v := range variants {
			enum[i] = v
		}
		return &Schema{Type: Types{"string"}, Enum: enum}
	}
	switch t.Kind() {
	case reflect.Bool:
		return &Schema{Type: Types{"boolean"}}
	case reflect.String:
		return &Schema{Type: Types{"string"}}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return integerSchema(t)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return integerSchema(t)
	case reflect.Float32:
		return &Schema{Type: Types{"number"}, Format: "float"}
	case reflect.Float64:
		return &Schema{Type: Types{"number"}, Format: "double"}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: Types{"string"}, ContentEncoding: "base64"}
		}
		return &Schema{Type: Types{"array"}, Items: g.schemaFor(t.Elem())}
	case reflect.Array:
		n := t.Len()
		return &Schema{Type: Types{"array"}, Items: g.schemaFor(t.Elem()), MinItems: &n, MaxItems: &n}
	case reflect.Map:
		return &Schema{Type: Types{"object"}, AdditionalProperties: g.schemaFor(t.Elem())}
	case reflect.Struct:
		return g.structSchema(t)
	default:
		return &Schema{}
	}
}

func (g *Generator) structSchema(t reflect.Type) *Schema {
	deny := false
	fields := structFields(t, &deny)
	s := &Schema{Type: Types{"object"}}
	if len(fields) > 0 {
		s.Properties = make(map[string]*Schema, len(fields))
	}
	for _, f := range fields {
		fs := g.schemaFor(f.typ)
		if f.doc != "" {
			fs.Description = f.doc
		}
		s.Properties[f.name] = fs
		if f.required {
			s.Required = append(s.Required, f.name)
		}
	}
	if deny {
		s.AdditionalProperties = false
	}
	return s
}

type structField struct {
	name     string
	typ      reflect.Type
	required bool
	doc      string
}

func structFields(t reflect.Type, deny *bool) []structField {
	var direct, embedded []structField
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Name == "_" {
			if f.Tag.Get("jsonschema") == denyUnknown {
				*deny = true
			}
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded = append(embedded, structFields(ft, deny)...)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		required := f.Type.Kind() != reflect.Pointer && !hasOption(opts, "omitempty") && !hasOption(opts, "omitzero")
		direct = append(direct, structField{name: name, typ: f.Type, required: required, doc: f.Tag.Get("doc")})
	}
	seen := make(map[string]bool, len(direct))
	for _, f := range direct {
		seen[f.name] = true
	}
	for _, f := range embedded {
		if !seen[f.name] {
			seen[f.name] = true
			direct = append(direct, f)
		}
	}
	return direct
}

func hasOption(opts, want string) bool {
	for opts != "" {
		var opt string
		opt, opts, _ = strings.Cut(opts, ",")
		if opt == want {
			return true
		}
	}
	return false
}

func integerSchema(t reflect.Type) *Schema {
	bits := t.Bits()
	if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uintptr {
		var maxVal uint64 = ^uint64(0)
		if bits < 64 {
			maxVal = uint64(1)<<bits - 1
		}
		return &Schema{
			Type:    Types{"integer"},
			Format:  "uint" + strconv.Itoa(bits),
			Minimum: uint64(0),
			Maximum: maxVal,
		}
	}
	minVal := int64(-1) << (bits - 1)
	return &Schema{
		Type:    Types{"integer"},
		Format:  "int" + strconv.Itoa(bits),
		Minimum: minVal,
		Maximum: ^minVal,
	}
}

func nullable(s *Schema) *Schema {
	if s.Ref == "" && len(s.Type) > 0 && s.AnyOf == nil {
		s.Type = append(slices.Clone(s.Type), "null")
		if s.Enum != nil {
			s.Enum = append(s.Enum, nil)
		}
		return s
	}
	if s.Ref == "" && len(s.Type) == 0 && s.AnyOf == nil && s.Enum == nil && s.Properties == nil && s.Items == nil && s.AdditionalProperties == nil {
		return s
	}
	return &Schema{AnyOf: []*Schema{s, &Schema{Type: Types{"null"}}}}
}

func definable(t reflect.Type) bool {
	if t.Name() == "" || t.PkgPath() == "" || t == timeType {
		return false
	}
	if t.Kind() == reflect.Struct {
		return true
	}
	return reflect.PointerTo(t).Implements(schemaerType) || reflect.PointerTo(t).Implements(enumerType)
}

func (g *Generator) assignName(t reflect.Type) string {
	base := sanitize(t.Name())
	if _, taken := g.used[base]; !taken {
		g.used[base] = t
		return base
	}
	qualified := sanitize(strings.ReplaceAll(t.PkgPath(), "/", ".") + "." + t.Name())
	name := qualified
	for i := 2; ; i++ {
		other, taken := g.used[name]
		if !taken || other == t {
			break
		}
		name = qualified + "_" + strconv.Itoa(i)
	}
	g.used[name] = t
	return name
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
			return r
		default:
			return '_'
		}
	}, s)
}
