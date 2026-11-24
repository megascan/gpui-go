//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

var testScreenDeviceDescription objc.ID

func testScreenDeviceDescriptionMethod(_ objc.ID, _ objc.SEL) objc.ID {
	return testScreenDeviceDescription
}

func TestWindow_DisplayIDForScreenHandlesMissingMetadata(t *testing.T) {
	if _, ok := displayIDForScreen(objc.Nil); ok {
		t.Fatal("expected no display id for a nil screen")
	}
	objc.AutoreleasePool(func() {
		class, err := objc.RegisterClass("GPUIDisplayIdTestScreen", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{Cmd: objc.Sel("deviceDescription"), Fn: testScreenDeviceDescriptionMethod},
		})
		if err != nil {
			t.Fatalf("test screen class: %v", err)
		}
		screen := objc.SendClass[objc.ID](class, objc.Sel("new"))
		defer objc.Release(screen)

		testScreenDeviceDescription = objc.Nil
		if _, ok := displayIDForScreen(screen); ok {
			t.Fatal("expected no display id without a device description")
		}

		dictionary := objc.SendClass[objc.ID](objc.GetClass("NSMutableDictionary"), objc.Sel("new"))
		defer objc.Release(dictionary)
		testScreenDeviceDescription = dictionary
		if _, ok := displayIDForScreen(screen); ok {
			t.Fatal("expected no display id without NSScreenNumber")
		}

		for _, displayID := range []uint32{0, 42} {
			number := objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithUnsignedInt:"), displayID)
			objc.SendVoid(dictionary, objc.Sel("setObject:forKey:"), number, nsString("NSScreenNumber"))
			got, ok := displayIDForScreen(screen)
			if !ok || got != displayID {
				t.Fatalf("displayIDForScreen = (%d, %v), want (%d, true)", got, ok, displayID)
			}
		}
	})
}
