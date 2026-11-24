package gpui

import (
	"encoding/binary"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"sync"

	"github.com/megascan/gpui-go/jsonschema"
)

// FontFallbacks are the fallback font family names configured for a font. Equal lists share
// one interned backing list, so FontFallbacks is comparable by content.
type FontFallbacks struct {
	list *[]string
}

var fontFallbacksIntern sync.Map

// FallbackList returns the fallback font family names.
func (f FontFallbacks) FallbackList() []string {
	if f.list == nil {
		return []string{}
	}
	return *f.list
}

// FontFallbacksFromFonts creates font fallbacks from a list of family names.
func FontFallbacksFromFonts(fonts []string) FontFallbacks {
	if len(fonts) == 0 {
		return FontFallbacks{}
	}
	var key strings.Builder
	for _, name := range fonts {
		key.Write(binary.AppendUvarint(nil, uint64(len(name))))
		key.WriteString(name)
	}
	list := append([]string(nil), fonts...)
	actual, _ := fontFallbacksIntern.LoadOrStore(key.String(), &list)
	return FontFallbacks{list: actual.(*[]string)}
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
