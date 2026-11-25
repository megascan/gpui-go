package gpui

import (
	"cmp"
	"reflect"
	"slices"
)

// KeymapVersion is an opaque identifier of which version of the keymap is currently active.
// The keymap's version is changed whenever bindings are added or removed.
type KeymapVersion struct {
	v uint
}

// Keymap is a collection of key bindings for the user's application.
type Keymap struct {
	bindings                 []KeyBinding
	bindingIndicesByActionID map[reflect.Type][]int
	disabledBindingIndices   []int
	version                  KeymapVersion
}

// BindingIndex is the index of a binding within a keymap.
type BindingIndex struct {
	ix int
}

func disabledBindingMatchesContext(disabledBinding *KeyBinding, binding *KeyBinding) bool {
	switch {
	case disabledBinding.contextPredicate == nil:
		return true
	case binding.contextPredicate == nil:
		return false
	default:
		return disabledBinding.contextPredicate.IsSuperset(binding.contextPredicate)
	}
}

func bindingIsUnbound(disabledBinding *KeyBinding, binding *KeyBinding) bool {
	if !slices.Equal(disabledBinding.keystrokes, binding.keystrokes) {
		return false
	}
	unbind, ok := disabledBinding.Action().(Unbind)
	return ok && string(unbind) == binding.action.Name()
}

// DefaultKeymap returns an empty keymap (upstream's Keymap::default).
func DefaultKeymap() *Keymap {
	return &Keymap{}
}

// NewKeymap creates a new keymap with the given bindings.
func NewKeymap(bindings []KeyBinding) *Keymap {
	this := DefaultKeymap()
	this.AddBindings(bindings)
	return this
}

// Version returns the current version of the keymap.
func (k *Keymap) Version() KeymapVersion {
	return k.version
}

// AddBindings adds more bindings to the keymap.
func (k *Keymap) AddBindings(bindings []KeyBinding) {
	for _, binding := range bindings {
		actionID := reflect.TypeOf(binding.Action())
		if IsNoAction(binding.action) || IsUnbind(binding.action) {
			k.disabledBindingIndices = append(k.disabledBindingIndices, len(k.bindings))
		} else {
			if k.bindingIndicesByActionID == nil {
				k.bindingIndicesByActionID = make(map[reflect.Type][]int)
			}
			k.bindingIndicesByActionID[actionID] = append(k.bindingIndicesByActionID[actionID], len(k.bindings))
		}
		k.bindings = append(k.bindings, binding)
	}

	k.version.v++
}

// Clear resets this keymap to its initial state.
func (k *Keymap) Clear() {
	clear(k.bindings)
	k.bindings = k.bindings[:0]
	clear(k.bindingIndicesByActionID)
	k.disabledBindingIndices = k.disabledBindingIndices[:0]
	k.version.v++
}

// Bindings returns all bindings, in the order they were added. The slice is the keymap's own
// storage and must not be modified.
func (k *Keymap) Bindings() []KeyBinding {
	return k.bindings
}

// BindingsForAction returns all bindings for the given action, in the order they were added.
// For display, the last binding should take precedence.
func (k *Keymap) BindingsForAction(action Action) []*KeyBinding {
	actionID := reflect.TypeOf(action)
	var result []*KeyBinding
outer:
	for _, ix := range k.bindingIndicesByActionID[actionID] {
		binding := &k.bindings[ix]
		if !binding.Action().PartialEq(action) {
			continue
		}

		for _, disabledIx := range k.disabledBindingIndices {
			if disabledIx > ix {
				disabledBinding := &k.bindings[disabledIx]
				if !slices.Equal(disabledBinding.keystrokes, binding.keystrokes) {
					continue
				}

				if IsNoAction(disabledBinding.action) {
					if disabledBindingMatchesContext(disabledBinding, binding) {
						continue outer
					}
				} else if IsUnbind(disabledBinding.action) &&
					disabledBindingMatchesContext(disabledBinding, binding) &&
					bindingIsUnbound(disabledBinding, binding) {
					continue outer
				}
			}
		}

		result = append(result, binding)
	}
	return result
}

// AllBindingsForInput returns all bindings that might match the input without checking
// context, in precedence order (reverse of the order they were added to the keymap).
func (k *Keymap) AllBindingsForInput(input []Keystroke) []KeyBinding {
	var result []KeyBinding
	for i := len(k.bindings) - 1; i >= 0; i-- {
		binding := &k.bindings[i]
		if pending, ok := binding.MatchKeystrokes(input); ok && !pending {
			result = append(result, binding.Clone())
		}
	}
	return result
}

type indexedBinding struct {
	ix      int
	binding *KeyBinding
}

func (k *Keymap) reversedBindings() []indexedBinding {
	result := make([]indexedBinding, 0, len(k.bindings))
	for i := len(k.bindings) - 1; i >= 0; i-- {
		result = append(result, indexedBinding{ix: i, binding: &k.bindings[i]})
	}
	return result
}

// BindingsForInput returns the bindings that match the given input, and whether more
// bindings might match if the input was longer. Bindings are returned in precedence order
// (higher precedence first, reverse of the order they were added to the keymap).
//
// Precedence is defined by the depth in the tree (matches on the Editor take precedence over
// matches on the Pane, then the Workspace, etc.). Bindings with no context are treated as the
// same as the deepest context. At equal depth, bindings added later take precedence.
//
// A binding disabled with `"x": null` is not returned. Disabled bindings follow the same
// precedence rules, and only suppress bindings from sources with equal or weaker precedence.
func (k *Keymap) BindingsForInput[K any, PK interface {
	*K
	AsKeystroke
}](input []K, contextStack []KeyContext) ([]KeyBinding, bool) {
	bindings, pending := resolveBindingsForInput[K, PK](k, input, contextStack, k.reversedBindings())
	result := make([]KeyBinding, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, binding.binding.Clone())
	}
	return result, pending
}

type matchedBinding struct {
	depth   int
	ix      BindingIndex
	binding *KeyBinding
}

type resolvedBinding struct {
	ix      BindingIndex
	binding *KeyBinding
}

func resolveBindingsForInput[K any, PK interface {
	*K
	AsKeystroke
}](k *Keymap, input []K, contextStack []KeyContext, keymapBindings []indexedBinding) ([]resolvedBinding, bool) {
	var matchedBindings []matchedBinding
	var pendingBindings []resolvedBinding

	for _, entry := range keymapBindings {
		binding := entry.binding
		pending, ok := binding.MatchKeystrokes[K, PK](input)
		if !ok {
			continue
		}
		depth, ok := k.bindingEnabled(binding, contextStack)
		if !ok {
			continue
		}

		if !pending {
			matchedBindings = append(matchedBindings, matchedBinding{depth: depth, ix: BindingIndex{entry.ix}, binding: binding})
		} else {
			pendingBindings = append(pendingBindings, resolvedBinding{ix: BindingIndex{entry.ix}, binding: binding})
		}
	}

	slices.SortStableFunc(matchedBindings, func(a, b matchedBinding) int {
		return cmp.Or(cmp.Compare(b.depth, a.depth), cmp.Compare(b.ix.ix, a.ix.ix))
	})

	var bindings []resolvedBinding
	firstBindingIndex := None[BindingIndex]()
	var unboundBindings []*KeyBinding
	noActionMeta := None[uint32]()

	for _, matched := range matchedBindings {
		binding := matched.binding
		meta := uint32(binding.meta.UnwrapOr(0))
		if IsNoAction(binding.action) {
			if existing, ok := noActionMeta.Get(); ok {
				noActionMeta = Some(min(existing, meta))
			} else {
				noActionMeta = Some(meta)
			}
			continue
		}

		if existing, ok := noActionMeta.Get(); ok && meta >= existing {
			continue
		}

		if IsUnbind(binding.action) {
			unboundBindings = append(unboundBindings, binding)
			continue
		}

		if slices.ContainsFunc(unboundBindings, func(disabledBinding *KeyBinding) bool {
			return bindingIsUnbound(disabledBinding, binding)
		}) {
			continue
		}

		bindings = append(bindings, resolvedBinding{ix: matched.ix, binding: binding})
		if firstBindingIndex.IsNone() {
			firstBindingIndex = Some(matched.ix)
		}
	}

	var pending [][]KeybindingKeystroke
	indexOf := func(keystrokes []KeybindingKeystroke) int {
		return slices.IndexFunc(pending, func(p []KeybindingKeystroke) bool {
			return slices.Equal(p, keystrokes)
		})
	}
	for i := len(pendingBindings) - 1; i >= 0; i-- {
		entry := pendingBindings[i]
		if bindingIx, ok := firstBindingIndex.Get(); ok && bindingIx.ix > entry.ix.ix {
			continue
		}
		if IsNoAction(entry.binding.action) || IsUnbind(entry.binding.action) {
			if at := indexOf(entry.binding.keystrokes); at >= 0 {
				pending = slices.Delete(pending, at, at+1)
			}
			continue
		}
		if indexOf(entry.binding.keystrokes) < 0 {
			pending = append(pending, entry.binding.keystrokes)
		}
	}

	return bindings, len(pending) != 0
}

func (k *Keymap) bindingEnabled(binding *KeyBinding, contexts []KeyContext) (int, bool) {
	if binding.contextPredicate != nil {
		return binding.contextPredicate.DepthOf(contexts)
	}
	return len(contexts), true
}

// PossibleNextBindingsForInput finds the bindings that can follow the current input sequence,
// in precedence order. It only includes bindings that dispatch would run for their full
// keystrokes, so bindings disabled with `null` or Unbind are left out.
func (k *Keymap) PossibleNextBindingsForInput(input []Keystroke, contextStack []KeyContext) []KeyBinding {
	var bindingsExtendingInput []indexedBinding
	for _, entry := range k.reversedBindings() {
		if pending, ok := entry.binding.MatchKeystrokes(input); ok && pending {
			bindingsExtendingInput = append(bindingsExtendingInput, entry)
		}
	}

	var fullInput []Keystroke
	var candidates []matchedBinding
	for _, entry := range bindingsExtendingInput {
		binding := entry.binding
		if IsNoAction(binding.action) || IsUnbind(binding.action) {
			continue
		}
		depth, ok := k.bindingEnabled(binding, contextStack)
		if !ok {
			continue
		}
		fullInput = append(fullInput[:0], input...)
		for i := len(input); i < len(binding.keystrokes); i++ {
			fullInput = append(fullInput, *binding.keystrokes[i].AsKeystroke())
		}
		dispatched, _ := resolveBindingsForInput[Keystroke](k, fullInput, contextStack, bindingsExtendingInput)
		if slices.ContainsFunc(dispatched, func(d resolvedBinding) bool { return d.ix == BindingIndex{entry.ix} }) {
			candidates = append(candidates, matchedBinding{depth: depth, ix: BindingIndex{entry.ix}, binding: binding})
		}
	}
	slices.SortFunc(candidates, func(a, b matchedBinding) int {
		return cmp.Or(cmp.Compare(b.depth, a.depth), cmp.Compare(b.ix.ix, a.ix.ix))
	})

	result := make([]KeyBinding, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.binding.Clone())
	}
	return result
}
