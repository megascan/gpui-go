//go:build windows

package windows

import (
	"io"
	"strconv"
	"sync"
	"unicode/utf16"
	"unsafe"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const (
	maxRetainedScratchCapacity = 4096
	maxPooledBuffers           = 2
)

type stringScratch struct {
	mu   sync.Mutex
	pool [][]uint16
}

var stringScratchPool stringScratch

type stringBuffer struct {
	buf     []uint16
	scratch *stringScratch
}

func acquireStringBuffer() *stringBuffer {
	return stringScratchPool.acquire()
}

func (s *stringScratch) acquire() *stringBuffer {
	s.mu.Lock()
	defer s.mu.Unlock()
	var buf []uint16
	if n := len(s.pool); n > 0 {
		buf = s.pool[n-1]
		s.pool = s.pool[:n-1]
	}
	return &stringBuffer{buf: buf, scratch: s}
}

func (b *stringBuffer) release() {
	if b.scratch == nil {
		return
	}
	buf := b.buf
	b.buf = nil
	scratch := b.scratch
	b.scratch = nil
	if cap(buf) > maxRetainedScratchCapacity {
		buf = make([]uint16, 0, maxRetainedScratchCapacity)
	}
	scratch.mu.Lock()
	defer scratch.mu.Unlock()
	if len(scratch.pool) < maxPooledBuffers {
		scratch.pool = append(scratch.pool, buf)
	}
}

type wideString struct {
	buf *[]uint16
}

func newWideString(buffer *[]uint16) wideString {
	*buffer = (*buffer)[:0]
	return wideString{buf: buffer}
}

func (w wideString) WriteString(s string) (int, error) {
	for _, r := range s {
		*w.buf = utf16.AppendRune(*w.buf, r)
	}
	return len(s), nil
}

func (w wideString) writeChar(c rune) {
	*w.buf = utf16.AppendRune(*w.buf, c)
}

func (w wideString) equal(other wideString) bool {
	a, b := *w.buf, *other.buf
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (w wideString) toBSTR() com.BSTR {
	return bstrFromWide(*w.buf)
}

func (w wideString) toVariant() Variant {
	return variantFromBSTR(w.toBSTR())
}

func variantFromLocaleName(name string) Variant {
	lcid := localeNameToLCID(name, localeAllowNeutralNames)
	if lcid == 0 {
		return variantEmpty()
	}
	return variantFromI32(int32(lcid))
}

// Variant owns a VARIANT until it is converted with into.
type Variant struct {
	v com.VARIANT
}

func (v Variant) into() com.VARIANT {
	return v.v
}

func variantEmpty() Variant {
	return Variant{v: com.VariantEmpty()}
}

func (v Variant) isEmpty() bool {
	return v.v.VT == com.VT_EMPTY
}

func variantFromBSTR(b com.BSTR) Variant {
	v := com.VARIANT{VT: com.VT_BSTR}
	*(*com.BSTR)(unsafe.Pointer(&v.Val[0])) = b
	return Variant{v: v}
}

func variantFromString(value string, buffer *[]uint16) Variant {
	result := newWideString(buffer)
	result.WriteString(value)
	return result.toVariant()
}

func variantFromUnknown(value *com.IUnknown) Variant {
	v := com.VARIANT{VT: com.VT_UNKNOWN}
	*(**com.IUnknown)(unsafe.Pointer(&v.Val[0])) = value
	return Variant{v: v}
}

func variantFromI32(value int32) Variant {
	return Variant{v: com.VariantI4(value)}
}

func variantFromF64(value float64) Variant {
	return Variant{v: com.VariantR8(value)}
}

func variantFromBool(value bool) Variant {
	return Variant{v: com.VariantBool(value)}
}

func variantFromColor(value a11y.Color) Variant {
	rgb := int32(value.Red) | int32(value.Green)<<8 | int32(value.Blue)<<16
	return variantFromI32(rgb)
}

func variantFromTextDecorationStyle(value a11y.TextDecorationStyle) Variant {
	var style int32
	switch value {
	case a11y.TextDecorationStyleSolid:
		style = textDecorationLineStyleSingle
	case a11y.TextDecorationStyleDotted:
		style = textDecorationLineStyleDot
	case a11y.TextDecorationStyleDashed:
		style = textDecorationLineStyleDash
	case a11y.TextDecorationStyleDouble:
		style = textDecorationLineStyleDouble
	case a11y.TextDecorationStyleWavy:
		style = textDecorationLineStyleWavy
	}
	return variantFromI32(style)
}

func variantFromTextAlign(value a11y.TextAlign) Variant {
	var align int32
	switch value {
	case a11y.TextAlignLeft:
		align = horizontalTextAlignmentLeft
	case a11y.TextAlignRight:
		align = horizontalTextAlignmentRight
	case a11y.TextAlignCenter:
		align = horizontalTextAlignmentCentered
	case a11y.TextAlignJustify:
		align = horizontalTextAlignmentJustified
	}
	return variantFromI32(align)
}

func variantFromOption[T any](value T, ok bool, convert func(T) Variant) Variant {
	if !ok {
		return variantEmpty()
	}
	return convert(value)
}

func variantFromTextRangePropertyValue[T comparable](value consumer.TextRangePropertyValue[T], convert func(T) Variant) Variant {
	if single, ok := value.Single(); ok {
		return convert(single)
	}
	mixed, err := uiaGetReservedMixedAttributeValue()
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	return variantFromUnknown(mixed)
}

func variantFromUnknowns(value []*com.IUnknown) Variant {
	defer func() {
		for _, item := range value {
			item.Release()
		}
	}()
	if len(value) == 0 {
		return variantEmpty()
	}
	return Variant{v: com.VariantArray(com.VT_UNKNOWN, safeArrayFromComSlice(value))}
}

func safeArrayFromPrimitiveSlice[T any](vt com.VARTYPE, slice []T) *com.SAFEARRAY {
	sa, err := com.SafeArrayCreateVector(vt, 0, uint32(len(slice)))
	if err != nil || sa == nil {
		panic("SAFEARRAY allocation failed")
	}
	for i := range slice {
		if err := com.SafeArrayPutElement(sa, int32(i), unsafe.Pointer(&slice[i])); err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
	}
	return sa
}

func safeArrayFromI32Slice(slice []int32) *com.SAFEARRAY {
	return safeArrayFromPrimitiveSlice(com.VT_I4, slice)
}

func safeArrayFromF64Slice(slice []float64) *com.SAFEARRAY {
	return safeArrayFromPrimitiveSlice(com.VT_R8, slice)
}

func safeArrayFromComSlice(slice []*com.IUnknown) *com.SAFEARRAY {
	sa, err := com.SafeArrayCreateVector(com.VT_UNKNOWN, 0, uint32(len(slice)))
	if err != nil || sa == nil {
		panic("SAFEARRAY allocation failed")
	}
	for i, item := range slice {
		if err := com.SafeArrayPutElement(sa, int32(i), unsafe.Pointer(item)); err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
	}
	return sa
}

type queuedEvent interface {
	isQueuedEvent()
	release()
}

type queuedEventSimple struct {
	element *com.IUnknown
	eventID uiaEventID
}

type queuedEventPropertyChanged struct {
	element    *com.IUnknown
	propertyID uiaPropertyID
	oldValue   com.VARIANT
	newValue   com.VARIANT
}

func (*queuedEventSimple) isQueuedEvent()          {}
func (*queuedEventPropertyChanged) isQueuedEvent() {}

func (e *queuedEventSimple) release() {
	if e.element != nil {
		e.element.Release()
		e.element = nil
	}
}

func (e *queuedEventPropertyChanged) release() {
	if e.element != nil {
		e.element.Release()
		e.element = nil
	}
	com.VariantClear(&e.oldValue)
	com.VariantClear(&e.newValue)
}

func notImplemented() error {
	return com.E_NOTIMPL
}

func invalidArg() error {
	return com.E_INVALIDARG
}

func requiredParam(param *com.IUnknown) (*com.IUnknown, error) {
	if param == nil {
		return nil, invalidArg()
	}
	return param, nil
}

func elementNotAvailable() error {
	return uiaEElementNotAvailable
}

func elementNotEnabled() error {
	return uiaEElementNotEnabled
}

func invalidOperation() error {
	return uiaEInvalidOperation
}

func notSupported() error {
	return uiaENotSupported
}

func clientTopLeft(hwnd WindowHandle) a11y.Point {
	var result win32.POINT
	if !win32.ClientToScreen(hwnd.HWND, &result) {
		panic("called `Result::unwrap()` on an `Err` value: ClientToScreen failed")
	}
	return a11y.NewPoint(float64(result.X), float64(result.Y))
}

func windowTitle(hwnd WindowHandle, buffer *[]uint16) (com.BSTR, bool) {
	result := int(win32.DefWindowProcW(hwnd.HWND, wmGetTextLength, 0, 0))
	if result <= 0 {
		return nil, false
	}
	capacity := result + 1
	if cap(*buffer) < capacity {
		*buffer = make([]uint16, 0, capacity)
	}
	*buffer = (*buffer)[:capacity]
	result = int(win32.DefWindowProcW(hwnd.HWND, wmGetText, win32.WPARAM(capacity), win32.LPARAM(uintptr(unsafe.Pointer(&(*buffer)[0])))))
	if result <= 0 {
		*buffer = (*buffer)[:0]
		return nil, false
	}
	*buffer = (*buffer)[:result]
	return bstrFromWide(*buffer), true
}

func toolkitDescription(state *consumer.TreeState, buffer *[]uint16) wideString {
	result := newWideString(buffer)
	result.WriteString(state.ToolkitName())
	if version, ok := state.ToolkitVersion(); ok {
		result.writeChar(' ')
		result.WriteString(version)
	}
	return result
}

func upgrade(w weak.Pointer[Context]) (*Context, error) {
	strong := w.Value()
	if strong == nil || strong.released.Load() {
		return nil, elementNotAvailable()
	}
	return strong, nil
}

type ariaProperties struct {
	inner         io.StringWriter
	needSeparator bool
}

func newAriaProperties(inner io.StringWriter) *ariaProperties {
	return &ariaProperties{inner: inner}
}

func (p *ariaProperties) writeProperty(name, value string) error {
	if p.needSeparator {
		if _, err := p.inner.WriteString(";"); err != nil {
			return err
		}
	}
	if _, err := p.inner.WriteString(name); err != nil {
		return err
	}
	if _, err := p.inner.WriteString("="); err != nil {
		return err
	}
	if _, err := p.inner.WriteString(value); err != nil {
		return err
	}
	p.needSeparator = true
	return nil
}

func (p *ariaProperties) writeBoolProperty(name string, value bool) error {
	if value {
		return p.writeProperty(name, "true")
	}
	return p.writeProperty(name, "false")
}

func (p *ariaProperties) writeUsizeProperty(name string, value int) error {
	return p.writeProperty(name, strconv.Itoa(value))
}

func (p *ariaProperties) hasProperties() bool {
	return p.needSeparator
}
