//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"log/slog"
	"syscall/js"

	"github.com/megascan/gpui-go"
)

// WebViewport holds browser viewport and safe-area frame snapshots in canvas coordinates.
// Browser geometry is sampled before rendering, never by geometry getters. Safe-area padding is
// resolved by CSS so display cutouts use the browser's own environment values.
type WebViewport struct {
	safeAreaProbes [4]js.Value
	visibleBounds  gpui.Bounds[gpui.Pixels]
	insets         gpui.WindowInsets
	heldLayout     gpui.Option[heldLayout]
	closed         bool
}

type heldLayout struct {
	width           float64
	height          float64
	canvasHeight    string
	releaseDeadline gpui.Option[float64]
}

const keyboardCloseTimeoutMs float64 = 1000.

func newWebViewport(document js.Value) (*WebViewport, error) {
	var probes [4]js.Value
	for i, edge := range [4]string{"top", "right", "bottom", "left"} {
		probe, err := safeAreaProbe(document, edge)
		if err != nil {
			return nil, err
		}
		probes[i] = probe
	}
	viewport := &WebViewport{safeAreaProbes: probes}
	body := jsGet(document, "body")
	if jsIsNullish(body) {
		return nil, errors.New("Missing document body")
	}
	for _, probe := range viewport.safeAreaProbes {
		if _, err := jsCall(body, "appendChild", probe); err != nil {
			viewport.Close()
			return nil, fmt.Errorf("Failed to attach safe-area probe: %s", jsErrorDebug(err))
		}
	}
	return viewport, nil
}

func (v *WebViewport) observeSafeArea(observer js.Value) {
	for _, probe := range v.safeAreaProbes {
		_, _ = jsCall(observer, "observe", probe)
	}
}

func dateNow() float64 {
	return js.Global().Get("Date").Call("now").Float()
}

func clampF64(v, lo, hi float64) float64 {
	if lo > hi {
		panic(fmt.Sprintf("min > max, or either was NaN. min = %v, max = %v", lo, hi))
	}
	return max(lo, min(v, hi))
}

func clientSize(root js.Value) (float64, float64) {
	width, _ := jsNumber(jsGet(root, "clientWidth"))
	height, _ := jsNumber(jsGet(root, "clientHeight"))
	return float64(int32(width)), float64(int32(height))
}

type domRect struct {
	left, top, width, height, right, bottom float64
}

func boundingClientRect(element js.Value) domRect {
	rect, err := jsCall(element, "getBoundingClientRect")
	if err != nil {
		return domRect{}
	}
	f := func(name string) float64 {
		n, _ := jsNumber(jsGet(rect, name))
		return n
	}
	return domRect{f("left"), f("top"), f("width"), f("height"), f("right"), f("bottom")}
}

func (v *WebViewport) update(window js.Value, canvas js.Value) (bool, error) {
	document := jsGet(window, "document")
	if jsIsNullish(document) {
		return false, errors.New("Missing document")
	}
	root := jsGet(document, "documentElement")
	if jsIsNullish(root) {
		return false, errors.New("Missing root")
	}
	layoutWidth, layoutHeight := clientSize(root)
	if held, ok := v.heldLayout.Get(); ok {
		releasing := held.releaseDeadline.IsSome()
		deadline, hasDeadline := held.releaseDeadline.Get()
		if held.width != layoutWidth || layoutHeight > held.height {
			v.restoreCanvasHeight(canvas)
			if !releasing {
				v.holdCanvasHeight(window, canvas)
			}
		} else if hasDeadline && (layoutHeight >= held.height || dateNow() >= deadline) {
			v.restoreCanvasHeight(canvas)
		} else {
			layoutHeight = held.height
		}
	}
	canvasBounds := boundingClientRect(canvas)
	var left, top, width, height float64
	if visual := jsGet(window, "visualViewport"); !jsIsNullish(visual) {
		left, _ = jsNumber(jsGet(visual, "offsetLeft"))
		top, _ = jsNumber(jsGet(visual, "offsetTop"))
		width, _ = jsNumber(jsGet(visual, "width"))
		height, _ = jsNumber(jsGet(visual, "height"))
	} else {
		left, top, width, height = 0., 0., layoutWidth, layoutHeight
	}
	x := clampF64(left-canvasBounds.left, 0., canvasBounds.width)
	y := clampF64(top-canvasBounds.top, 0., canvasBounds.height)
	right := clampF64(left+width-canvasBounds.left, x, canvasBounds.width)
	bottom := clampF64(top+height-canvasBounds.top, y, canvasBounds.height)
	visibleBounds := gpui.NewBounds(
		gpui.NewPoint(gpui.Px(float32(x)), gpui.Px(float32(y))),
		gpui.NewSize(gpui.Px(float32(right-x)), gpui.Px(float32(bottom-y))),
	)

	var probeHeights [4]float64
	for i, probe := range v.safeAreaProbes {
		probeHeights[i] = boundingClientRect(probe).height
	}
	topInset, rightInset, bottomInset, leftInset := probeHeights[0], probeHeights[1], probeHeights[2], probeHeights[3]
	safeArea := gpui.Edges[gpui.Pixels]{
		Top:    gpui.Px(float32(max(topInset-canvasBounds.top, 0.))),
		Right:  gpui.Px(float32(max(canvasBounds.right-layoutWidth+rightInset, 0.))),
		Bottom: gpui.Px(float32(max(canvasBounds.bottom-layoutHeight+bottomInset, 0.))),
		Left:   gpui.Px(float32(max(leftInset-canvasBounds.left, 0.))),
	}
	insets := gpui.WindowInsets{SafeArea: safeArea}
	changed := v.visibleBounds != visibleBounds || v.insets != insets
	v.visibleBounds = visibleBounds
	v.insets = insets
	return changed, nil
}

func (v *WebViewport) holdCanvasHeight(window js.Value, canvas js.Value) {
	if held := v.heldLayout.AsMut(); held != nil {
		deadline := held.releaseDeadline.Take()
		if d, ok := deadline.Get(); ok && dateNow() >= d {
			v.restoreCanvasHeight(canvas)
		} else {
			return
		}
	}
	root := jsGet(jsGet(window, "document"), "documentElement")
	if jsIsNullish(root) {
		return
	}
	style := jsGet(canvas, "style")
	canvasHeight := ""
	if value, err := jsCall(style, "getPropertyValue", "height"); err == nil {
		canvasHeight, _ = jsString(value)
	}
	height := boundingClientRect(canvas).height
	if _, err := jsCall(style, "setProperty", "height", formatF64(height)+"px"); err != nil {
		slog.Warn(fmt.Sprintf("Failed to hold the canvas height: %s", jsErrorDebug(err)))
		return
	}
	width, rootHeight := clientSize(root)
	v.heldLayout = gpui.Some(heldLayout{
		width:           width,
		height:          rootHeight,
		canvasHeight:    canvasHeight,
		releaseDeadline: gpui.None[float64](),
	})
}

func (v *WebViewport) releaseCanvasHeight() {
	if held := v.heldLayout.AsMut(); held != nil {
		if held.releaseDeadline.IsNone() {
			held.releaseDeadline = gpui.Some(dateNow() + keyboardCloseTimeoutMs)
		}
	}
}

func (v *WebViewport) restoreCanvasHeight(canvas js.Value) {
	held, ok := v.heldLayout.Take().Get()
	if !ok {
		return
	}
	if _, err := jsCall(jsGet(canvas, "style"), "setProperty", "height", held.canvasHeight); err != nil {
		slog.Warn(fmt.Sprintf("Failed to release the canvas height: %s", jsErrorDebug(err)))
	}
}

func (v *WebViewport) setCanvasHeight(canvas js.Value, height string) {
	if held := v.heldLayout.AsMut(); held != nil {
		held.canvasHeight = height
		return
	}
	_, _ = jsCall(jsGet(canvas, "style"), "setProperty", "height", height)
}

func (v *WebViewport) keyboardHeight(window js.Value) (float64, bool) {
	viewport := jsGet(window, "visualViewport")
	if jsIsNullish(viewport) {
		return 0, false
	}
	scale, _ := jsNumber(jsGet(viewport, "scale"))
	if abs := scale - 1.; abs >= 0.001 || abs <= -0.001 {
		return 0, false
	}
	var layoutHeight float64
	if held, ok := v.heldLayout.Get(); ok {
		layoutHeight = held.height
	} else {
		root := jsGet(jsGet(window, "document"), "documentElement")
		if jsIsNullish(root) {
			return 0, false
		}
		_, layoutHeight = clientSize(root)
	}
	height, _ := jsNumber(jsGet(viewport, "height"))
	return max(layoutHeight-height, 0.), true
}

// Close removes the safe-area probes from the document; it is upstream's drop and runs once.
func (v *WebViewport) Close() {
	if v.closed {
		return
	}
	v.closed = true
	for _, probe := range v.safeAreaProbes {
		_, _ = jsCall(probe, "remove")
	}
}

func safeAreaProbe(document js.Value, edge string) (js.Value, error) {
	probe, err := jsCall(document, "createElement", "div")
	if err != nil {
		return js.Undefined(), fmt.Errorf("Failed to create safe-area probe: %s", jsErrorDebug(err))
	}
	if !jsInstanceOf(probe, "HTMLElement") {
		return js.Undefined(), fmt.Errorf("Invalid safe-area probe element: %s", jsDebug(probe))
	}
	height := fmt.Sprintf("env(safe-area-inset-%s,0px)", edge)
	style := probe.Get("style")
	for _, pv := range [...][2]string{
		{"position", "fixed"},
		{"left", "0"},
		{"top", "0"},
		{"width", "0"},
		{"padding", "0"},
		{"border", "0"},
		{"visibility", "hidden"},
		{"pointer-events", "none"},
		{"height", height},
	} {
		if _, err := jsCall(style, "setProperty", pv[0], pv[1]); err != nil {
			return js.Undefined(), fmt.Errorf("Failed to set safe-area probe %s style: %s", pv[0], jsErrorDebug(err))
		}
	}
	return probe, nil
}
