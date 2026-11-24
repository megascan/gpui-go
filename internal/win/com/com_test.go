//go:build windows

package com

import (
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestHRESULT_Values(t *testing.T) {
	for _, c := range []struct {
		name string
		hr   HRESULT
		want uint32
	}{
		{"E_NOTIMPL", E_NOTIMPL, 0x80004001},
		{"E_NOINTERFACE", E_NOINTERFACE, 0x80004002},
		{"E_POINTER", E_POINTER, 0x80004003},
		{"E_ABORT", E_ABORT, 0x80004004},
		{"E_FAIL", E_FAIL, 0x80004005},
		{"E_UNEXPECTED", E_UNEXPECTED, 0x8000FFFF},
		{"E_ACCESSDENIED", E_ACCESSDENIED, 0x80070005},
		{"E_HANDLE", E_HANDLE, 0x80070006},
		{"E_OUTOFMEMORY", E_OUTOFMEMORY, 0x8007000E},
		{"E_INVALIDARG", E_INVALIDARG, 0x80070057},
		{"CO_E_NOTINITIALIZED", CO_E_NOTINITIALIZED, 0x800401F0},
		{"RPC_E_CHANGED_MODE", RPC_E_CHANGED_MODE, 0x80010106},
		{"REGDB_E_CLASSNOTREG", REGDB_E_CLASSNOTREG, 0x80040154},
		{"CLASS_E_NOAGGREGATION", CLASS_E_NOAGGREGATION, 0x80040110},
	} {
		if uint32(c.hr) != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, uint32(c.hr), c.want)
		}
	}
	if HRESULTFromWin32(5) != E_ACCESSDENIED {
		t.Errorf("HRESULTFromWin32(5) = %#x", uint32(HRESULTFromWin32(5)))
	}
	if S_FALSE.Err() != nil || E_FAIL.Err() == nil {
		t.Error("Err() does not follow the sign of the HRESULT")
	}
	if got := HRESULTFromError(windows.ERROR_ACCESS_DENIED); got != E_ACCESSDENIED {
		t.Errorf("HRESULTFromError(ERROR_ACCESS_DENIED) = %#x", uint32(got))
	}
}

func TestHRESULT_ErrorFormat(t *testing.T) {
	message := E_FAIL.Message()
	if message == "" || strings.TrimSpace(message) != message {
		t.Errorf("E_FAIL.Message() = %q", message)
	}
	if got := E_FAIL.Error(); got != message+" (0x80004005)" {
		t.Errorf("E_FAIL.Error() = %q", got)
	}
	if got := HRESULT(-0x1).Error(); got != "0xFFFFFFFF" && !strings.HasSuffix(got, " (0xFFFFFFFF)") {
		t.Errorf("HRESULT(-1).Error() = %q", got)
	}
}

func TestCom_LayoutSizes(t *testing.T) {
	var v VARIANT
	var a SAFEARRAY
	for _, c := range []struct {
		name      string
		got, want uintptr
	}{
		{"VARIANT", unsafe.Sizeof(v), 24},
		{"VARIANT.Val", unsafe.Offsetof(v.Val), 8},
		{"SAFEARRAYBOUND", unsafe.Sizeof(SAFEARRAYBOUND{}), 8},
		{"SAFEARRAY", unsafe.Sizeof(a), 32},
		{"SAFEARRAY.PvData", unsafe.Offsetof(a.PvData), 16},
		{"SAFEARRAY.Rgsabound", unsafe.Offsetof(a.Rgsabound), 24},
		{"IUnknown", unsafe.Sizeof(IUnknown{}), 8},
		{"GUID", unsafe.Sizeof(GUID{}), 16},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestCom_MustGUID(t *testing.T) {
	g := MustGUID("{00000000-0000-0000-C000-000000000046}")
	if g != (GUID{Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}) {
		t.Fatalf("IID_IUnknown = %v", g)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustGUID accepted a malformed GUID")
		}
	}()
	MustGUID("not a guid")
}

var (
	iidTestAdder = MustGUID("{6E0F9C11-2E7B-4C55-9F51-2B3E5C7A1D01}")
	iidTestNamer = MustGUID("{6E0F9C11-2E7B-4C55-9F51-2B3E5C7A1D02}")

	testAdderVtable = NewLazyVtable(func() *Vtable {
		return NewVtable([]GUID{iidTestAdder}, windows.NewCallback(func(this *IUnknown, a, b uintptr, out *int32) uintptr {
			impl, ok := ImplOf[*testImpl](this)
			if !ok {
				return E_UNEXPECTED.Uintptr()
			}
			*out = int32(a) + int32(b) + impl.bias
			return S_OK.Uintptr()
		}))
	})
	testNamerVtable = NewLazyVtable(func() *Vtable {
		return NewVtable([]GUID{iidTestNamer}, windows.NewCallback(func(this *IUnknown, out *int32) uintptr {
			*out = 42
			return S_OK.Uintptr()
		}))
	})
)

type testImpl struct {
	bias     int32
	released bool
}

func (t *testImpl) FinalRelease() {
	t.released = true
}

func TestObject_CallsThroughVtable(t *testing.T) {
	impl := &testImpl{bias: 100}
	object := NewObject(impl, testAdderVtable.Get())
	adder := object.Interface(0)

	var sum int32
	if err := CallHR(adder.Ptr(), IUnknownSlots, 2, 3, uintptr(unsafe.Pointer(&sum))); err != nil {
		t.Fatal(err)
	}
	if sum != 105 {
		t.Fatalf("Add(2, 3) = %d, want 105", sum)
	}

	if got := adder.AddRef(); got != 2 {
		t.Fatalf("AddRef = %d, want 2", got)
	}
	if got := adder.Release(); got != 1 {
		t.Fatalf("Release = %d, want 1", got)
	}

	unknown, err := adder.QueryInterface(&IID_IUnknown)
	if err != nil {
		t.Fatal(err)
	}
	if unknown != adder {
		t.Fatal("IID_IUnknown is not the identity pointer")
	}
	if _, err := adder.QueryInterface(&iidTestNamer); err != E_NOINTERFACE {
		t.Fatalf("QueryInterface(unsupported) = %v, want E_NOINTERFACE", err)
	}
	if object.RefCount() != 2 {
		t.Fatalf("RefCount = %d, want 2", object.RefCount())
	}
	unknown.Release()
	if impl.released {
		t.Fatal("released before the last reference")
	}
	if got := adder.Release(); got != 0 {
		t.Fatalf("final Release = %d", got)
	}
	if !impl.released {
		t.Fatal("FinalRelease not called")
	}
	if ObjectOf(adder) != nil {
		t.Fatal("released object still registered")
	}
}

func TestObject_MultipleInterfaces(t *testing.T) {
	impl := &testImpl{}
	object := NewObject(impl, testAdderVtable.Get(), testNamerVtable.Get())
	adder := object.Interface(0)

	namer, err := adder.QueryInterface(&iidTestNamer)
	if err != nil {
		t.Fatal(err)
	}
	if namer != object.Interface(1) {
		t.Fatal("QueryInterface returned the wrong vtable slot")
	}
	var value int32
	if err := CallHR(namer.Ptr(), IUnknownSlots, uintptr(unsafe.Pointer(&value))); err != nil {
		t.Fatal(err)
	}
	if value != 42 {
		t.Fatalf("Name() = %d", value)
	}
	identity, err := namer.QueryInterface(&IID_IUnknown)
	if err != nil {
		t.Fatal(err)
	}
	if identity != adder {
		t.Fatal("identity differs between interfaces")
	}
	identity.Release()
	namer.Release()
	adder.Release()
	if !impl.released {
		t.Fatal("FinalRelease not called")
	}
}

func TestOleaut_BSTRRoundTrip(t *testing.T) {
	b, err := NewBSTR("héllo")
	if err != nil {
		t.Fatal(err)
	}
	defer FreeBSTR(b)
	if BSTRLen(b) != 5 {
		t.Fatalf("BSTRLen = %d", BSTRLen(b))
	}
	if got := BSTRToString(b); got != "héllo" {
		t.Fatalf("BSTRToString = %q", got)
	}
	empty, err := NewBSTR("")
	if err != nil {
		t.Fatal(err)
	}
	defer FreeBSTR(empty)
	if BSTRToString(empty) != "" || BSTRToString(nil) != "" {
		t.Fatal("empty BSTR is not empty")
	}
}

func TestOleaut_BSTREmbeddedNul(t *testing.T) {
	b, err := NewBSTR("a\x00b😀")
	if err != nil {
		t.Fatal(err)
	}
	defer FreeBSTR(b)
	if BSTRLen(b) != 5 {
		t.Fatalf("BSTRLen = %d", BSTRLen(b))
	}
	if got := BSTRToString(b); got != "a\x00b😀" {
		t.Fatalf("BSTRToString = %q", got)
	}
}

func TestOleaut_Variants(t *testing.T) {
	i4 := VariantI4(-7)
	if i4.VT != VT_I4 || i4.I4() != -7 {
		t.Fatalf("VariantI4 = %+v", i4)
	}
	r8 := VariantR8(1.5)
	if r8.VT != VT_R8 || r8.R8() != 1.5 {
		t.Fatalf("VariantR8 = %+v", r8)
	}
	b := VariantBool(true)
	if b.VT != VT_BOOL || !b.Bool() {
		t.Fatalf("VariantBool = %+v", b)
	}
	s, err := VariantBSTR("text")
	if err != nil {
		t.Fatal(err)
	}
	if BSTRToString(s.BSTR()) != "text" {
		t.Fatal("VariantBSTR lost its string")
	}
	if err := VariantClear(&s); err != nil {
		t.Fatal(err)
	}
	if s.VT != VT_EMPTY {
		t.Fatalf("VariantClear left VT %d", s.VT)
	}
}

func TestOleaut_SafeArrayInt32(t *testing.T) {
	array, err := SafeArrayFromInt32([]int32{4, 5, 6})
	if err != nil {
		t.Fatal(err)
	}
	defer SafeArrayDestroy(array)
	vt, err := SafeArrayGetVartype(array)
	if err != nil || vt != VT_I4 {
		t.Fatalf("SafeArrayGetVartype = %d, %v", vt, err)
	}
	upper, err := SafeArrayGetUBound(array, 1)
	if err != nil || upper != 2 {
		t.Fatalf("SafeArrayGetUBound = %d, %v", upper, err)
	}
	var value int32
	if err := SafeArrayGetElement(array, 1, unsafe.Pointer(&value)); err != nil || value != 5 {
		t.Fatalf("SafeArrayGetElement(1) = %d, %v", value, err)
	}
}

func TestWinrt_HSTRINGRoundTrip(t *testing.T) {
	h, err := NewHSTRING("Windows.UI.ViewManagement.UISettings")
	if err != nil {
		t.Fatal(err)
	}
	defer h.Delete()
	if got := h.String(); got != "Windows.UI.ViewManagement.UISettings" {
		t.Fatalf("HSTRING = %q", got)
	}
	empty, err := NewHSTRING("")
	if err != nil || empty != 0 || empty.String() != "" {
		t.Fatalf("empty HSTRING = %v, %v", empty, err)
	}
}

func TestWinrt_HSTRINGEmbeddedNul(t *testing.T) {
	h, err := NewHSTRING("a\x00b😀")
	if err != nil {
		t.Fatal(err)
	}
	defer h.Delete()
	if got := h.String(); got != "a\x00b😀" {
		t.Fatalf("HSTRING = %q", got)
	}
}

func TestWinrt_ActivateInstance(t *testing.T) {
	instance, err := RoActivateInstance("Windows.UI.ViewManagement.UISettings")
	if err != nil {
		t.Skipf("UISettings unavailable: %v", err)
	}
	defer instance.Release()
	name, err := instance.GetRuntimeClassName()
	if err != nil {
		t.Fatal(err)
	}
	if name != "Windows.UI.ViewManagement.UISettings" {
		t.Fatalf("GetRuntimeClassName = %q", name)
	}
}
