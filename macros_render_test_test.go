package gpui_test

import (
	"testing"

	"github.com/megascan/gpui-go"
)

type renderTestElement struct{}

func (*renderTestElement) Render(*gpui.Window, *gpui.Context[renderTestElement]) gpui.IntoElement {
	return gpui.Empty{}
}

func TestRenderTest_DeriveRender(t *testing.T) {
	var _ gpui.Render[renderTestElement] = &renderTestElement{}
}
