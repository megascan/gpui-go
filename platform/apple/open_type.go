//go:build darwin

package apple

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/ct"
)

// ApplyFeaturesAndFallbacks replaces *font with a copy carrying the OpenType feature settings and,
// when the list is non-empty, a cascade list of the fallbacks followed by the system fallbacks.
// The replaced font is released.
func ApplyFeaturesAndFallbacks(font **cf.Owned[ct.Font], features gpui.FontFeatures, fallbacks gpui.Option[gpui.FontFallbacks]) error {
	keys := []cf.TypeRef{cf.TypeRef(ct.FontFeatureSettingsAttribute)}
	featureArray := generateFeatureArray(features)
	values := []*cf.Owned[ct.MutableArray]{featureArray}
	if fb, ok := fallbacks.Get(); ok && len(fb.FallbackList()) > 0 {
		keys = append(keys, cf.TypeRef(ct.FontCascadeListAttribute))
		values = append(values, generateFallbackArray(fb, (*font).Ref()))
	}
	refs := make([]cf.TypeRef, len(values))
	for i, v := range values {
		refs[i] = v.TypeRef()
	}
	attrs := cf.DictionaryCreate(keys, refs)
	for _, v := range values {
		v.Release()
	}
	newDescriptor := ct.FontDescriptorCreateWithAttributes(attrs.Ref())
	attrs.Release()
	defer newDescriptor.Release()
	newFont := (*font).Ref().CopyWithAttributes(0.0, newDescriptor.Ref())
	(*font).Release()
	*font = newFont
	return nil
}

func generateFeatureArray(features gpui.FontFeatures) *cf.Owned[ct.MutableArray] {
	featureArray := ct.ArrayCreateMutable()
	for _, f := range features.TagValueList() {
		tag := cf.StringCreate(f.Tag)
		value := ct.NumberCreateInt32(int32(f.Value))
		dict := cf.DictionaryCreate(
			[]cf.TypeRef{cf.TypeRef(ct.FontOpenTypeFeatureTag), cf.TypeRef(ct.FontOpenTypeFeatureValue)},
			[]cf.TypeRef{tag.TypeRef(), value.TypeRef()},
		)
		tag.Release()
		value.Release()
		featureArray.Ref().Append(dict.TypeRef())
		dict.Release()
	}
	return featureArray
}

func generateFallbackArray(fallbacks gpui.FontFallbacks, font ct.Font) *cf.Owned[ct.MutableArray] {
	symbolicTraits := font.SymbolicTraits()
	allTraits := fontTraits{dict: font.CopyTraits()}
	defer allTraits.release()

	fallbackArray := ct.ArrayCreateMutable()
	for _, userFallback := range fallbacks.FallbackList() {
		name := cf.StringCreate(userFallback)
		weightValue := ct.NumberCreateFloat64(allTraits.normalizedWeight())
		slant := 0.0
		if symbolicTraits&ct.FontItalicTrait != 0 {
			slant = 1.0
		}
		slantValue := ct.NumberCreateFloat64(slant)
		traits := cf.DictionaryCreate(
			[]cf.TypeRef{cf.TypeRef(ct.FontWeightTrait), cf.TypeRef(ct.FontSlantTrait)},
			[]cf.TypeRef{weightValue.TypeRef(), slantValue.TypeRef()},
		)
		weightValue.Release()
		slantValue.Release()

		attrs := cf.DictionaryCreate(
			[]cf.TypeRef{cf.TypeRef(ct.FontFamilyNameAttribute), cf.TypeRef(ct.FontTraitsAttribute)},
			[]cf.TypeRef{name.TypeRef(), traits.TypeRef()},
		)
		traits.Release()
		name.Release()

		fallbackDesc := ct.FontDescriptorCreateWithAttributes(attrs.Ref())
		attrs.Release()

		fallbackArray.Ref().Append(fallbackDesc.TypeRef())
		fallbackDesc.Release()
	}

	appendSystemFallbacks(fallbackArray.Ref(), font)
	return fallbackArray
}

func appendSystemFallbacks(fallbackArray ct.MutableArray, font ct.Font) {
	preferredLanguages := ct.LocaleCopyPreferredLanguages()
	defer preferredLanguages.Release()

	defaultFallbacks := font.CascadeListForLanguages(preferredLanguages.Ref())
	if defaultFallbacks == nil {
		return
	}
	defer defaultFallbacks.Release()
	arr := defaultFallbacks.Ref()
	for i := range arr.Count() {
		desc := ct.FontDescriptor(arr.At(i))
		if _, ok := desc.FontPath(); ok {
			fallbackArray.Append(cf.TypeRef(desc))
		}
	}
}
