package gpui

import (
	"encoding/binary"
	"encoding/json/jsontext"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"unique"

	"github.com/megascan/gpui-go/jsonschema"
)

// FontFeature is one OpenType feature tag and its value.
type FontFeature struct {
	Tag   string
	Value uint32
}

// FontFeatures are the OpenType features configured for a font. Equal lists share one
// interned key, so FontFeatures is comparable by content; the zero value is the shared empty
// list. Interned keys are released once no FontFeatures refers to them.
type FontFeatures struct {
	key unique.Handle[string]
}

// NewFontFeatures creates font features from a list of tags and values.
func NewFontFeatures(features []FontFeature) FontFeatures {
	if len(features) == 0 {
		return FontFeatures{}
	}
	var key []byte
	for _, feature := range features {
		key = binary.AppendUvarint(key, uint64(len(feature.Tag)))
		key = append(key, feature.Tag...)
		key = binary.LittleEndian.AppendUint32(key, feature.Value)
	}
	return FontFeatures{key: unique.Make(string(key))}
}

// DefaultFontFeatures returns the shared empty feature list.
func DefaultFontFeatures() FontFeatures {
	return FontFeatures{}
}

// FontFeaturesDisableLigatures disables `calt`.
func FontFeaturesDisableLigatures() FontFeatures {
	return NewFontFeatures([]FontFeature{{Tag: "calt", Value: 0}})
}

// TagValueList returns the configured tags and values; only enabled or disabled features are returned.
func (f FontFeatures) TagValueList() []FontFeature {
	features := []FontFeature{}
	if f.key == (unique.Handle[string]{}) {
		return features
	}
	key := f.key.Value()
	for key != "" {
		length, n := binary.Uvarint([]byte(key[:min(len(key), binary.MaxVarintLen64)]))
		key = key[n:]
		tag := key[:length]
		key = key[length:]
		features = append(features, FontFeature{Tag: tag, Value: binary.LittleEndian.Uint32([]byte(key[:4]))})
		key = key[4:]
	}
	return features
}

// IsCaltEnabled reports whether the `calt` feature is enabled; ok is false if it is not present.
func (f FontFeatures) IsCaltEnabled() (enabled bool, ok bool) {
	for _, feature := range f.TagValueList() {
		if feature.Tag == "calt" {
			return feature.Value == 1, true
		}
	}
	return false, false
}

// String formats the features like upstream's Debug output.
func (f FontFeatures) String() string {
	var b strings.Builder
	b.WriteString("FontFeatures")
	list := f.TagValueList()
	if len(list) == 0 {
		return b.String()
	}
	b.WriteString(" { ")
	for i, feature := range list {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s: %d", feature.Tag, feature.Value)
	}
	b.WriteString(" }")
	return b.String()
}

// MarshalJSONTo encodes the features as a map of tag to value.
func (f FontFeatures) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, feature := range f.TagValueList() {
		if err := enc.WriteToken(jsontext.String(feature.Tag)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Uint(uint64(feature.Value))); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom decodes the features from a map of tag to a boolean or an unsigned integer.
// Invalid tags and values are logged and skipped, null values are skipped.
func (f *FontFeatures) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("invalid type: %s, expected a map of font features", tok.Kind())
	}
	var featureList []FontFeature
	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		key := keyTok.String()
		valueTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		var (
			hasValue bool
			isBool   bool
			enable   bool
			number   string
		)
		switch valueTok.Kind() {
		case 'n':
		case 't', 'f':
			hasValue, isBool, enable = true, true, valueTok.Bool()
		case '0':
			hasValue, number = true, valueTok.String()
		default:
			return fmt.Errorf("data did not match any variant of untagged enum FeatureValue")
		}
		if !isValidFeatureTag(key) {
			slog.Error(fmt.Sprintf("Incorrect font feature tag: %s", key))
			continue
		}
		if !hasValue {
			continue
		}
		if isBool {
			if enable {
				featureList = append(featureList, FontFeature{Tag: key, Value: 1})
			} else {
				featureList = append(featureList, FontFeature{Tag: key, Value: 0})
			}
			continue
		}
		value, err := strconv.ParseUint(number, 10, 64)
		if err == nil && value > math.MaxUint32 {
			err = strconv.ErrRange
		}
		if err != nil {
			slog.Error(fmt.Sprintf("Incorrect font feature value %s for feature tag %s", number, key))
			continue
		}
		featureList = append(featureList, FontFeature{Tag: key, Value: uint32(value)})
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	*f = NewFontFeatures(featureList)
	return nil
}

// JSONSchema describes the features as an object of four-character tags.
func (FontFeatures) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"object"}}
}

func isValidFeatureTag(tag string) bool {
	if len(tag) != 4 {
		return false
	}
	for _, c := range tag {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}
