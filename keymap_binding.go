package gpui

import (
	"fmt"
	"slices"
	"strings"
)

// KeyBinding is a keybinding and its associated metadata, from the keymap.
type KeyBinding struct {
	action           Action
	keystrokes       []KeybindingKeystroke
	contextPredicate KeyBindingContextPredicate
	meta             Option[KeyBindingMetaIndex]
	actionInput      Option[SharedString]
}

// Clone returns a copy of the binding with a cloned action.
func (b KeyBinding) Clone() KeyBinding {
	return KeyBinding{
		action:           b.action.BoxedClone(),
		keystrokes:       slices.Clone(b.keystrokes),
		contextPredicate: b.contextPredicate,
		meta:             b.meta,
		actionInput:      b.actionInput,
	}
}

// NewKeyBinding constructs a static keybinding with explicit modifiers. It panics on a parse
// error. Use KeyBindingLoad for dynamic input or to resolve `secondary-` using the host OS.
func NewKeyBinding(keystrokes string, action Action, context Option[string]) KeyBinding {
	var contextPredicate KeyBindingContextPredicate
	if context, ok := context.Get(); ok {
		predicate, err := KeyBindingContextPredicateParse(context)
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		contextPredicate = predicate
	}
	binding, err := keyBindingLoadInternal(keystrokes, action, contextPredicate, false, None[SharedString](), DummyKeyboardMapper{}, KeystrokeParse)
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	return binding
}

// KeyBindingLoad loads a keybinding from the given raw data. contextPredicate may be nil.
func KeyBindingLoad(keystrokes string, action Action, contextPredicate KeyBindingContextPredicate, useKeyEquivalents bool, actionInput Option[SharedString], cx *App) (KeyBinding, error) {
	os := cx.PlatformInfo().Os
	return keyBindingLoadInternal(keystrokes, action, contextPredicate, useKeyEquivalents, actionInput, cx.KeyboardMapper(), func(source string) (Keystroke, error) {
		return KeystrokeParseWithOs(source, os)
	})
}

func keyBindingLoadInternal(keystrokes string, action Action, contextPredicate KeyBindingContextPredicate, useKeyEquivalents bool, actionInput Option[SharedString], keyboardMapper PlatformKeyboardMapper, parseKeystroke func(string) (Keystroke, error)) (KeyBinding, error) {
	fields := strings.Fields(keystrokes)
	parsed := make([]KeybindingKeystroke, 0, len(fields))
	for _, source := range fields {
		keystroke, err := parseKeystroke(source)
		if err != nil {
			return KeyBinding{}, err
		}
		parsed = append(parsed, NewKeybindingKeystrokeWithMapper(keystroke, useKeyEquivalents, keyboardMapper))
	}

	return KeyBinding{
		keystrokes:       parsed,
		action:           action,
		contextPredicate: contextPredicate,
		meta:             None[KeyBindingMetaIndex](),
		actionInput:      actionInput,
	}, nil
}

// WithMeta returns the binding with its metadata set.
func (b KeyBinding) WithMeta(meta KeyBindingMetaIndex) KeyBinding {
	b.meta = Some(meta)
	return b
}

// SetMeta sets the metadata for this binding.
func (b *KeyBinding) SetMeta(meta KeyBindingMetaIndex) {
	b.meta = Some(meta)
}

// MatchKeystrokes checks if the given keystrokes match this binding. ok is false when they
// do not match; pending reports whether the binding has more keystrokes than typed.
func (b *KeyBinding) MatchKeystrokes[K any, PK interface {
	*K
	AsKeystroke
}](typed []K) (pending bool, ok bool) {
	if len(b.keystrokes) < len(typed) {
		return false, false
	}

	for i := range typed {
		if !PK(&typed[i]).AsKeystroke().ShouldMatch(b.keystrokes[i]) {
			return false, false
		}
	}

	return len(b.keystrokes) > len(typed), true
}

// Keystrokes returns the keystrokes associated with this binding.
func (b *KeyBinding) Keystrokes() []KeybindingKeystroke {
	return b.keystrokes
}

// Action returns the action associated with this binding.
func (b *KeyBinding) Action() Action {
	return b.action
}

// Predicate returns the predicate used to match this binding, or nil.
func (b *KeyBinding) Predicate() KeyBindingContextPredicate {
	return b.contextPredicate
}

// Meta returns the metadata for this binding.
func (b *KeyBinding) Meta() (KeyBindingMetaIndex, bool) {
	return b.meta.Get()
}

// ActionInput returns the JSON input string used when building the binding's action.
func (b *KeyBinding) ActionInput() (SharedString, bool) {
	return b.actionInput.Get()
}

// String formats the binding as upstream's Debug implementation does.
func (b KeyBinding) String() string {
	var predicate string
	if b.contextPredicate == nil {
		predicate = "None"
	} else {
		predicate = "Some(" + b.contextPredicate.String() + ")"
	}
	return fmt.Sprintf("KeyBinding { keystrokes: %v, context_predicate: %s, action: %q }", b.keystrokes, predicate, b.action.Name())
}

// KeyBindingMetaIndex is a unique identifier for retrieval of metadata associated with a key
// binding, such as the source of the binding.
type KeyBindingMetaIndex uint32
