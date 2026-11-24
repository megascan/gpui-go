//go:build ios

package ios

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
)

func init() {
	runtime.LockOSThread()
}

func handlerQueries() int { return windowReferenceQueries }

func TestMain(m *testing.M) {
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("native test runs on main thread")
	}
	if err := uikit.Load(); err != nil {
		panic(err)
	}
	objc.AutoreleasePool(func() {
		nativeProtocolSmokeTest()
		staleObjectsDoNotRouteToHandler(handlerQueries)
	})
	fmt.Println("UIKit text input native protocol smoke test passed")
	os.Exit(m.Run())
}
