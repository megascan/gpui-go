//go:build darwin

package objc

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestObjc_StructLayout(t *testing.T) {
	cases := []struct {
		name string
		size uintptr
		want uintptr
	}{
		{"CGPoint", unsafe.Sizeof(CGPoint{}), 16},
		{"CGSize", unsafe.Sizeof(CGSize{}), 16},
		{"CGRect", unsafe.Sizeof(CGRect{}), 32},
		{"NSRange", unsafe.Sizeof(NSRange{}), 16},
	}
	for _, c := range cases {
		if c.size != c.want {
			t.Errorf("sizeof(%s) = %d, want %d", c.name, c.size, c.want)
		}
	}
	var r CGRect
	if off := unsafe.Offsetof(r.Size); off != 16 {
		t.Errorf("offsetof(CGRect.size) = %d, want 16", off)
	}
	var p CGPoint
	if off := unsafe.Offsetof(p.Y); off != 8 {
		t.Errorf("offsetof(CGPoint.y) = %d, want 8", off)
	}
	var n NSRange
	if off := unsafe.Offsetof(n.Length); off != 8 {
		t.Errorf("offsetof(NSRange.length) = %d, want 8", off)
	}
}

func TestObjc_ActivityOptions(t *testing.T) {
	if NSActivityUserInitiatedAllowingIdleSystemSleep != 0x00EFFFFF {
		t.Fatalf("NSActivityUserInitiatedAllowingIdleSystemSleep = %#x", uint64(NSActivityUserInitiatedAllowingIdleSystemSleep))
	}
}

func TestObjc_ClassesAndSelectors(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	selectors := map[string][]string{
		"NSString":       {"initWithBytes:length:encoding:", "lengthOfBytesUsingEncoding:", "getCString:maxLength:encoding:"},
		"NSArray":        {"initWithObjects:count:", "count", "objectAtIndex:"},
		"NSDictionary":   {"initWithObjects:forKeys:count:", "objectForKey:"},
		"NSData":         {"initWithBytes:length:", "length", "getBytes:length:"},
		"NSURL":          {"initFileURLWithPath:", "initWithString:", "absoluteString", "path"},
		"NSError":        {"domain", "code", "localizedDescription"},
		"NSObject":       {"init", "respondsToSelector:"},
		"NSValue":        {"rangeValue"},
		"NSProcessInfo":  {"beginActivityWithOptions:reason:", "endActivity:", "thermalState"},
		"NSThread":       {"isMainThread"},
		"NSRunLoop":      {"addTimer:forMode:"},
		"NSNotification": {"name", "object"},
	}
	for class, sels := range selectors {
		c := GetClass(class)
		if c == 0 {
			t.Errorf("class %s not found", class)
			continue
		}
		if GetClass(class) != c {
			t.Errorf("class cache for %s returned a different class", class)
		}
		for _, s := range sels {
			if !InstancesRespondToSelector(c, Sel(s)) {
				t.Errorf("%s does not respond to %s", class, s)
			}
		}
	}
	if Sel("count") != Sel("count") {
		t.Error("selector cache is not stable")
	}
	if GetClass("GPUINoSuchClass") != 0 {
		t.Error("missing class resolved")
	}
}

func TestObjc_StringRoundTrip(t *testing.T) {
	for _, s := range []string{"", "hello", "zed ✓ ünïcødé 🚀", "nul\x00inside", "\x00"} {
		str := NSStringFromGo(s)
		if got := GoString(str); got != s {
			t.Errorf("GoString(NSStringFromGo(%q)) = %q", s, got)
		}
		Release(str)
	}
	if GoString(Nil) != "" {
		t.Error("GoString(nil) is not empty")
	}
}

func TestObjc_CollectionsRoundTrip(t *testing.T) {
	a, b := NSStringFromGo("a"), NSStringFromGo("b")
	defer Release(a)
	defer Release(b)
	array := NSArrayFromIDs([]ID{a, b})
	defer Release(array)
	got := NSArrayToIDs(array)
	if len(got) != 2 || GoString(got[0]) != "a" || GoString(got[1]) != "b" {
		t.Fatalf("NSArrayToIDs = %v", got)
	}
	dict := NSDictionaryFromIDs([]ID{a}, []ID{b})
	defer Release(dict)
	if v := NSDictionaryObjectForKey(dict, a); GoString(v) != "b" {
		t.Fatalf("objectForKey = %q", GoString(v))
	}
	if v := NSDictionaryObjectForKey(dict, b); v != Nil {
		t.Fatal("missing key resolved")
	}
	data := NSDataFromBytes([]byte{1, 2, 3, 0, 255})
	defer Release(data)
	if got := NSDataToBytes(data); !reflect.DeepEqual(got, []byte{1, 2, 3, 0, 255}) {
		t.Fatalf("NSDataToBytes = %v", got)
	}
	empty := NSDataFromBytes(nil)
	defer Release(empty)
	if got := NSDataToBytes(empty); len(got) != 0 {
		t.Fatalf("empty NSData = %v", got)
	}
}

func TestObjc_URL(t *testing.T) {
	u := NSURLFromString("https://zed.dev/a?b=c")
	defer Release(u)
	if got := NSURLAbsoluteString(u); got != "https://zed.dev/a?b=c" {
		t.Fatalf("absoluteString = %q", got)
	}
	f := NSURLFromFilePath("/tmp/gpui file")
	defer Release(f)
	if got := NSURLPath(f); got != "/tmp/gpui file" {
		t.Fatalf("path = %q", got)
	}
}

func TestObjc_Error(t *testing.T) {
	domain := NSStringFromGo("GPUITestDomain")
	defer Release(domain)
	err := Send[ID](Alloc(GetClass("NSError")), Sel("initWithDomain:code:userInfo:"), domain, NSInteger(-7), Nil)
	defer Release(err)
	e := NSErrorFromID(err)
	if e == nil || e.Domain != "GPUITestDomain" || e.Code != -7 || e.LocalizedDescription == "" {
		t.Fatalf("NSErrorFromID = %+v", e)
	}
	if NSErrorFromID(Nil) != nil {
		t.Fatal("NSErrorFromID(nil) is not nil")
	}
}

func TestObjc_StructPassing(t *testing.T) {
	value := SendClass[ID](GetClass("NSValue"), Sel("valueWithRange:"), MakeNSRange(3, 9))
	if got := Send[NSRange](value, Sel("rangeValue")); got.Location != 3 || got.Length != 9 {
		t.Fatalf("rangeValue = %+v", got)
	}
	nsvalue := GetClass("NSValue")
	if !RespondsToSelector(ID(nsvalue), Sel("valueWithRect:")) {
		t.Skip("NSValue has no valueWithRect: on this OS")
	}
	rect := MakeCGRect(1.5, -2, 30, 40.25)
	rv := SendClass[ID](nsvalue, Sel("valueWithRect:"), rect)
	if got := Send[CGRect](rv, Sel("rectValue")); got != rect {
		t.Fatalf("rectValue = %+v, want %+v", got, rect)
	}
	var rectValue func(ID, SEL) CGRect
	BindMsgSend(&rectValue)
	if got := rectValue(rv, Sel("rectValue")); got != rect {
		t.Fatalf("bound rectValue = %+v, want %+v", got, rect)
	}
	pv := SendClass[ID](nsvalue, Sel("valueWithPoint:"), MakeCGPoint(4, 5))
	if got := Send[CGPoint](pv, Sel("pointValue")); got != MakeCGPoint(4, 5) {
		t.Fatalf("pointValue = %+v", got)
	}
	sv := SendClass[ID](nsvalue, Sel("valueWithSize:"), MakeCGSize(6, 7))
	if got := Send[CGSize](sv, Sel("sizeValue")); got != MakeCGSize(6, 7) {
		t.Fatalf("sizeValue = %+v", got)
	}
}

func TestObjc_RetainedAndPool(t *testing.T) {
	AutoreleasePool(func() {
		s := Autorelease(NSStringFromGo("pooled"))
		r := RetainObject(s)
		if GoString(r.ID()) != "pooled" {
			t.Error("retained object lost its contents")
		}
		r.Release()
		r.Release()
	})
	r := Own(NSStringFromGo("owned"))
	r.Release()
	defer func() {
		if recover() == nil {
			t.Error("ID after Release did not panic")
		}
	}()
	_ = r.ID()
}

func TestObjc_RegisterClassAndBlock(t *testing.T) {
	calls := 0
	methods := []MethodDef{{Cmd: Sel("gpuiAdd:"), Fn: func(self ID, _ SEL, n int) int {
		calls++
		return n + 1
	}}}
	c, err := RegisterClass("GPUIObjcTestClass", GetClass("NSObject"), []string{"NSObject"}, nil, methods)
	if err != nil {
		t.Fatal(err)
	}
	again, err := RegisterClass("GPUIObjcTestClass", GetClass("NSObject"), nil, nil, methods)
	if err != nil || again != c {
		t.Fatalf("second RegisterClass = %v, %v", again, err)
	}
	obj := AllocInit(c)
	defer Release(obj)
	if got := Send[int](obj, Sel("gpuiAdd:"), 41); got != 42 || calls != 1 {
		t.Fatalf("gpuiAdd: = %d after %d calls", got, calls)
	}
	block := NewBlock(func(_ Block, x int) int { return x * 2 })
	defer block.Release()
	got, err := InvokeBlock[int](block, 21)
	if err != nil || got != 42 {
		t.Fatalf("InvokeBlock = %d, %v", got, err)
	}
}

func TestObjc_FoundationConstant(t *testing.T) {
	modes := FoundationConstant("NSRunLoopCommonModes")
	if GoString(modes) != "kCFRunLoopCommonModes" {
		t.Fatalf("NSRunLoopCommonModes = %q", GoString(modes))
	}
}
