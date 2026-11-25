package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

const totalItems = 10000

const (
	scrollbarThumbWidth  gpui.Pixels = 8.
	scrollbarThumbHeight gpui.Pixels = 100.
)

// Quote is one randomly generated row of market data.
type Quote struct {
	name             gpui.SharedString
	symbol           gpui.SharedString
	lastDone         float64
	prevClose        float64
	open             float64
	high             float64
	low              float64
	timestamp        time.Duration
	volume           int64
	turnover         float64
	ttm              float64
	marketCap        float64
	floatCap         float64
	shares           float64
	pb               float64
	pe               float64
	eps              float64
	dividend         float64
	dividendYield    float64
	dividendPerShare float64
	dividendDate     gpui.SharedString
	dividendPayment  float64
}

func randFloat(lo, hi float64) float64 {
	return lo + rand.Float64()*(hi-lo)
}

func randInt(lo, hi int64) int64 {
	return lo + rand.Int64N(hi-lo)
}

func randLetter() byte {
	return byte('A' + rand.IntN('Z'-'A'+1))
}

// RandomQuote generates a random Quote.
func RandomQuote() *Quote {
	prevClose := randFloat(100.0, 200.0)
	change := randFloat(-5.0, 5.0)
	lastDone := prevClose + change
	open := prevClose + randFloat(-3.0, 3.0)
	high := max(prevClose+randFloat(0.0, 10.0), open)
	low := min(prevClose-randFloat(0.0, 10.0), open)
	timestamp := time.Duration(randInt(0, 86400)) * time.Second
	volume := randInt(1_000_000, 100_000_000)
	turnover := lastDone * float64(volume)
	var ticker strings.Builder
	if rand.IntN(2) == 0 {
		fmt.Fprintf(&ticker, "%03d.%d", randInt(100, 1000), randInt(0, 10))
	} else {
		fmt.Fprintf(&ticker, "%c%c", randLetter(), randLetter())
	}
	fmt.Fprintf(&ticker, ".%c", randLetter())
	symbol := ticker.String()
	name := fmt.Sprintf("%s %d - #%d", symbol, randInt(1, 100), randInt(10000, 100000))
	ttm := randFloat(0.0, 10.0)
	marketCap := randFloat(1_000_000.0, 10_000_000.0)
	floatCap := marketCap + randFloat(1_000.0, 10_000.0)
	shares := randFloat(100.0, 1000.0)
	pb := marketCap / shares
	pe := marketCap / shares
	eps := marketCap / shares
	dividend := randFloat(0.0, 10.0)
	dividendYield := randFloat(0.0, 10.0)
	dividendPerShare := randFloat(0.0, 10.0)
	dividendDate := gpui.SharedString(fmt.Sprintf("%d-%d-%d", randInt(2000, 2023), randInt(1, 12), randInt(1, 28)))
	dividendPayment := randFloat(0.0, 10.0)

	return &Quote{
		name:             gpui.SharedString(name),
		symbol:           gpui.SharedString(symbol),
		lastDone:         lastDone,
		prevClose:        prevClose,
		open:             open,
		high:             high,
		low:              low,
		timestamp:        timestamp,
		volume:           volume,
		turnover:         turnover,
		pb:               pb,
		pe:               pe,
		eps:              eps,
		ttm:              ttm,
		marketCap:        marketCap,
		floatCap:         floatCap,
		shares:           shares,
		dividend:         dividend,
		dividendYield:    dividendYield,
		dividendPerShare: dividendPerShare,
		dividendDate:     dividendDate,
		dividendPayment:  dividendPayment,
	}
}

func (q *Quote) change() float64 {
	return (q.lastDone - q.prevClose) / q.prevClose * 100.0
}

func (q *Quote) changeColor() gpui.Hsla {
	if q.change() > 0.0 {
		return gpui.Green()
	}
	return gpui.Red()
}

func (q *Quote) turnoverRatio() float64 {
	return float64(q.volume) / q.turnover * 100.0
}

// TableRow is one rendered row of the table.
type TableRow struct {
	ix    int
	quote *Quote
}

// NewTableRow creates a TableRow.
func NewTableRow(ix int, quote *Quote) TableRow {
	return TableRow{ix: ix, quote: quote}
}

// IntoAnyElement renders the row as a component.
func (r TableRow) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(r)
}

func (r TableRow) renderCell(key string, width gpui.Pixels, color gpui.Hsla) gpui.IntoElement {
	q := r.quote
	var cell *gpui.Div
	switch key {
	case "id":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%d", r.ix))
	case "symbol":
		cell = gpui.NewDiv().Child(q.symbol)
	case "name":
		cell = gpui.NewDiv().Child(q.name)
	case "last_done":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.3f", q.lastDone))
	case "prev_close":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.3f", q.prevClose))
	case "change":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f%%", q.change()))
	case "timestamp":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%d", uint64(q.timestamp/time.Second)))
	case "open":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f", q.open))
	case "low":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f", q.low))
	case "high":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f", q.high))
	case "ttm":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f", q.ttm))
	case "eps":
		cell = gpui.NewDiv().TextColor(color).ChildText(fmt.Sprintf("%.2f", q.eps))
	case "market_cap":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f M", q.marketCap/1_000_000.0))
	case "float_cap":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f M", q.floatCap/1_000_000.0))
	case "turnover":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f M", q.turnover/1_000_000.0))
	case "volume":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f M", float64(q.volume)/1_000_000.0))
	case "turnover_ratio":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f%%", q.turnoverRatio()))
	case "pe":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.pe))
	case "pb":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.pb))
	case "shares":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.shares))
	case "dividend":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.dividend))
	case "yield":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f%%", q.dividendYield))
	case "dividend_per_share":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.dividendPerShare))
	case "dividend_date":
		cell = gpui.NewDiv().ChildText(string(q.dividendDate))
	case "dividend_payment":
		cell = gpui.NewDiv().ChildText(fmt.Sprintf("%.2f", q.dividendPayment))
	default:
		cell = gpui.NewDiv().ChildText("--")
	}
	return gpui.NewDiv().
		WhitespaceNowrap().
		Truncate().
		W(width).
		Px1().
		Child(cell)
}

type field struct {
	key   string
	width float32
}

var fields = [24]field{
	{"id", 64.},
	{"symbol", 64.},
	{"name", 180.},
	{"last_done", 80.},
	{"prev_close", 80.},
	{"open", 80.},
	{"low", 80.},
	{"high", 80.},
	{"ttm", 50.},
	{"market_cap", 96.},
	{"float_cap", 96.},
	{"turnover", 120.},
	{"volume", 100.},
	{"turnover_ratio", 96.},
	{"pe", 64.},
	{"pb", 64.},
	{"eps", 64.},
	{"shares", 96.},
	{"dividend", 64.},
	{"yield", 64.},
	{"dividend_per_share", 64.},
	{"dividend_date", 96.},
	{"dividend_payment", 64.},
	{"timestamp", 120.},
}

// Render renders the row.
func (r TableRow) Render(_ *gpui.Window, _ *gpui.App) gpui.IntoElement {
	color := r.quote.changeColor()
	bg := gpui.Rgb(0xFAFAFA)
	if r.ix%2 == 0 {
		bg = gpui.Rgb(0xFFFFFF)
	}
	cells := make([]gpui.IntoElement, 0, len(fields))
	for _, f := range fields {
		cells = append(cells, r.renderCell(f.key, gpui.Px(f.width), color))
	}
	return gpui.NewDiv().
		Flex().
		FlexRow().
		BorderB1().
		BorderColor(gpui.Rgb(0xE0E0E0)).
		Bg(bg).
		Py0p5().
		Px2().
		Children(cells...)
}

// DataTable renders 10000 quotes in a uniform list with a custom scrollbar.
type DataTable struct {
	quotes       []*Quote
	visibleRange gpui.Range
	scrollHandle gpui.UniformListScrollHandle
	dragPosition gpui.Option[gpui.Point[gpui.Pixels]]
}

// NewDataTable creates an empty DataTable.
func NewDataTable() *DataTable {
	return &DataTable{
		quotes:       nil,
		visibleRange: gpui.Range{Start: 0, End: 0},
		scrollHandle: gpui.NewUniformListScrollHandle(),
		dragPosition: gpui.None[gpui.Point[gpui.Pixels]](),
	}
}

func (t *DataTable) generate() {
	t.quotes = make([]*Quote, totalItems)
	for i := range t.quotes {
		t.quotes[i] = RandomQuote()
	}
}

func (t *DataTable) tableBounds() gpui.Bounds[gpui.Pixels] {
	return t.scrollHandle.State.BaseHandle.Bounds()
}

func (t *DataTable) scrollTop() gpui.Pixels {
	return t.scrollHandle.State.BaseHandle.Offset().Y
}

func (t *DataTable) scrollHeight() gpui.Pixels {
	return t.scrollHandle.State.LastItemSize.UnwrapOrDefault().Contents.Height
}

func clampPx(v, lo, hi gpui.Pixels) gpui.Pixels {
	if !(lo <= hi) {
		panic("assertion failed: min <= max")
	}
	return min(max(v, lo), hi)
}

func (t *DataTable) renderScrollbar(_ *gpui.Window, cx *gpui.Context[DataTable]) gpui.IntoElement {
	scrollHeight := t.scrollHeight()
	tableBounds := t.tableBounds()
	tableHeight := tableBounds.Size.Height
	if tableHeight == gpui.Px(0.) {
		return gpui.NewDiv().ID("scrollbar")
	}

	percentage := float32(-t.scrollTop() / scrollHeight)
	offsetTop := clampPx(
		tableHeight*gpui.Pixels(percentage),
		gpui.Px(4.),
		max(tableHeight-scrollbarThumbHeight-gpui.Px(4.), gpui.Px(4.)),
	)
	entity := cx.Entity()
	scrollHandle := t.scrollHandle.State.BaseHandle

	return gpui.NewDiv().
		ID("scrollbar").
		Absolute().
		Top(offsetTop).
		Right1().
		H(scrollbarThumbHeight).
		W(scrollbarThumbWidth).
		Bg(gpui.Rgb(0xC0C0C0)).
		Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(this).Bg(gpui.Rgb(0xA0A0A0)) }).
		RoundedLg().
		Child(
			gpui.NewCanvas(
				func(gpui.Bounds[gpui.Pixels], *gpui.Window, *gpui.App) struct{} { return struct{}{} },
				func(thumbBounds gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, _ *gpui.App) {
					{
						entity := entity.Clone()
						window.OnMouseEvent(func(ev *gpui.MouseDownEvent, _ gpui.DispatchPhase, _ *gpui.Window, cx *gpui.App) {
							if !thumbBounds.Contains(ev.Position) {
								return
							}

							entity.Update(cx, func(this *DataTable, _ *gpui.Context[DataTable]) {
								this.dragPosition = gpui.Some(
									ev.Position.Sub(thumbBounds.Origin).Sub(tableBounds.Origin),
								)
							})
						})
					}
					{
						entity := entity.Clone()
						window.OnMouseEvent(func(_ *gpui.MouseUpEvent, _ gpui.DispatchPhase, _ *gpui.Window, cx *gpui.App) {
							entity.Update(cx, func(this *DataTable, _ *gpui.Context[DataTable]) {
								this.dragPosition = gpui.None[gpui.Point[gpui.Pixels]]()
							})
						})
					}

					window.OnMouseEvent(func(ev *gpui.MouseMoveEvent, _ gpui.DispatchPhase, _ *gpui.Window, cx *gpui.App) {
						if !ev.Dragging() {
							return
						}

						dragPos, ok := entity.Read(cx).dragPosition.Get()
						if !ok {
							return
						}

						insideOffset := dragPos.Y
						percentage := min(max(float32((ev.Position.Y-tableBounds.Origin.Y+insideOffset)/
							tableBounds.Size.Height), 0.), 1.)

						offsetY := clampPx(
							(scrollHeight-tableBounds.Size.Height)*gpui.Pixels(percentage),
							gpui.Px(0.),
							scrollHeight-scrollbarThumbHeight,
						)
						scrollHandle.SetOffset(gpui.NewPoint(gpui.Px(0.), -offsetY))
						cx.Notify(entity.EntityID())
					})
				},
			).SizeFull(),
		)
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

// Render renders the view.
func (t *DataTable) Render(window *gpui.Window, cx *gpui.Context[DataTable]) gpui.IntoElement {
	headers := make([]gpui.IntoElement, 0, len(fields))
	for _, f := range fields {
		headers = append(headers, gpui.NewDiv().
			WhitespaceNowrap().
			FlexShrink0().
			Truncate().
			Px1().
			W(gpui.Px(f.width)).
			ChildText(strings.ToUpper(strings.ReplaceAll(f.key, "_", " "))))
	}

	return gpui.NewDiv().
		Bg(gpui.White()).
		TextSm().
		SizeFull().
		P4().
		Gap2().
		Flex().
		FlexCol().
		ChildText(fmt.Sprintf(
			"Total %d items, visible range: %d..%d",
			len(t.quotes),
			t.visibleRange.Start,
			t.visibleRange.End,
		)).
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				Flex1().
				OverflowHidden().
				Border1().
				BorderColor(gpui.Rgb(0xE0E0E0)).
				RoundedSm().
				Child(
					gpui.NewDiv().
						Flex().
						FlexRow().
						WFull().
						OverflowHidden().
						BorderB1().
						BorderColor(gpui.Rgb(0xE0E0E0)).
						TextColor(gpui.Rgb(0x555555)).
						Bg(gpui.Rgb(0xF0F0F0)).
						Py1().
						Px2().
						TextXs().
						Children(headers...),
				).
				Child(
					gpui.NewDiv().
						Relative().
						SizeFull().
						Child(
							gpui.NewUniformList(
								"items",
								len(t.quotes),
								cx.Processor(func(this *DataTable, visibleRange gpui.Range, _ *gpui.Window, _ *gpui.Context[DataTable]) []TableRow {
									this.visibleRange = visibleRange
									items := make([]TableRow, 0, visibleRange.End-visibleRange.Start)
									for i := visibleRange.Start; i < visibleRange.End; i++ {
										if i < len(this.quotes) {
											items = append(items, NewTableRow(i, this.quotes[i]))
										}
									}
									return items
								}),
							).
								SizeFull().
								TrackUniformListScroll(t.scrollHandle),
						).
						Child(t.renderScrollbar(window, cx)),
				),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.Focus = true
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(1280.0), gpui.Px(1000.0)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[DataTable] {
			return cx.New(func(*gpui.Context[DataTable]) *DataTable {
				table := NewDataTable()
				table.generate()
				return table
			})
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	runExample()
}
