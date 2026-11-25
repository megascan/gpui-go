package gpui

// PlatformKeyboardLayout is a platform-specific keyboard layout.
type PlatformKeyboardLayout interface {
	ID() string
	Name() string
}

// PlatformKeyboardMapper is a platform-specific keyboard mapping.
type PlatformKeyboardMapper interface {
	MapKeyEquivalent(keystroke Keystroke, useKeyEquivalents bool) KeybindingKeystroke
	GetKeyEquivalents() map[rune]rune
}

// DummyKeyboardMapper is a PlatformKeyboardMapper that maps every keystroke to itself.
type DummyKeyboardMapper struct{}

// MapKeyEquivalent returns the keystroke unchanged.
func (DummyKeyboardMapper) MapKeyEquivalent(keystroke Keystroke, useKeyEquivalents bool) KeybindingKeystroke {
	return KeybindingKeystrokeFromKeystroke(keystroke)
}

// GetKeyEquivalents returns nil: there are no key equivalents.
func (DummyKeyboardMapper) GetKeyEquivalents() map[rune]rune {
	return nil
}
