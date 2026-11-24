# Elements and styling

This guide shows how to build and style element trees. The code comes from [snippets/elements/elements.go](snippets/elements/elements.go), which compiles as part of the repo.

## Build a layout with divs

`gpui.NewDiv()` is the general-purpose element. Chain style methods on it and add children with `Child`, `Children` or `ChildText`:

```go
func card(title string, body string) *gpui.Div {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap2().
		P(gpui.Px(16)).
		W(gpui.Px(280)).
		RoundedLg().
		Border1().
		BorderColor(gpui.Rgb(0x45475a)).
		Bg(gpui.Rgb(0x1e1e2e)).
		ShadowLg().
		Child(gpui.NewDiv().TextLg().FontWeight(gpui.FontWeightBold).ChildText(title)).
		Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(0xa6adc8)).ChildText(body))
}
```

Layout is flexbox (and CSS grid with `Grid` and `GridCols`), computed by a port of the Taffy layout engine. The method names follow Tailwind: `Flex`, `FlexCol`, `FlexRow`, `ItemsCenter`, `JustifyBetween`, `Gap2`, `P4`, `RoundedMd`. Most scale shortcuts exist (`Size8`, `W64`, `Px2`, `Gap3`), along with versions that take a length (`Size`, `W`, `Px`, `Gap`).

`ChildText` takes a Go `string`. To pass a `gpui.SharedString`, use `Child`, since `SharedString` is an element too.

A row that spreads its children across the full width:

```go
func toolbar() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		FlexRow().
		ItemsCenter().
		JustifyBetween().
		WFull().
		H(gpui.Px(40)).
		Px(gpui.Px(12)).
		Child(gpui.NewDiv().ChildText("Left")).
		Child(gpui.NewDiv().ChildText("Right"))
}
```

`SizeFull`, `WFull` and `HFull` mean 100% of the parent. They do nothing useful when the parent has no size of its own, so give some ancestor a definite size (the window root is sized by the window).

## Lengths

Length arguments are typed, so untyped numbers do not compile:

```go
func lengths() *gpui.Div {
	return gpui.NewDiv().
		W(gpui.Relative(0.5)).
		H(gpui.Rems(2)).
		MinW(gpui.Px(120)).
		Mx(gpui.Auto())
}
```

- `gpui.Px(n)` is logical pixels.
- `gpui.Rems(n)` is a multiple of the window's rem size.
- `gpui.Relative(f)` is a fraction of the parent.
- `gpui.Auto()` is `auto`. Padding and gap take a definite length, so they reject `Auto()`.

The name `Px` appears twice. `gpui.Px(12)` builds a length; `.Px(...)` on an element sets horizontal padding.

## Colors and backgrounds

```go
func colors() *gpui.Div {
	accent := gpui.NewHsla(0.6, 0.8, 0.6, 1)
	return gpui.NewDiv().
		Bg(gpui.Rgb(0x313244)).
		TextColor(gpui.White()).
		BorderColor(accent.Opacity(0.5)).
		Child(gpui.NewDiv().Size8().Bg(gpui.NewRgba(0xff000080))).
		Child(gpui.NewDiv().Size8().Bg(gpui.LinearGradient(
			90,
			gpui.NewLinearColorStop(gpui.Red(), 0),
			gpui.NewLinearColorStop(gpui.Blue(), 1),
		)))
}
```

`gpui.Rgb(0xRRGGBB)` and `gpui.NewRgba(0xRRGGBBAA)` take hex values. `gpui.NewHsla(h, s, l, a)` takes values from 0 to 1. `Bg` also accepts gradients and patterns; see [examples/gradient](../../examples/gradient/main.go) and [examples/pattern](../../examples/pattern/main.go).

## Hover and active styles

`Hover`, `Active`, `Focus` and `FocusVisible` take a function that edits a `*gpui.StyleRefinement`. That type has no chainable style methods of its own, so bind it to a `gpui.Styled` first. A small helper does this once:

```go
func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func hoverBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x45475a))
}

func activeBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x585b70))
}
```

Then use it on a button:

```go
func button(id string, label string, onClick func(event *gpui.ClickEvent, window *gpui.Window, cx *gpui.App)) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Px3().
		Py1().
		RoundedMd().
		Bg(gpui.Rgb(0x313244)).
		Hover(hoverBg).
		Active(activeBg).
		CursorPointer().
		OnClick(onClick).
		ChildText(label)
}
```

`Hover` works on any element. `Active`, `OnClick`, `Tooltip`, `OnDrag` and scrolling need an id, so they only exist after `ID`.

## Conditional styling

`When` applies a function only if a condition is true. `WhenSome` applies it only if an `Option` has a value:

```go
func row(selected bool, subtitle gpui.Option[string]) *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap2().
		When(selected, func(this *gpui.Div) *gpui.Div {
			return this.Bg(gpui.Rgb(0x45475a))
		}).
		WhenSome(subtitle, func(this *gpui.Div, text string) *gpui.Div {
			return this.Child(gpui.NewDiv().TextXs().ChildText(text))
		})
}
```

The closure type follows the element's current type. After `ID`, the element is a `*gpui.Stateful[*gpui.Div]`, and so is the closure's parameter:

```go
func statefulRow(ix int, selected bool) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		Px2().
		ID(ix).
		When(selected, func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
			return this.Bg(gpui.Rgb(0x45475a))
		})
}
```

`ID` accepts strings, integers and `gpui.ElementID` values. For ids built from a name and a number, use `gpui.ElementIDNamedInteger("item", uint64(ix))`. GPUI combines an element's id with the ids of its ancestors, so an id only has to be unique under its nearest ancestor with an id. Keep ids stable across frames: state such as hover, click and scroll position is stored under them.

## Reusable components

A view is an entity with state. For a stateless piece of UI, write a *component*: a value type with a `Render(window, cx *gpui.App)` method, plus an `IntoAnyElement` method that wraps it with `gpui.NewComponent`:

```go
type Badge struct {
	Label string
	Color gpui.Hsla
}

// IntoAnyElement renders the badge as a component.
func (b Badge) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(b)
}

// Render renders the badge.
func (b Badge) Render(_ *gpui.Window, _ *gpui.App) gpui.IntoElement {
	return gpui.NewDiv().
		Px2().
		RoundedFull().
		TextXs().
		Bg(b.Color.Opacity(0.2)).
		TextColor(b.Color).
		ChildText(b.Label)
}
```

A `Badge` value can now go anywhere an element can:

```go
func badges() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap1().
		Child(Badge{Label: "new", Color: gpui.Green()}).
		Child(Badge{Label: "beta", Color: gpui.Yellow()})
}
```

This replaces Rust GPUI's `#[derive(IntoElement)]` on a `RenderOnce` type. Plain functions that return `*gpui.Div` (like `card` above) are often enough too.

To show a child view, pass its entity to `Child`. That moves the handle into the element tree, so pass a clone if you keep the handle. See [Concepts](concepts.md#views-inside-views).

## Images and SVG

`Svg.Path` and image paths given to `gpui.ImageSourceFromString` load through the app's asset source. Give the app one with `WithAssets`. This one serves files embedded in the binary:

```go
//go:embed assets
var assetFS embed.FS

// Assets serves embedded files to Svg.Path and image asset paths.
type Assets struct{}

// Load returns the bytes of the asset at path.
func (Assets) Load(path string) ([]byte, bool, error) {
	data, err := fs.ReadFile(assetFS, path)
	if err != nil {
		return nil, false, fmt.Errorf("load asset %s: %w", path, err)
	}
	return data, true, nil
}

// List returns the names of the assets under path.
func (Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := fs.ReadDir(assetFS, path)
	if err != nil {
		return nil, fmt.Errorf("list assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(entry.Name()))
	}
	return names, nil
}
```

Register it before `Run`, as [examples/svg_example](../../examples/svg_example/main.go) does:

```go
	platform.Application().
		WithAssets(Assets{base: examplesDir()}).
		Run(func(cx *gpui.App) {
```

Then draw:

```go
func media() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap2().
		Child(gpui.NewSvg().Path("assets/check.svg").Size6().TextColor(gpui.Green())).
		Child(gpui.NewImg(gpui.ImageSourceFromPath("/usr/share/pixmaps/example.png")).Size16()).
		Child(gpui.NewImg(gpui.ImageSourceFromString("https://example.com/avatar.png")).Size16().RoundedFull())
}
```

- `NewSvg().Path(...)` reads through the asset source. `TextColor` sets the fill color. Text and embedded images inside SVG files are not drawn (limit L-SVGTEXT).
- `ImageSourceFromPath` reads the file system directly and needs no asset source.
- `ImageSourceFromString` treats a value with a URL scheme (`https:`, `file:`, ...) as a URL and everything else as an asset path.

On an `Img`, call the image methods (`ObjectFit`, `Grayscale`, `WithLoading`, `WithFallback`) before `ID`. See [examples/image_example](../../examples/image_example/main.go) and [examples/image_loading](../../examples/image_loading/main.go) for loading states and fallbacks.

## Animation

Animations are free functions that wrap an element:

```go
func spinner() gpui.IntoElement {
	return gpui.WithAnimation(
		gpui.NewSvg().Path("assets/check.svg").Size6().TextColor(gpui.White()),
		"spinner",
		gpui.NewAnimation(time.Second).Repeat(),
		func(svg *gpui.Svg, delta float32) *gpui.Svg {
			return svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))
		},
	)
}
```

The callback receives `delta` from 0 to 1 and returns the element styled for that point. Add easing with `.WithEasing(gpui.EaseInOut)` and friends. [examples/animation](../../examples/animation/main.go) has more.

## Popovers and overlays

`gpui.NewDeferred` paints its child after everything else, and `gpui.NewAnchored` positions it relative to a point and keeps it inside the window. Together they make popovers and menus. From [examples/anchor](../../examples/anchor/main.go):

```go
				return this.Child(gpui.NewDeferred(
					gpui.NewAnchored().
						Anchor(corner).
						Position(position).
						PositionMode(gpui.AnchoredPositionModeLocal).
						SnapToWindow().
```

## Custom painting

`gpui.NewCanvas` takes a prepaint function and a paint function that draw straight into the window. See [examples/gradient](../../examples/gradient/main.go) and [examples/painting](../../examples/painting/main.go).

## Debugging layout

Call `.Debug()` on an element to draw a debug border around it. In tests, `DebugSelector` names an element so `VisualTestContext.DebugBounds` can find it (see [Testing](testing.md#click-an-element)).

## More examples

| Topic | Example |
|---|---|
| Grid layout | [examples/grid_layout](../../examples/grid_layout/main.go) |
| Text styles and wrapping | [examples/text](../../examples/text/main.go), [examples/text_wrapper](../../examples/text_wrapper/main.go) |
| Shadows | [examples/shadow](../../examples/shadow/main.go) |
| Opacity | [examples/opacity](../../examples/opacity/main.go) |
| Drag and drop | [examples/drag_drop](../../examples/drag_drop/main.go) |
| Popovers | [examples/popover](../../examples/popover/main.go) |
