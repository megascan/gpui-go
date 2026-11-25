package gpui

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// KeyContext is a set of identifiers and key-value pairs representing the current context
// for the keymap, used to resolve whether an action should be dispatched at a point in the
// element tree.
type KeyContext struct {
	entries []ContextEntry
}

// ContextEntry is an entry in a KeyContext.
type ContextEntry struct {
	// Key is the key, or the name if there is no value.
	Key SharedString
	// Value is the value.
	Value Option[SharedString]
}

// NewKeyContextWithDefaults initializes a key context with the supplied host operating system.
func NewKeyContextWithDefaults(os OperatingSystem) KeyContext {
	var context KeyContext
	var name SharedString
	switch os {
	case OperatingSystemMac:
		name = "macos"
	case OperatingSystemLinux, OperatingSystemFreeBsd:
		name = "linux"
	case OperatingSystemWindows:
		name = "windows"
	case OperatingSystemIos:
		name = "ios"
	case OperatingSystemAndroid:
		name = "android"
	default:
		name = "unknown"
	}
	context.Set("os", name)
	return context
}

// Clone returns a copy of the context that shares no storage with k.
func (k KeyContext) Clone() KeyContext {
	return KeyContext{entries: slices.Clone(k.entries)}
}

// Primary returns the primary context entry (usually the name of the component), or nil.
func (k *KeyContext) Primary() *ContextEntry {
	for i := range k.entries {
		if k.entries[i].Value.IsNone() {
			return &k.entries[i]
		}
	}
	return nil
}

// Secondary returns everything except the primary context entry.
func (k *KeyContext) Secondary() iter.Seq[*ContextEntry] {
	primary := k.Primary()
	return func(yield func(*ContextEntry) bool) {
		for i := range k.entries {
			p := &k.entries[i]
			if primary != nil && *p == *primary {
				continue
			}
			if !yield(p) {
				return
			}
		}
	}
}

// KeyContextParse parses a key context from a string: a single identifier such as
// `StatusBar`, a key value pair such as `mode = visible`, or several of these separated by
// whitespace, such as `StatusBar mode = visible`.
func KeyContextParse(source string) (KeyContext, error) {
	var context KeyContext
	source = skipWhitespace(source)
	if err := keyContextParseExpr(source, &context); err != nil {
		return KeyContext{}, err
	}
	return context, nil
}

func keyContextParseExpr(source string, context *KeyContext) error {
	for source != "" {
		before := source
		key := takeIdentifierChars(source)
		source = skipWhitespace(source[len(key):])
		if suffix, ok := strings.CutPrefix(source, "="); ok {
			source = skipWhitespace(suffix)
			value := takeIdentifierChars(source)
			source = skipWhitespace(source[len(value):])
			context.Set(SharedString(key), SharedString(value))
		} else {
			context.Add(SharedString(key))
		}
		if source == before {
			next, _ := utf8.DecodeRuneInString(source)
			return fmt.Errorf("unexpected character '%q'", next)
		}
	}
	return nil
}

func takeIdentifierChars(source string) string {
	for i, c := range source {
		if !isIdentifierChar(c) {
			return source[:i]
		}
	}
	return source
}

// IsEmpty reports whether this context is empty.
func (k *KeyContext) IsEmpty() bool {
	return len(k.entries) == 0
}

// Clear clears this context.
func (k *KeyContext) Clear() {
	clear(k.entries)
	k.entries = k.entries[:0]
}

// Extend extends this context with another context.
func (k *KeyContext) Extend(other *KeyContext) {
	for _, entry := range other.entries {
		if !k.Contains(entry.Key.AsStr()) {
			k.entries = append(k.entries, entry)
		}
	}
}

// Add adds an identifier to this context, if it's not already in this context.
func (k *KeyContext) Add(identifier SharedString) {
	if !k.Contains(identifier.AsStr()) {
		k.entries = append(k.entries, ContextEntry{Key: identifier, Value: None[SharedString]()})
	}
}

// Set sets a key value pair in this context, if it's not already set.
func (k *KeyContext) Set(key SharedString, value SharedString) {
	if !k.Contains(key.AsStr()) {
		k.entries = append(k.entries, ContextEntry{Key: key, Value: Some(value)})
	}
}

// Contains reports whether this context contains a given identifier or key.
func (k *KeyContext) Contains(key string) bool {
	for _, entry := range k.entries {
		if entry.Key.AsStr() == key {
			return true
		}
	}
	return false
}

// Get returns the associated value for a given identifier or key.
func (k *KeyContext) Get(key string) (SharedString, bool) {
	for _, entry := range k.entries {
		if entry.Key.AsStr() == key {
			return entry.Value.Get()
		}
	}
	return "", false
}

func (k KeyContext) equal(other KeyContext) bool {
	return slices.Equal(k.entries, other.entries)
}

// String formats the context as upstream's Debug implementation does.
func (k KeyContext) String() string {
	var b strings.Builder
	for i, entry := range k.entries {
		if value, ok := entry.Value.Get(); ok {
			fmt.Fprintf(&b, "%s=%s", entry.Key, value)
		} else {
			b.WriteString(entry.Key.AsStr())
		}
		if i+1 < len(k.entries) {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// KeyBindingContextPredicate is a small language describing which contexts correspond to
// which actions. Variants are the KeyBindingContextPredicate* structs; values compare with
// == structurally.
type KeyBindingContextPredicate interface {
	isKeyBindingContextPredicate()
	// String formats the predicate in keymap syntax.
	String() string
	// DepthOf finds the deepest depth at which the predicate matches.
	DepthOf(contexts []KeyContext) (int, bool)
	// Eval evaluates the predicate against contexts arranged from lowest to highest.
	Eval(contexts []KeyContext) bool
	// EvalInner evaluates the predicate against contexts arranged from lowest to highest.
	EvalInner(contexts []KeyContext, allContexts []KeyContext) bool
	// IsSuperset reports whether this predicate matches all contexts matched by other.
	IsSuperset(other KeyBindingContextPredicate) bool
}

// KeyBindingContextPredicateIdentifier matches a given identifier.
type KeyBindingContextPredicateIdentifier struct {
	// Name is the identifier.
	Name SharedString
}

// KeyBindingContextPredicateEqual matches a given key-value pair.
type KeyBindingContextPredicateEqual struct {
	// Left is the key.
	Left SharedString
	// Right is the value.
	Right SharedString
}

// KeyBindingContextPredicateNotEqual matches a given key-value pair not being present.
type KeyBindingContextPredicateNotEqual struct {
	// Left is the key.
	Left SharedString
	// Right is the value.
	Right SharedString
}

// KeyBindingContextPredicateDescendant matches a predicate appearing below another predicate
// in the element tree.
type KeyBindingContextPredicateDescendant struct {
	// Parent is the ancestor predicate.
	Parent KeyBindingContextPredicate
	// Child is the descendant predicate.
	Child KeyBindingContextPredicate
}

// KeyBindingContextPredicateNot inverts another predicate.
type KeyBindingContextPredicateNot struct {
	// Predicate is the inverted predicate.
	Predicate KeyBindingContextPredicate
}

// KeyBindingContextPredicateAnd matches if both of its children match.
type KeyBindingContextPredicateAnd struct {
	// Left is the first child.
	Left KeyBindingContextPredicate
	// Right is the second child.
	Right KeyBindingContextPredicate
}

// KeyBindingContextPredicateOr matches if either of its children match.
type KeyBindingContextPredicateOr struct {
	// Left is the first child.
	Left KeyBindingContextPredicate
	// Right is the second child.
	Right KeyBindingContextPredicate
}

func (KeyBindingContextPredicateIdentifier) isKeyBindingContextPredicate() {}
func (KeyBindingContextPredicateEqual) isKeyBindingContextPredicate()      {}
func (KeyBindingContextPredicateNotEqual) isKeyBindingContextPredicate()   {}
func (KeyBindingContextPredicateDescendant) isKeyBindingContextPredicate() {}
func (KeyBindingContextPredicateNot) isKeyBindingContextPredicate()        {}
func (KeyBindingContextPredicateAnd) isKeyBindingContextPredicate()        {}
func (KeyBindingContextPredicateOr) isKeyBindingContextPredicate()         {}

// String formats the predicate.
func (p KeyBindingContextPredicateIdentifier) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateEqual) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateNotEqual) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateDescendant) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateNot) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateAnd) String() string { return predicateString(p) }

// String formats the predicate.
func (p KeyBindingContextPredicateOr) String() string { return predicateString(p) }

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateIdentifier) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateEqual) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateNotEqual) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateDescendant) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateNot) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateAnd) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// DepthOf finds the deepest depth at which the predicate matches.
func (p KeyBindingContextPredicateOr) DepthOf(contexts []KeyContext) (int, bool) {
	return predicateDepthOf(p, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateIdentifier) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateEqual) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateNotEqual) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateDescendant) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateNot) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateAnd) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// Eval evaluates the predicate against contexts arranged from lowest to highest.
func (p KeyBindingContextPredicateOr) Eval(contexts []KeyContext) bool {
	return p.EvalInner(contexts, contexts)
}

// EvalInner matches the identifier against the last context.
func (p KeyBindingContextPredicateIdentifier) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	return contexts[len(contexts)-1].Contains(p.Name.AsStr())
}

// EvalInner matches the key-value pair against the last context.
func (p KeyBindingContextPredicateEqual) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	value, ok := contexts[len(contexts)-1].Get(p.Left.AsStr())
	return ok && value == p.Right
}

// EvalInner matches the absence of the key-value pair against the last context.
func (p KeyBindingContextPredicateNotEqual) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	value, ok := contexts[len(contexts)-1].Get(p.Left.AsStr())
	return !ok || value != p.Right
}

// EvalInner reports whether no prefix of allContexts matches the inner predicate.
func (p KeyBindingContextPredicateNot) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	for i := range allContexts {
		if p.Predicate.EvalInner(allContexts[:i+1], allContexts) {
			return false
		}
	}
	return true
}

// EvalInner matches the parent against the shallowest matching prefix and the child against
// the rest.
func (p KeyBindingContextPredicateDescendant) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	for i := 0; i < len(contexts)-1; i++ {
		if p.Parent.EvalInner(contexts[:i+1], allContexts) {
			if !p.Child.EvalInner(contexts[i+1:], contexts[i+1:]) {
				return false
			}
			return true
		}
	}
	return false
}

// EvalInner reports whether both children match.
func (p KeyBindingContextPredicateAnd) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	return p.Left.EvalInner(contexts, allContexts) && p.Right.EvalInner(contexts, allContexts)
}

// EvalInner reports whether either child matches.
func (p KeyBindingContextPredicateOr) EvalInner(contexts []KeyContext, allContexts []KeyContext) bool {
	if len(contexts) == 0 {
		return false
	}
	return p.Left.EvalInner(contexts, allContexts) || p.Right.EvalInner(contexts, allContexts)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateIdentifier) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateEqual) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateNotEqual) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateDescendant) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateNot) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateAnd) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

// IsSuperset reports whether this predicate matches all contexts matched by other.
func (p KeyBindingContextPredicateOr) IsSuperset(other KeyBindingContextPredicate) bool {
	return predicateIsSuperset(p, other)
}

func predicateDepthOf(p KeyBindingContextPredicate, contexts []KeyContext) (int, bool) {
	for depth := len(contexts); depth >= 0; depth-- {
		if p.EvalInner(contexts[:depth], contexts) {
			return depth, true
		}
	}
	return 0, false
}

func predicateIsSuperset(p KeyBindingContextPredicate, other KeyBindingContextPredicate) bool {
	if p == other {
		return true
	}

	if or, ok := p.(KeyBindingContextPredicateOr); ok {
		return or.Left.IsSuperset(other) || or.Right.IsSuperset(other)
	}

	switch other := other.(type) {
	case KeyBindingContextPredicateDescendant:
		return p.IsSuperset(other.Child)
	case KeyBindingContextPredicateAnd:
		return p.IsSuperset(other.Left) || p.IsSuperset(other.Right)
	default:
		return false
	}
}

func predicateString(p KeyBindingContextPredicate) string {
	var b strings.Builder
	writePredicate(&b, p)
	return b.String()
}

func writePredicate(b *strings.Builder, p KeyBindingContextPredicate) {
	switch p := p.(type) {
	case KeyBindingContextPredicateIdentifier:
		b.WriteString(p.Name.AsStr())
	case KeyBindingContextPredicateEqual:
		fmt.Fprintf(b, "%s == %s", p.Left, p.Right)
	case KeyBindingContextPredicateNotEqual:
		fmt.Fprintf(b, "%s != %s", p.Left, p.Right)
	case KeyBindingContextPredicateDescendant:
		writePredicate(b, p.Parent)
		b.WriteString(" > ")
		writePredicate(b, p.Child)
	case KeyBindingContextPredicateNot:
		if name, ok := p.Predicate.(KeyBindingContextPredicateIdentifier); ok {
			b.WriteString("!")
			b.WriteString(name.Name.AsStr())
		} else {
			b.WriteString("!(")
			writePredicate(b, p.Predicate)
			b.WriteString(")")
		}
	case KeyBindingContextPredicateAnd:
		first := true
		writeJoined(b, p, " && ", logicalOperatorAnd, func(node KeyBindingContextPredicate) bool {
			_, ok := node.(KeyBindingContextPredicateOr)
			return ok
		}, &first)
	case KeyBindingContextPredicateOr:
		first := true
		writeJoined(b, p, " || ", logicalOperatorOr, func(node KeyBindingContextPredicate) bool {
			_, ok := node.(KeyBindingContextPredicateAnd)
			return ok
		}, &first)
	}
}

func writeJoined(b *strings.Builder, p KeyBindingContextPredicate, separator string, operator logicalOperator, needsParens func(KeyBindingContextPredicate) bool, first *bool) {
	if and, ok := p.(KeyBindingContextPredicateAnd); ok && operator == logicalOperatorAnd {
		writeJoined(b, and.Left, separator, operator, needsParens, first)
		writeJoined(b, and.Right, separator, operator, needsParens, first)
		return
	}
	if or, ok := p.(KeyBindingContextPredicateOr); ok && operator == logicalOperatorOr {
		writeJoined(b, or.Left, separator, operator, needsParens, first)
		writeJoined(b, or.Right, separator, operator, needsParens, first)
		return
	}
	if !*first {
		b.WriteString(separator)
	}
	*first = false

	if needsParens(p) {
		b.WriteString("(")
		writePredicate(b, p)
		b.WriteString(")")
	} else {
		writePredicate(b, p)
	}
}

// KeyBindingContextPredicateParse parses a string in the same format as the keymap's context
// field: identifiers (`StatusBar`), key-value checks (`mode == visible`, `mode != visible`),
// `&&`, `||`, negation with `!`, parentheses, and the child operator `>`
// (`StatusBar > mode == visible`).
func KeyBindingContextPredicateParse(source string) (KeyBindingContextPredicate, error) {
	source = skipWhitespace(source)
	predicate, rest, err := predicateParseExpr(source, 0, 0)
	if err != nil {
		return nil, err
	}
	if next, _ := utf8.DecodeRuneInString(rest); rest != "" {
		return nil, fmt.Errorf("unexpected character '%q'", next)
	}
	return predicate, nil
}

type predicateOp func(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error)

type predicateOperator struct {
	operator    string
	precedence  uint32
	constructor predicateOp
}

var predicateOperators = [...]predicateOperator{
	{">", precedenceChild, predicateNewChild},
	{"&&", precedenceAnd, predicateNewAnd},
	{"||", precedenceOr, predicateNewOr},
	{"==", precedenceEq, predicateNewEq},
	{"!=", precedenceEq, predicateNewNeq},
}

const maxPredicateDepth = 128

var errPredicateTooDeep = fmt.Errorf("context predicate nests deeper than %d levels", maxPredicateDepth)

func predicateParseExpr(source string, minPrecedence uint32, depth int) (KeyBindingContextPredicate, string, error) {
	if depth > maxPredicateDepth {
		return nil, "", errPredicateTooDeep
	}
	predicate, rest, err := predicateParsePrimary(source, depth)
	if err != nil {
		return nil, "", err
	}
	source = rest

parse:
	for {
		for _, op := range predicateOperators {
			if strings.HasPrefix(source, op.operator) && op.precedence >= minPrecedence {
				source = skipWhitespace(source[len(op.operator):])
				right, rest, err := predicateParseExpr(source, op.precedence+1, depth+1)
				if err != nil {
					return nil, "", err
				}
				predicate, err = op.constructor(predicate, right)
				if err != nil {
					return nil, "", err
				}
				source = rest
				continue parse
			}
		}
		break
	}

	return predicate, source, nil
}

func predicateParsePrimary(source string, depth int) (KeyBindingContextPredicate, string, error) {
	next, _ := utf8.DecodeRuneInString(source)
	if source == "" {
		return nil, "", errors.New("unexpected end")
	}
	switch {
	case next == '(':
		source = skipWhitespace(source[1:])
		predicate, rest, err := predicateParseExpr(source, 0, depth+1)
		if err != nil {
			return nil, "", err
		}
		stripped, ok := strings.CutPrefix(rest, ")")
		if !ok {
			return nil, "", errors.New("expected a ')'")
		}
		return predicate, skipWhitespace(stripped), nil
	case next == '!':
		source := skipWhitespace(source[1:])
		predicate, source, err := predicateParseExpr(source, precedenceNot, depth+1)
		if err != nil {
			return nil, "", err
		}
		return KeyBindingContextPredicateNot{Predicate: predicate}, source, nil
	case isIdentifierChar(next):
		length := len(source)
		for i, c := range source {
			if !isIdentifierChar(c) && !isVimOperatorChar(c) {
				length = i
				break
			}
		}
		identifier, rest := source[:length], source[length:]
		return KeyBindingContextPredicateIdentifier{Name: SharedString(identifier)}, skipWhitespace(rest), nil
	case isVimOperatorChar(next):
		operator, rest := source[:1], source[1:]
		return KeyBindingContextPredicateIdentifier{Name: SharedString(operator)}, skipWhitespace(rest), nil
	default:
		return nil, "", fmt.Errorf("unexpected character '%q'", next)
	}
}

func predicateNewOr(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error) {
	return KeyBindingContextPredicateOr{Left: left, Right: right}, nil
}

func predicateNewAnd(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error) {
	return KeyBindingContextPredicateAnd{Left: left, Right: right}, nil
}

func predicateNewChild(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error) {
	return KeyBindingContextPredicateDescendant{Parent: left, Child: right}, nil
}

func predicateNewEq(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error) {
	l, lok := left.(KeyBindingContextPredicateIdentifier)
	r, rok := right.(KeyBindingContextPredicateIdentifier)
	if lok && rok {
		return KeyBindingContextPredicateEqual{Left: l.Name, Right: r.Name}, nil
	}
	return nil, errors.New("operands of == must be identifiers")
}

func predicateNewNeq(left, right KeyBindingContextPredicate) (KeyBindingContextPredicate, error) {
	l, lok := left.(KeyBindingContextPredicateIdentifier)
	r, rok := right.(KeyBindingContextPredicateIdentifier)
	if lok && rok {
		return KeyBindingContextPredicateNotEqual{Left: l.Name, Right: r.Name}, nil
	}
	return nil, errors.New("operands of != must be identifiers")
}

type logicalOperator uint8

const (
	logicalOperatorAnd logicalOperator = iota
	logicalOperatorOr
)

const (
	precedenceChild uint32 = 1
	precedenceOr    uint32 = 2
	precedenceAnd   uint32 = 3
	precedenceEq    uint32 = 4
	precedenceNot   uint32 = 5
)

func isIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.Is(unicode.Other_Alphabetic, c) || unicode.IsNumber(c) || c == '_' || c == '-'
}

func isVimOperatorChar(c rune) bool {
	return c == '>' || c == '<' || c == '~' || c == '"' || c == '?'
}

func skipWhitespace(source string) string {
	for i, c := range source {
		if !unicode.Is(unicode.White_Space, c) {
			return source[i:]
		}
	}
	return ""
}
