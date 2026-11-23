package gpui

import (
	"fmt"
	"strings"
	"unicode"
)

// SharedString is an immutable string that can be cheaply cloned (D-STRING).
type SharedString string

// NewSharedStringStatic creates a SharedString from a static string.
func NewSharedStringStatic(str string) SharedString {
	return SharedString(str)
}

// NewSharedString creates a SharedString.
func NewSharedString(str string) SharedString {
	return SharedString(str)
}

// AsStr returns the underlying string.
func (s SharedString) AsStr() string {
	return string(s)
}

// String returns the underlying string.
func (s SharedString) String() string {
	return string(s)
}

// GoString formats the string quoted, like its Rust Debug output.
func (s SharedString) GoString() string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range string(s) {
		switch r {
		case 0:
			b.WriteString(`\0`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			if unicode.IsPrint(r) || r == ' ' {
				b.WriteRune(r)
			} else {
				fmt.Fprintf(&b, `\u{%x}`, r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
