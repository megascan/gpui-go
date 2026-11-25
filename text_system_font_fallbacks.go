package gpui

import (
	"encoding/binary"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"unique"

	"github.com/megascan/gpui-go/jsonschema"
)

// FontFallbacks are the fallback font family names configured for a font. Equal lists share
// one interned key, so FontFallbacks is comparable by content. Interned keys are released once
// no FontFallbacks refers to them.
type FontFallbacks struct {
	key unique.Handle[string]
}

// FallbackList returns the fallback font family names.
func (f FontFallbacks) FallbackList() []string {
	fonts := []string{}
	if f.key == (unique.Handle[string]{}) {
		return fonts
	}
	key := f.key.Value()
	for key != "" {
		length, n := binary.Uvarint([]byte(key[:min(len(key), binary.MaxVarintLen64)]))
		key = key[n:]
		fonts = append(fonts, key[:length])
		key = key[length:]
	}
	return fonts
}

// FontFallbacksFromFonts creates font fallbacks from a list of family names.
func FontFallbacksFromFonts(fonts []string) FontFallbacks {
	if len(fonts) == 0 {
		return FontFallbacks{}
	}
	var key []byte
	for _, name := range fonts {
		key = binary.AppendUvarint(key, uint64(len(name)))
		key = append(key, name...)
	}
	return FontFallbacks{key: unique.Make(string(key))}
}

// MarshalJSONTo encodes the fallbacks as an array of family names.
func (f FontFallbacks) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, f.FallbackList())
}

// UnmarshalJSONFrom decodes the fallbacks from an array of family names.
func (f *FontFallbacks) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var fonts []string
	if err := json.UnmarshalDecode(dec, &fonts); err != nil {
		return err
	}
	*f = FontFallbacksFromFonts(fonts)
	return nil
}

// JSONSchema describes the fallbacks as an array of strings.
func (FontFallbacks) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:  jsonschema.Types{"array"},
		Items: &jsonschema.Schema{Type: jsonschema.Types{"string"}},
	}
}
