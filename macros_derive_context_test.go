package gpui_test

import (
	"testing"

	"github.com/megascan/gpui-go"
)

type myCustomContext struct {
	*gpui.App
	window *gpui.Window
}

func TestDeriveContext_DeriveContext(t *testing.T) {
	var _ gpui.AppContext = myCustomContext{}
}
