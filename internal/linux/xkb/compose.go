//go:build linux

package xkb

type (
	// ComposeCompileFlags are xkb_compose_compile_flags.
	ComposeCompileFlags = uint32
	// ComposeFormat is an xkb_compose_format.
	ComposeFormat = uint32
	// ComposeStateFlags are xkb_compose_state_flags.
	ComposeStateFlags = uint32
)

// Compose flag values from xkbcommon-compose.h.
const (
	ComposeCompileNoFlags ComposeCompileFlags = 0
	ComposeFormatTextV1   ComposeFormat       = 1
	ComposeStateNoFlags   ComposeStateFlags   = 0
)

// Status is an xkb_compose_status.
type Status uint8

// Status values.
const (
	StatusNothing Status = iota
	StatusComposing
	StatusComposed
	StatusCancelled
)

// FeedResult is an xkb_compose_feed_result.
type FeedResult uint8

// FeedResult values.
const (
	FeedResultIgnored FeedResult = iota
	FeedResultAccepted
)

// ComposeTable is an xkb_compose_table.
type ComposeTable struct{ ptr uintptr }

// NewComposeTableFromLocale calls xkb_compose_table_new_from_locale; ok is false on failure.
func NewComposeTableFromLocale(ctx *Context, locale string, flags ComposeCompileFlags) (*ComposeTable, bool) {
	mustLoad()
	p := xkb_compose_table_new_from_locale(ctx.ptr, locale, flags)
	if p == 0 {
		return nil, false
	}
	return &ComposeTable{ptr: p}, true
}

// NewComposeTableFromBuffer calls xkb_compose_table_new_from_buffer; ok is false on failure.
func NewComposeTableFromBuffer(ctx *Context, buffer []byte, locale string, format ComposeFormat, flags ComposeCompileFlags) (*ComposeTable, bool) {
	mustLoad()
	var p uintptr
	if len(buffer) == 0 {
		var zero byte
		p = xkb_compose_table_new_from_buffer(ctx.ptr, &zero, 0, locale, format, flags)
	} else {
		p = xkb_compose_table_new_from_buffer(ctx.ptr, &buffer[0], uintptr(len(buffer)), locale, format, flags)
	}
	if p == 0 {
		return nil, false
	}
	return &ComposeTable{ptr: p}, true
}

// GetRawPtr returns the underlying pointer.
func (t *ComposeTable) GetRawPtr() uintptr { return t.ptr }

// Clone takes a new reference.
func (t *ComposeTable) Clone() *ComposeTable { return &ComposeTable{ptr: xkb_compose_table_ref(t.ptr)} }

// Close drops this reference. It is idempotent.
func (t *ComposeTable) Close() {
	if t.ptr != 0 {
		xkb_compose_table_unref(t.ptr)
		t.ptr = 0
	}
}

// ComposeState is an xkb_compose_state.
type ComposeState struct{ ptr uintptr }

// NewComposeState calls xkb_compose_state_new.
func NewComposeState(table *ComposeTable, flags ComposeStateFlags) *ComposeState {
	mustLoad()
	return &ComposeState{ptr: xkb_compose_state_new(table.ptr, flags)}
}

// GetRawPtr returns the underlying pointer.
func (s *ComposeState) GetRawPtr() uintptr { return s.ptr }

// Clone takes a new reference to the same state.
func (s *ComposeState) Clone() *ComposeState { return &ComposeState{ptr: xkb_compose_state_ref(s.ptr)} }

// Close drops this reference. It is idempotent.
func (s *ComposeState) Close() {
	if s.ptr != 0 {
		xkb_compose_state_unref(s.ptr)
		s.ptr = 0
	}
}

// ComposeTable returns a new reference to the state's table; the caller closes it.
func (s *ComposeState) ComposeTable() *ComposeTable {
	return &ComposeTable{ptr: xkb_compose_table_ref(xkb_compose_state_get_compose_table(s.ptr))}
}

// Feed feeds a keysym into the compose state.
func (s *ComposeState) Feed(keysym Keysym) FeedResult {
	return FeedResult(xkb_compose_state_feed(s.ptr, uint32(keysym)))
}

// Reset resets the compose sequence.
func (s *ComposeState) Reset() { xkb_compose_state_reset(s.ptr) }

// Status returns the current compose status.
func (s *ComposeState) Status() Status { return Status(xkb_compose_state_get_status(s.ptr)) }

// Utf8 returns the composed text, if any.
func (s *ComposeState) Utf8() (string, bool) {
	var buf [256]byte
	n := xkb_compose_state_get_utf8(s.ptr, &buf[0], uintptr(len(buf)))
	if n == 0 {
		return "", false
	}
	return string(buf[:clampLen(n, len(buf)-1)]), true
}

// Keysym returns the composed keysym, if any.
func (s *ComposeState) Keysym() (Keysym, bool) {
	k := Keysym(xkb_compose_state_get_one_sym(s.ptr))
	if k == KEY_NoSymbol {
		return 0, false
	}
	return k, true
}
