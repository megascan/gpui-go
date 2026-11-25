//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSAppearance wraps an NSAppearance object.
type NSAppearance objc.ID

// ID returns the underlying object.
func (a NSAppearance) ID() objc.ID { return objc.ID(a) }

// NSAppearanceNameAqua returns NSAppearanceNameAqua.
func NSAppearanceNameAqua() objc.ID { return Constant("NSAppearanceNameAqua") }

// NSAppearanceNameDarkAqua returns NSAppearanceNameDarkAqua.
func NSAppearanceNameDarkAqua() objc.ID { return Constant("NSAppearanceNameDarkAqua") }

// NSAppearanceNameVibrantLight returns NSAppearanceNameVibrantLight.
func NSAppearanceNameVibrantLight() objc.ID { return Constant("NSAppearanceNameVibrantLight") }

// NSAppearanceNameVibrantDark returns NSAppearanceNameVibrantDark.
func NSAppearanceNameVibrantDark() objc.ID { return Constant("NSAppearanceNameVibrantDark") }

// NSAppearanceNamed returns the appearance named name or nil.
func NSAppearanceNamed(name objc.ID) NSAppearance {
	return NSAppearance(sendClass[objc.ID]("NSAppearance", "appearanceNamed:", name))
}

// Name returns the appearance name.
func (a NSAppearance) Name() objc.ID { return send[objc.ID](a.ID(), "name") }

// BestMatchFromAppearancesWithNames returns the best matching name or nil.
func (a NSAppearance) BestMatchFromAppearancesWithNames(names []objc.ID) objc.ID {
	array := objc.NSArrayFromIDs(names)
	defer objc.Release(array)
	return send[objc.ID](a.ID(), "bestMatchFromAppearancesWithNames:", array)
}
