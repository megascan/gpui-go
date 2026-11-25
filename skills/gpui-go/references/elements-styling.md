# Elements and styling

## Contents
- Div, mixins and the receiver-type rule
- Units, colors and text
- Hover, active and focus styles
- Conditional styling and children
- Components (RenderOnce)
- Composing views
- Scrolling, uniform_list and list
- Images, SVG and assets
- Canvas
- Popovers (deferred + anchored)
- Tooltips
- Animation

## Div, mixins and the receiver-type rule

`gpui.NewDiv()` returns `*gpui.Div`, which embeds these mixins:
- `Styled`: the 3000+ Tailwind-style methods in `styled.go` and `styled_gen.go`
- `Interactive`: mouse and key listeners, `Hover`, `KeyContext`, `OnAction`, `TrackFocus`, `DebugSelector`, `OnDrop`
- `InteractiveID`: `ID`
- `Parent`: `Child`, `Children`, `ChildText`
- `Fluent`: `When`, `WhenElse`, `WhenSome`, `WhenNone`, `Map`

Every method returns the receiver type, so chains keep their type. `.ID(x)` returns `*gpui.Stateful[*gpui.Div]`, which adds `StatefulInteractive`: `OnClick`, `OnAuxClick`, `Active`, `OnHover`, `Tooltip`, `OnDrag`, `OverflowScroll`, `OverflowYScroll`, `TrackScroll` and `Focusable`. A helper that returns `*gpui.Div` therefore can't be followed by `OnClick` until something calls `.ID(...)`.

- `ID` accepts the `gpui.ElementIDSource` set: `~string`, `~int`, `~int32`, `~uint`, `~uint32`, `~uint64`, `gpui.ElementID`, `*gpui.FocusHandle`, `*gpui.SourceLocation`, `[16]byte` (UUID) and `[20]byte` (opaque id). Other int kinds (`int64`, `uint8`, ...) need a conversion. For composite ids use `gpui.ElementIDNamedInteger("row", uint64(ix))`.
- IDs must be stable between frames and unique among siblings. Scroll offsets, hover and active state, and animations are all keyed by them.
- Tailwind scale: `P2`, `Px4`, `Gap3`, `Size8`, `W64`, `H32`, `RoundedMd`, `TextSm`, `ShadowLg`, `Border1`, `BorderB1`, ... Fractions keep the underscore (`W1_2`).
- Layout: `Flex`, `FlexCol`, `FlexRow`, `Flex1`, `FlexGrow`, `FlexShrink0`, `FlexWrap`, `ItemsCenter`, `JustifyCenter`, `JustifyBetween`, `Grid`, `GridCols(n)`, `Relative`, `Absolute`, `Top`/`Left`/`Inset0`, `OverflowHidden`, `SizeFull`, `WFull`, `HFull`.
- `SizeFull` only works when the parent has a size. A window root is sized, but each nested level must be sized too (`Flex1` inside a flex parent also works).

## Units, colors and text

```go
func Units() *gpui.Div {
	return gpui.NewDiv().
		W(gpui.Px(320)).
		H(gpui.Rems(2.5)).
		MaxW(gpui.Relative(0.5)).
		Mx(gpui.Auto()).
		Gap(gpui.Px(8)).
		Bg(gpui.NewRgba(0x1e1e2eff)).
		BorderColor(gpui.NewHsla(0.6, 0.5, 0.5, 1))
}
```

- `gpui.Px(f)` makes `Pixels`, `gpui.Rems(f)` is a type conversion, `gpui.Relative(f)` is a fraction of the parent, and `gpui.Auto()` is a `Length`. Untyped constants are rejected.
- `W`, `H`, `Size`, `M*` and `Top` take `IntoLength`. `P*` and `Gap` take `IntoDefiniteLength`, which `Auto()` doesn't satisfy. `Rounded` and `Border` take `IntoAbsoluteLength`.
- Colors: `gpui.Rgb(0xRRGGBB)`, `gpui.NewRgba(0xRRGGBBAA)`, `gpui.NewHsla(h, s, l, a)`, `gpui.ParseRgba("#...")`. Named colors are `gpui.White()`, `gpui.Black()`, `gpui.Red()`, `gpui.Green()`, `gpui.Blue()`, `gpui.Yellow()`, `gpui.TransparentBlack()`. `hsla.Opacity(f)` scales alpha.
- `Bg` takes `IntoFill`: a color, or a `gpui.Background` such as `gpui.LinearGradient(...)`, `gpui.PatternSlash(...)` or `gpui.SolidBackground(c)`.
- Text: `TextColor`, `TextSize`, `TextXs`...`Text3xl`, `FontWeight`, `FontFamily`, `LineHeight`, `Italic`, `Underline`, `LineThrough`, `Truncate`, `TextEllipsis`, `LineClamp(n)`, `TextCenter`, `WhitespaceNowrap`.
- `ChildText(string)` takes Go strings. A `gpui.SharedString` is itself an element, so use `Child(s)`. Keep `SharedString` in state for text you render often.

## Hover, active and focus styles

`Hover`, `Active`, `Focus`, `InFocus`, `FocusVisible` and `GroupHover` callbacks receive a bare `*gpui.StyleRefinement`. Bind a `Styled` mixin to get the fluent methods. Its methods return `*gpui.StyleRefinement`, so call them one after another rather than chaining:

```go
func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func Button(id string, label string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Px4().
		Py2().
		RoundedMd().
		Bg(gpui.Rgb(0x313244)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		CursorPointer().
		Hover(func(s *gpui.StyleRefinement) *gpui.StyleRefinement {
			return styled(s).Bg(gpui.Rgb(0x45475a))
		}).
		Active(func(s *gpui.StyleRefinement) *gpui.StyleRefinement {
			return styled(s).Opacity(0.8)
		}).
		ChildText(label)
}
```

For two properties, write `st := styled(s); st.Border2(); return st.BorderColor(c)`. Setting `Hover` twice on one element panics while debug assertions are on (the default build).

## Conditional styling and children

```go
func (p *Panel) Render(_ *gpui.Window, cx *gpui.Context[Panel]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap2().
		P(gpui.Px(12)).
		SizeFull().
		Child(p.title).
		Child(Badge{Label: "beta", Color: gpui.Blue()}).
		When(p.expanded, func(d *gpui.Div) *gpui.Div {
			return d.Border1().BorderColor(gpui.Rgb(0x585b70))
		}).
		WhenSome(p.error, func(d *gpui.Div, msg string) *gpui.Div {
			return d.Child(gpui.NewDiv().TextColor(gpui.Red()).ChildText(msg))
		}).
		Child(
			Button("toggle", "Toggle").
				OnClick(cx.Listener(func(this *Panel, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Panel]) {
					this.expanded = !this.expanded
					cx.Notify()
				})),
		).
		Child(
			gpui.NewDiv().
				ID("log").
				Flex1().
				OverflowYScroll().
				Children(logLines(p.log)...),
		)
}

func logLines(lines []string) []gpui.IntoElement {
	out := make([]gpui.IntoElement, 0, len(lines))
	for _, line := range lines {
		out = append(out, gpui.NewDiv().TextSm().ChildText(line))
	}
	return out
}
```

- A `When` closure's parameter and result types match the chain at that point: `*gpui.Div` before `ID`, `*gpui.Stateful[*gpui.Div]` after it.
- `gpui.Option[T]` comes from `gpui.Some(v)`/`gpui.None[T]()` and is read with `o.Get()`.
- `Children` takes `...gpui.IntoElement`, so build a `[]gpui.IntoElement`. A `[]*gpui.Div` won't convert.

## Components (RenderOnce)

The replacement for `#[derive(IntoElement)]` is a value type with `Render(*gpui.Window, *gpui.App)` plus `IntoAnyElement` that calls `gpui.NewComponent`:

```go
type Badge struct {
	Label string
	Color gpui.Hsla
}

func (b Badge) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(b)
}

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

## Composing views

A child view is an `Entity`. Create it once, store it, release it in `Drop`, and pass a clone to `Child`, which consumes the handle:

```go
type Shell struct {
	panel gpui.Entity[Panel]
}

func (s *Shell) Drop(_ *gpui.App) {
	s.panel.Release()
}

func (s *Shell) Render(_ *gpui.Window, _ *gpui.Context[Shell]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().Child(s.panel.Clone())
}
```

Don't create entities inside `Render`, because that makes a new entity every frame. For per-element state, use `window.UseKeyedState(id, cx, init)` and release what it returns.

## Scrolling, uniform_list and list

- Scrollable div: `.ID("x").OverflowYScroll()` in a parent with a bounded height. Add `.TrackScroll(handle)` with `gpui.NewScrollHandle()` to read or set the offset.
- `gpui.NewUniformList(id, count, func(visible gpui.Range, window *gpui.Window, cx *gpui.App) []R)` renders only the visible rows. Every row has the height of row 0. It needs a bounded height (`Flex1` in a flex column, `HFull`, or `H(...)`), or it lays out everything. Scroll it from code with `gpui.NewUniformListScrollHandle()` + `.TrackUniformListScroll(h)` + `h.ScrollToItem(ix, gpui.ScrollStrategyNearest)` (also `ScrollStrategyTop`, `ScrollStrategyCenter`, `ScrollStrategyBottom`).
- In the row callback, read a snapshot of the state taken in `Render`. Don't use `cx.Processor`, which keeps a strong handle alive and fails strict leak checks. The keyboard todo-list example (in the actions and focus reference) shows the full pattern.
- For variable-height rows, use `gpui.NewList(state, func(ix int, window *gpui.Window, cx *gpui.App) gpui.AnyElement)` with a `gpui.ListState`, which is a value whose copies share state:

```go
type Feed struct {
	state gpui.ListState
	posts []string
}

func NewFeed(posts []string) *Feed {
	return &Feed{posts: posts, state: gpui.NewListState(len(posts), gpui.ListAlignmentTop, gpui.Px(200))}
}

func (f *Feed) Render(_ *gpui.Window, _ *gpui.Context[Feed]) gpui.IntoElement {
	posts := f.posts
	return gpui.NewDiv().SizeFull().Flex().FlexCol().Child(
		gpui.NewList(f.state, func(ix int, _ *gpui.Window, _ *gpui.App) gpui.AnyElement {
			return gpui.NewDiv().P(gpui.Px(8)).ChildText(posts[ix]).IntoAnyElement()
		}).Flex1(),
	)
}

func (f *Feed) Append(post string, cx *gpui.Context[Feed]) {
	f.state.Splice(gpui.Range{Start: len(f.posts), End: len(f.posts)}, 1)
	f.posts = append(f.posts, post)
	f.state.ScrollToEnd()
	cx.Notify()
}
```

`ListAlignmentBottom` gives a chat-style list. Other `ListState` methods are `Reset(n)`, `ScrollToRevealItem(ix)` and `MeasureAll()`.

## Images, SVG and assets

```go
func Media() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap2().
		Child(gpui.NewSvg().Path("icons/check.svg").Size4().TextColor(gpui.Green())).
		Child(gpui.NewImg(gpui.ImageSourceFromPath("/tmp/photo.png")).W(gpui.Px(64)).H(gpui.Px(64))).
		Child(gpui.NewImg(gpui.ImageSourceFromString("https://example.com/a.png")).Size8())
}
```

- `Svg.Path` and asset-path image strings resolve through the app's `AssetSource`, so without `platform.Application().WithAssets(src)` nothing renders. `ImageSourceFromPath` reads the filesystem directly.
- SVG color comes from `TextColor`. Text and embedded images inside SVGs aren't rendered (L-SVGTEXT).
- Call image style methods before `ID`, because `Stateful[*Img]` doesn't have them.
- A file-backed `AssetSource` (from examples/svg_example):

```go
type Assets struct {
	base string
}

func (a Assets) Load(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(filepath.Join(a.base, path))
	if err != nil {
		return nil, false, fmt.Errorf("loading asset %s: %w", path, err)
	}
	return data, true, nil
}

func (a Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := os.ReadDir(filepath.Join(a.base, path))
	if err != nil {
		return nil, fmt.Errorf("listing assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(entry.Name()))
	}
	return names, nil
}
```

## Canvas

`gpui.NewCanvas(prepaint, paint)` gives you raw painting inside layout bounds:

```go
func Swatch(color gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().Size16().Child(gpui.NewCanvas(
		func(gpui.Bounds[gpui.Pixels], *gpui.Window, *gpui.App) struct{} { return struct{}{} },
		func(bounds gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, _ *gpui.App) {
			window.PaintQuad(gpui.NewFill(bounds, gpui.SolidBackground(color)))
		},
	).SizeFull())
}
```

For paths, use `gpui.PathBuilderFill()` with `MoveTo`, `LineTo` and `Build()`, then `window.PaintPath(path, background)` (examples/gradient, examples/painting).

## Popovers (deferred + anchored)

```go
func Popover(open bool, position gpui.Point[gpui.Pixels]) *gpui.Div {
	return gpui.NewDiv().Relative().When(open, func(d *gpui.Div) *gpui.Div {
		return d.Child(gpui.NewDeferred(
			gpui.NewAnchored().
				Anchor(gpui.AnchorTopLeft).
				Position(position).
				SnapToWindow().
				Child(gpui.NewDiv().P(gpui.Px(8)).Bg(gpui.White()).ShadowLg().ChildText("menu")),
		).WithPriority(1))
	})
}
```

`NewDeferred` paints the popover after its siblings. `NewAnchored` positions it and keeps it inside the window. To close on an outside click, add `OnMouseDownOut` on the popover content.

## Tooltips

The builder returns an `AnyView`, which the window takes over, so building a new entity each time is correct:

```go
type TooltipView struct {
	text string
}

func (t *TooltipView) Render(_ *gpui.Window, _ *gpui.Context[TooltipView]) gpui.IntoElement {
	return gpui.NewDiv().P(gpui.Px(4)).Bg(gpui.Black()).TextColor(gpui.White()).ChildText(t.text)
}

func WithTooltip(text string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID("help").
		ChildText("?").
		Tooltip(func(_ *gpui.Window, cx *gpui.App) gpui.AnyView {
			return gpui.AnyViewFromEntity(cx.New(func(_ *gpui.Context[TooltipView]) *TooltipView {
				return &TooltipView{text: text}
			}))
		})
}
```

## Animation

Animations are free functions, not methods (D-ANIMEXT):

```go
func Spinner() gpui.IntoElement {
	return gpui.WithAnimation(
		gpui.NewDiv().Size4().RoundedFull().Bg(gpui.Blue()),
		"spinner",
		gpui.NewAnimation(time.Second).Repeat().WithEasing(gpui.PulsatingBetween(0.3, 1)),
		func(d *gpui.Div, delta float32) *gpui.Div { return d.Opacity(delta) },
	)
}
```

Easings: `gpui.Linear`, `gpui.Quadratic`, `gpui.EaseInOut`, `gpui.EaseOutQuint()`, `gpui.Bounce(easing)`, `gpui.PulsatingBetween(min, max)`. Also `gpui.WithAnimations` (a sequence) and `gpui.WithSpring`. For SVGs, use `svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))`.
