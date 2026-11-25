package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// DragInfo is the dragged value and its drag preview view.
type DragInfo struct {
	ix       int
	color    gpui.Hsla
	position gpui.Point[gpui.Pixels]
}

// NewDragInfo creates a DragInfo at the origin.
func NewDragInfo(ix int, color gpui.Hsla) DragInfo {
	return DragInfo{
		ix:       ix,
		color:    color,
		position: gpui.Point[gpui.Pixels]{},
	}
}

// Position returns d moved to pos.
func (d DragInfo) Position(pos gpui.Point[gpui.Pixels]) DragInfo {
	d.position = pos
	return d
}

// Render renders the drag preview.
func (d *DragInfo) Render(_ *gpui.Window, _ *gpui.Context[DragInfo]) gpui.IntoElement {
	size := gpui.NewSize(gpui.Px(120.), gpui.Px(50.))

	return gpui.NewDiv().
		Pl(d.position.X - size.Width.Half()).
		Pt(d.position.Y - size.Height.Half()).
		Child(
			gpui.NewDiv().
				Flex().
				JustifyCenter().
				ItemsCenter().
				W(size.Width).
				H(size.Height).
				Bg(d.color.Opacity(0.5)).
				TextColor(gpui.White()).
				TextXs().
				ShadowMd().
				ChildText(fmt.Sprintf("Item %d", d.ix)),
		)
}

// DragDrop is the drag_drop example view.
type DragDrop struct {
	dropOn gpui.Option[DragInfo]
}

// NewDragDrop creates the view.
func NewDragDrop() *DragDrop {
	return &DragDrop{dropOn: gpui.None[DragInfo]()}
}

// Render renders the view.
func (d *DragDrop) Render(_ *gpui.Window, cx *gpui.Context[DragDrop]) gpui.IntoElement {
	items := []gpui.Hsla{gpui.Blue(), gpui.Red(), gpui.Green()}

	children := make([]gpui.IntoElement, 0, len(items))
	for ix, color := range items {
		dragInfo := NewDragInfo(ix, color)

		children = append(children, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("item", uint64(ix))).
			Size32().
			Flex().
			JustifyCenter().
			ItemsCenter().
			Border2().
			BorderColor(color).
			TextColor(color).
			CursorMove().
			Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement {
				var s gpui.Styled[*gpui.StyleRefinement]
				gpui.BindStyled(&s, this)
				return s.Bg(color.Opacity(0.2))
			}).
			ChildText(fmt.Sprintf("Item (%d)", ix)).
			OnDrag(dragInfo, func(info DragInfo, position gpui.Point[gpui.Pixels], _ *gpui.Window, cx *gpui.App) gpui.Entity[DragInfo] {
				return cx.New(func(_ *gpui.Context[DragInfo]) *DragInfo {
					moved := info.Position(position)
					return &moved
				})
			}))
	}

	borderColor := gpui.Black()
	if info, ok := d.dropOn.Get(); ok {
		borderColor = info.color
	}
	onDrop := cx.Listener(func(this *DragDrop, info *DragInfo, _ *gpui.Window, _ *gpui.Context[DragDrop]) {
		this.dropOn = gpui.Some(*info)
	})

	return gpui.NewDiv().
		SizeFull().
		Flex().
		FlexCol().
		Gap5().
		Bg(gpui.White()).
		JustifyCenter().
		ItemsCenter().
		TextColor(gpui.Rgb(0x333333)).
		Child(gpui.NewDiv().TextXl().TextCenter().ChildText("Drop & Drop")).
		Child(
			gpui.NewDiv().
				WFull().
				Mb10().
				JustifyCenter().
				Flex().
				FlexRow().
				Gap4().
				ItemsCenter().
				Children(children...),
		).
		Child(
			gpui.NewDiv().
				ID("drop-target").
				W128().
				H32().
				Flex().
				JustifyCenter().
				ItemsCenter().
				Border3().
				BorderColor(borderColor).
				WhenSome(d.dropOn, func(this *gpui.Stateful[*gpui.Div], info DragInfo) *gpui.Stateful[*gpui.Div] {
					return this.Bg(info.color.Opacity(0.5))
				}).
				OnDrop(func(info DragInfo, window *gpui.Window, cx *gpui.App) {
					onDrop(&info, window, cx)
				}).
				ChildText("Drop items here"),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(800.), gpui.Px(600.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[DragDrop] {
			return cx.New(func(_ *gpui.Context[DragDrop]) *DragDrop { return NewDragDrop() })
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	runExample()
}
