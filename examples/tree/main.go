package main

import (
	"os"
	"strconv"
	"sync"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Tree renders a div with a deep children hierarchy.
type Tree struct{}

var depth = sync.OnceValue(func() uint64 {
	if v, ok := os.LookupEnv("GPUI_TREE_DEPTH"); ok {
		if d, err := strconv.ParseUint(v, 10, 64); err == nil {
			return d
		}
	}
	return 50
})

var colors = [4]gpui.Hsla{gpui.Red(), gpui.Blue(), gpui.Green(), gpui.Yellow()}

// Render renders the view.
func (*Tree) Render(_ *gpui.Window, _ *gpui.Context[Tree]) gpui.IntoElement {
	d := depth()
	next := 0
	nextDiv := func() *gpui.Div {
		c := colors[next%len(colors)]
		next++
		return gpui.NewDiv().P0p5().Bg(c)
	}
	innermostNode := nextDiv()
	for d > 0 {
		innermostNode = nextDiv().Child(innermostNode)
		d--
	}
	return innermostNode
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Tree] {
			return cx.New(func(*gpui.Context[Tree]) *Tree { return &Tree{} })
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
