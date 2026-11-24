package jsonschema

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// Draft is the `$schema` URI emitted on root schemas, matching schemars 1.x.
const Draft = "https://json-schema.org/draft/2020-12/schema"

// Schema is a JSON Schema document node. Zero-valued fields are omitted; a
// zero Schema marshals as `{}`, the schema that accepts every value.
type Schema struct {
	Schema               string             `json:"$schema,omitzero"`
	Ref                  string             `json:"$ref,omitzero"`
	Defs                 map[string]*Schema `json:"$defs,omitzero"`
	Title                string             `json:"title,omitzero"`
	Description          string             `json:"description,omitzero"`
	Type                 Types              `json:"type,omitzero"`
	Format               string             `json:"format,omitzero"`
	ContentEncoding      string             `json:"contentEncoding,omitzero"`
	Properties           map[string]*Schema `json:"properties,omitzero"`
	Required             []string           `json:"required,omitzero"`
	AdditionalProperties any                `json:"additionalProperties,omitzero"`
	Items                *Schema            `json:"items,omitzero"`
	MinItems             *int               `json:"minItems,omitzero"`
	MaxItems             *int               `json:"maxItems,omitzero"`
	Enum                 []any              `json:"enum,omitzero"`
	Minimum              any                `json:"minimum,omitzero"`
	Maximum              any                `json:"maximum,omitzero"`
	AnyOf                []*Schema          `json:"anyOf,omitzero"`
}

// Types is the `type` keyword: one name marshals as a string, several as an array.
type Types []string

// MarshalJSONTo implements json.MarshalerTo.
func (t Types) MarshalJSONTo(enc *jsontext.Encoder) error {
	if len(t) == 1 {
		return enc.WriteToken(jsontext.String(t[0]))
	}
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, name := range t {
		if err := enc.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

type plainSchema Schema

// MarshalJSON encodes the schema with encoding/json/v2 and sorted object keys.
func (s *Schema) MarshalJSON() ([]byte, error) {
	return json.Marshal((*plainSchema)(s), json.Deterministic(true))
}
