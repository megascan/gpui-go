//go:build darwin

package cf

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestCf_Layout(t *testing.T) {
	if s := unsafe.Sizeof(Range{}); s != 16 {
		t.Fatalf("sizeof(CFRange) = %d, want 16", s)
	}
	if StringEncodingUTF8 != 0x08000100 {
		t.Fatalf("kCFStringEncodingUTF8 = %#x", uint32(StringEncodingUTF8))
	}
}

func TestCf_Symbols(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if typeDictionaryKeyCallBacksAddr == 0 || typeDictionaryValueCallBacksAddr == 0 || typeArrayCallBacksAddr == 0 {
		t.Fatal("callback structs not resolved")
	}
	if BooleanTrue() == 0 || BooleanFalse() == 0 || !BooleanTrue().Value() || BooleanFalse().Value() {
		t.Fatal("CFBoolean constants wrong")
	}
	ids := map[string]TypeID{
		"string":     StringGetTypeID(),
		"data":       DataGetTypeID(),
		"dictionary": DictionaryGetTypeID(),
		"array":      ArrayGetTypeID(),
		"boolean":    BooleanGetTypeID(),
		"number":     NumberGetTypeID(),
	}
	seen := map[TypeID]string{}
	for name, id := range ids {
		if prev, ok := seen[id]; ok {
			t.Errorf("%s and %s share type id %d", name, prev, id)
		}
		seen[id] = name
	}
	if GetTypeID(TypeRef(BooleanTrue())) != BooleanGetTypeID() {
		t.Error("kCFBooleanTrue is not a CFBoolean")
	}
}

func TestCf_StringAndData(t *testing.T) {
	for _, s := range []string{"", "server.example", "ünïcødé ✓"} {
		str := StringCreate(s)
		if got := str.Ref().GoString(); got != s {
			t.Errorf("GoString(StringCreate(%q)) = %q", s, got)
		}
		if _, ok := AsString(str.TypeRef()); !ok {
			t.Errorf("AsString(%q) failed", s)
		}
		if _, ok := AsData(str.TypeRef()); ok {
			t.Errorf("AsData accepted a string")
		}
		str.Release()
		str.Release()
	}
	data := DataCreate([]byte{0, 1, 2, 254})
	defer data.Release()
	if got := data.Ref().Bytes(); !reflect.DeepEqual(got, []byte{0, 1, 2, 254}) {
		t.Fatalf("Bytes = %v", got)
	}
}

func TestCf_DictionaryAndArray(t *testing.T) {
	k1, k2 := StringCreate("k1"), StringCreate("k2")
	defer k1.Release()
	defer k2.Release()
	v := DataCreate([]byte("secret"))
	defer v.Release()
	dict := DictionaryCreate([]TypeRef{k1.TypeRef(), k2.TypeRef()}, []TypeRef{v.TypeRef(), TypeRef(BooleanTrue())})
	defer dict.Release()
	if dict.Ref().Count() != 2 {
		t.Fatalf("Count = %d", dict.Ref().Count())
	}
	if GetRetainCount(v.TypeRef()) < 2 {
		t.Error("dictionary did not retain its value")
	}
	got, ok := dict.Ref().Get(k1.TypeRef())
	if !ok {
		t.Fatal("k1 missing")
	}
	d, ok := AsData(got)
	if !ok || string(d.Bytes()) != "secret" {
		t.Fatalf("k1 = %v", d.Bytes())
	}
	missing := StringCreate("missing")
	defer missing.Release()
	if _, ok := dict.Ref().Get(missing.TypeRef()); ok {
		t.Fatal("missing key found")
	}
	arr := ArrayCreate([]TypeRef{k1.TypeRef(), k2.TypeRef()})
	defer arr.Release()
	if arr.Ref().Count() != 2 || !Equal(arr.Ref().At(1), k2.TypeRef()) {
		t.Fatal("array contents wrong")
	}
	again := RetainRef(arr.Ref())
	again.Release()
	defer func() {
		if recover() == nil {
			t.Error("Ref after Release did not panic")
		}
	}()
	_ = again.Ref()
}

func TestCf_StringCreateRejectsInvalidUTF8(t *testing.T) {
	if s := StringCreate("bad\xff"); s != nil {
		s.Release()
		t.Error("StringCreate accepted invalid UTF-8")
	}
}
