package platform_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
)

const (
	inspectorColumnCount = 100
	inspectorNodeSize    = float32(6.0)
	inspectorRowHeight   = float32(24.0)
	inspectorPanelWidth  = float32(320.0)
)

type closedInspectorInput struct {
	scale                int
	unit                 string
	openCloseCycles      int
	otherWindowInspected bool
}

func (i closedInspectorInput) String() string {
	s := fmt.Sprintf("%d-%s-%d-cycles", i.scale, i.unit, i.openCloseCycles)
	if i.otherWindowInspected {
		s += "-other-window"
	}
	return s
}

func closedInspectorInputs(unit string, scales ...int) []closedInspectorInput {
	lifecycles := []struct {
		cycles int
		other  bool
	}{{0, false}, {2, false}, {0, true}, {2, true}}
	var inputs []closedInspectorInput
	for _, scale := range scales {
		for _, lifecycle := range lifecycles {
			inputs = append(inputs, closedInspectorInput{
				scale:                scale,
				unit:                 unit,
				openCloseCycles:      lifecycle.cycles,
				otherWindowInspected: lifecycle.other,
			})
		}
	}
	return inputs
}

func BenchmarkInspectorRender_BareGrid(b *testing.B) {
	if gpuiutil.DebugAssertions {
		b.Skip("use -tags gpui_release")
	}
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "case", Group: "closed_inspector_grid", Fps: 120})
	gpui.RunBenchInputs(b, opts, closedInspectorInputs("nodes", 100, 1_000, 10_000), func(input closedInspectorInput, cx *gpui.BenchAppContext) {
		measureClosedInspector(input, cx,
			func() *bareGrid { return &bareGrid{nodeCount: input.scale} },
			func(grid *bareGrid) { grid.alternate = !grid.alternate },
		)
	})
}

func BenchmarkInspectorRender_Panel(b *testing.B) {
	if gpuiutil.DebugAssertions {
		b.Skip("use -tags gpui_release")
	}
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "case", Group: "closed_inspector_panel", Fps: 120})
	gpui.RunBenchInputs(b, opts, closedInspectorInputs("rows", 50, 500), func(input closedInspectorInput, cx *gpui.BenchAppContext) {
		measureClosedInspector(input, cx,
			func() *inspectorPanel { return newInspectorPanel(input.scale) },
			func(panel *inspectorPanel) { panel.selectedRow = (panel.selectedRow + 1) % len(panel.labels) },
		)
	})
}

func measureClosedInspector[V any](input closedInspectorInput, cx *gpui.BenchAppContext, buildView func() *V, updateView func(view *V)) {
	window := cx.AddEmptyWindow()
	view := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[V] {
		return w.ReplaceRoot(app, func(*gpui.Window, *gpui.Context[V]) *V { return buildView() })
	})
	cx.RunUntilIdle()
	for range input.openCloseCycles * 2 {
		window.Update(func(w *gpui.Window, app *gpui.App) {
			w.ToggleInspector(app)
			w.Refresh()
		})
		cx.RunUntilIdle()
	}
	if window.UpdateR(func(w *gpui.Window, app *gpui.App) bool { return w.IsInspectorPicking(app) }) {
		panic("the measured window's inspector must be closed")
	}
	if input.otherWindowInspected {
		otherWindow := cx.AddEmptyWindow()
		otherWindow.Update(func(w *gpui.Window, app *gpui.App) { w.ToggleInspector(app) })
		if !otherWindow.UpdateR(func(w *gpui.Window, app *gpui.App) bool { return w.IsInspectorPicking(app) }) {
			panic("the other window's inspector must be open")
		}
	}
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.ActivateWindow() })
	cx.RunUntilIdle()
	if !window.UpdateR(func(w *gpui.Window, _ *gpui.App) bool { return w.IsWindowActive() }) {
		panic("the measured window must be active")
	}
	cx.RunUntilIdle()
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.Refresh() })
	cx.RunUntilIdle()

	cx.BenchRenderer(view, func(view *V, _ *gpui.Window, cx *gpui.Context[V]) {
		updateView(view)
		cx.Notify()
	})
	if os.Getenv("GPUI_BENCH_SCENE_STATS") == "1" {
		window.Update(func(w *gpui.Window, _ *gpui.App) {
			fmt.Fprintf(os.Stderr, "scene stats inspector/%v: batches=%d\n", input, sceneBatchCount(w.RenderedScene()))
		})
	}
}

type bareGrid struct {
	nodeCount int
	alternate bool
}

func (g *bareGrid) Render(_ *gpui.Window, _ *gpui.Context[bareGrid]) gpui.IntoElement {
	background := gpui.Rgb(0x445566)
	if g.alternate {
		background = gpui.Rgb(0x334455)
	}
	rows := make([]gpui.IntoElement, 0, g.nodeCount/inspectorColumnCount)
	for rowIndex := range g.nodeCount / inspectorColumnCount {
		nodes := make([]gpui.IntoElement, 0, inspectorColumnCount)
		for columnIndex := range inspectorColumnCount {
			nodes = append(nodes, gpui.NewDiv().
				ID(gpui.ElementIDNamedInteger("node", uint64(columnIndex))).
				Size(gpui.Px(inspectorNodeSize)).
				FlexShrink0().
				Bg(background).
				Border1().
				BorderColor(gpui.Rgb(0x8899aa)).
				Rounded(gpui.Px(2.0)).
				When(columnIndex%2 == 0, func(element *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
					return element.CursorPointer()
				}))
		}
		rows = append(rows, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("row", uint64(rowIndex))).
			Flex().
			H(gpui.Px(inspectorNodeSize)).
			FlexShrink0().
			Children(nodes...))
	}
	return gpui.NewDiv().
		ID("bare-grid").
		Flex().
		FlexCol().
		Size(gpui.Px(float32(inspectorColumnCount) * inspectorNodeSize)).
		Children(rows...)
}

type inspectorPanel struct {
	labels      []gpui.SharedString
	counts      []gpui.SharedString
	selectedRow int
}

func newInspectorPanel(rowCount int) *inspectorPanel {
	panel := &inspectorPanel{}
	for index := range rowCount {
		panel.labels = append(panel.labels, gpui.SharedString(fmt.Sprintf("crates/module_%d/src/file_%d.rs", index%17, index)))
		panel.counts = append(panel.counts, gpui.SharedString(fmt.Sprint((index*7)%100)))
	}
	return panel
}

func inspectorStyled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var styled gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&styled, style)
	return &styled
}

func (p *inspectorPanel) Render(_ *gpui.Window, _ *gpui.Context[inspectorPanel]) gpui.IntoElement {
	selectedRow := p.selectedRow
	rows := make([]gpui.IntoElement, 0, len(p.labels))
	for index, label := range p.labels {
		count := p.counts[index]
		swatch := gpui.Rgb(0x8fbf6a)
		if index%3 == 0 {
			swatch = gpui.Rgb(0x6aa1ff)
		}
		rows = append(rows, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("row", uint64(index))).
			Flex().
			ItemsCenter().
			Gap2().
			Px2().
			H(gpui.Px(inspectorRowHeight)).
			FlexShrink0().
			RoundedSm().
			CursorPointer().
			Hover(func(s *gpui.StyleRefinement) *gpui.StyleRefinement { return inspectorStyled(s).Bg(gpui.Rgb(0x2a3038)) }).
			Active(func(s *gpui.StyleRefinement) *gpui.StyleRefinement { return inspectorStyled(s).Bg(gpui.Rgb(0x343c46)) }).
			When(index == selectedRow, func(row *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
				return row.Bg(gpui.Rgb(0x1f4f82))
			}).
			OnClick(func(*gpui.ClickEvent, *gpui.Window, *gpui.App) {}).
			Child(gpui.NewDiv().Size4().FlexShrink0().RoundedSm().Bg(swatch)).
			Child(gpui.NewDiv().Flex1().OverflowHidden().TextSm().TextColor(gpui.Rgb(0xd0d4da)).Child(label)).
			Child(gpui.NewDiv().TextXs().TextColor(gpui.Rgb(0x8a919c)).Child(count)))
	}
	return gpui.NewDiv().
		ID("panel").
		Flex().
		FlexCol().
		W(gpui.Px(inspectorPanelWidth)).
		H(gpui.Px(inspectorRowHeight * float32(len(p.labels)))).
		Bg(gpui.Rgb(0x1b1f24)).
		Children(rows...)
}
