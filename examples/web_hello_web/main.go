//go:build js && wasm

package main

import (
	_ "embed"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall/js"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
	"github.com/megascan/gpui-go/platform/web"
	"github.com/megascan/gpui-go/scheduler"
)

const NumChunks = 12

//go:embed fonts/IBMPlexSans-Regular.ttf
var IBMPlexSansRegular []byte

var application *gpui.ApplicationHandle

func IsPrime(n uint64) bool {
	if n < 2 {
		return false
	}
	if n < 4 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	for i := uint64(5); i*i <= n; i += 6 {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
	}
	return true
}

func CountPrimesInRange(start, end uint64) uint64 {
	var count uint64
	for n := start; n < end; n++ {
		if IsPrime(n) {
			count++
		}
	}
	return count
}

type Preset uint8

const (
	PresetTenMillion Preset = iota
	PresetFiftyMillion
	PresetHundredMillion
)

var allPresets = [...]Preset{PresetTenMillion, PresetFiftyMillion, PresetHundredMillion}

func (p Preset) Label() string {
	switch p {
	case PresetTenMillion:
		return "10 M"
	case PresetFiftyMillion:
		return "50 M"
	default:
		return "100 M"
	}
}

func (p Preset) Value() uint64 {
	switch p {
	case PresetTenMillion:
		return 10_000_000
	case PresetFiftyMillion:
		return 50_000_000
	default:
		return 100_000_000
	}
}

type ChunkResult struct {
	count uint64
}

type Run struct {
	limit        uint64
	chunksDone   uint64
	chunkResults []ChunkResult
	done         bool
	total        uint64
	elapsedMS    float64
}

type HelloWeb struct {
	selectedPreset Preset
	currentRun     *Run
	history        []gpui.SharedString
	tasks          []*gpui.Task[struct{}]
}

func NewHelloWeb(*gpui.Context[HelloWeb]) *HelloWeb {
	return &HelloWeb{selectedPreset: PresetTenMillion}
}

func (h *HelloWeb) StartSearch(cx *gpui.Context[HelloWeb]) {
	limit := h.selectedPreset.Value()
	chunkSize := limit / NumChunks

	h.currentRun = &Run{limit: limit}
	for _, task := range h.tasks {
		task.Cancel()
	}
	h.tasks = h.tasks[:0]
	cx.Notify()

	start := time.Now()
	for i := uint64(0); i < NumChunks; i++ {
		rangeStart := i * chunkSize
		rangeEnd := rangeStart + chunkSize
		if i == NumChunks-1 {
			rangeEnd = limit
		}

		task := cx.Spawn(func(this gpui.WeakEntity[HelloWeb], cx *gpui.AsyncApp) (struct{}, error) {
			count, err := cx.BackgroundSpawn(func(*scheduler.Scope) (uint64, error) {
				return CountPrimesInRange(rangeStart, rangeEnd), nil
			}).Await(cx)
			if err != nil {
				return struct{}{}, err
			}
			return struct{}{}, this.Update(cx, func(h *HelloWeb, cx *gpui.Context[HelloWeb]) {
				if h.currentRun == nil {
					return
				}
				r := h.currentRun
				r.chunkResults = append(r.chunkResults, ChunkResult{count: count})
				r.chunksDone++
				if r.chunksDone == NumChunks {
					var total uint64
					for _, c := range r.chunkResults {
						total += c.count
					}
					elapsedMS := float64(time.Since(start).Microseconds()) / 1000
					r.done = true
					r.total = total
					r.elapsedMS = elapsedMS
					h.history = append(h.history, gpui.SharedString(fmt.Sprintf(
						"π(%s) = %s (%.0f ms, %d chunks)",
						FormatNumber(r.limit), FormatNumber(total), elapsedMS, NumChunks,
					)))
				}
				cx.Notify()
			})
		})
		h.tasks = append(h.tasks, task)
	}
}

func FormatNumber(n uint64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

const (
	BgBase        uint32 = 0x1e1e2e
	BgSurface     uint32 = 0x313244
	BgOverlay     uint32 = 0x45475a
	TextPrimary   uint32 = 0xcdd6f4
	TextSecondary uint32 = 0xa6adc8
	TextDim       uint32 = 0x6c7086
	AccentYellow  uint32 = 0xf9e2af
	AccentGreen   uint32 = 0xa6e3a1
	AccentBlue    uint32 = 0x89b4fa
	AccentMauve   uint32 = 0xcba6f7
)

func (h *HelloWeb) Render(_ *gpui.Window, cx *gpui.Context[HelloWeb]) gpui.IntoElement {
	isRunning := h.currentRun != nil && !h.currentRun.done

	presetRow := gpui.NewDiv().Flex().FlexRow().Gap2()
	for index, p := range allPresets {
		isSelected := p == h.selectedPreset
		bg, text := BgOverlay, TextSecondary
		if isSelected {
			bg, text = AccentBlue, BgBase
		}
		selected := p
		presetRow = presetRow.Child(
			gpui.NewDiv().
				ID(gpui.ElementIDNamedInteger("preset", uint64(index))).
				Px3().Py1().RoundedMd().
				Bg(gpui.Rgb(bg)).
				TextColor(gpui.Rgb(text)).
				TextSm().
				CursorPointer().
				When(!isRunning, func(s *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
					return s.OnClick(cx.Listener(func(this *HelloWeb, _ *gpui.ClickEvent, _ *gpui.Window, _ *gpui.Context[HelloWeb]) {
						this.selectedPreset = selected
					}))
				}).
				ChildText(p.Label()),
		)
	}

	goBg, goText, goLabel := AccentGreen, BgBase, "Count Primes"
	if isRunning {
		goBg, goText, goLabel = BgOverlay, TextDim, "Running…"
	}
	goButton := gpui.NewDiv().
		ID("go").
		Px4().Py(gpui.Px(6)).RoundedMd().
		Bg(gpui.Rgb(goBg)).
		TextColor(gpui.Rgb(goText)).
		CursorPointer().
		When(!isRunning, func(s *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
			return s.OnClick(cx.Listener(func(this *HelloWeb, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[HelloWeb]) {
				this.StartSearch(cx)
			}))
		}).
		ChildText(goLabel)

	var statusArea gpui.IntoElement
	if r := h.currentRun; r != nil {
		progressFraction := float32(r.chunksDone) / float32(NumChunks)
		progressPct := uint32(progressFraction * 100)

		var statusText string
		if r.done {
			statusText = fmt.Sprintf("Found %s primes below %s in %.0f ms",
				FormatNumber(r.total), FormatNumber(r.limit), r.elapsedMS)
		} else {
			statusText = fmt.Sprintf("Searching up to %s … %d/%d chunks  (%d%%)",
				FormatNumber(r.limit), r.chunksDone, NumChunks, progressPct)
		}

		barColor := AccentBlue
		if r.done {
			barColor = AccentGreen
		}

		chunkDots := gpui.NewDiv().Flex().FlexRow().Gap1().Mt2()
		for i := 0; i < NumChunks; i++ {
			color := BgOverlay
			if uint64(i) < r.chunksDone {
				color = AccentMauve
			}
			chunkDots = chunkDots.Child(gpui.NewDiv().Size(gpui.Px(10)).RoundedSm().Bg(gpui.Rgb(color)))
		}

		statusArea = gpui.NewDiv().
			Flex().FlexCol().WFull().Gap2().
			Child(gpui.NewDiv().TextColor(gpui.Rgb(TextPrimary)).Child(gpui.SharedString(statusText))).
			Child(gpui.NewDiv().
				WFull().H(gpui.Px(8)).RoundedSm().Bg(gpui.Rgb(BgOverlay)).
				Child(gpui.NewDiv().
					HFull().RoundedSm().Bg(gpui.Rgb(barColor)).
					W(gpui.Relative(progressFraction)))).
			Child(chunkDots)
	} else {
		statusArea = gpui.NewDiv().Flex().FlexCol().WFull().
			Child(gpui.NewDiv().TextColor(gpui.Rgb(TextDim)).ChildText("Select a range and press Count Primes to begin."))
	}

	var historySection gpui.IntoElement
	if len(h.history) == 0 {
		historySection = gpui.NewDiv()
	} else {
		col := gpui.NewDiv().Flex().FlexCol().Gap1()
		for i := len(h.history) - 1; i >= 0; i-- {
			col = col.Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(TextSecondary)).Child(h.history[i]))
		}
		historySection = col
	}

	root := gpui.NewDiv().
		Flex().FlexCol().SizeFull().
		Bg(gpui.Rgb(BgBase)).
		JustifyCenter().ItemsCenter().
		Gap4().P4().
		Child(gpui.NewDiv().TextXl().TextColor(gpui.Rgb(TextPrimary)).ChildText("Prime Sieve — GPUI Web")).
		Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(TextDim)).ChildText(fmt.Sprintf(
			"Background threads: %d · Chunks per run: %d", max(runtime.NumCPU(), 2), NumChunks))).
		Child(gpui.NewDiv().
			Flex().FlexCol().ItemsCenter().Gap3().P4().
			W(gpui.Px(500)).RoundedLg().Bg(gpui.Rgb(BgSurface)).
			Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(AccentYellow)).ChildText("Count primes below:")).
			Child(presetRow).
			Child(goButton)).
		Child(gpui.NewDiv().
			Flex().FlexCol().W(gpui.Px(500)).P4().RoundedLg().Bg(gpui.Rgb(BgSurface)).
			Child(statusArea))

	if len(h.history) > 0 {
		root = root.Child(gpui.NewDiv().
			Flex().FlexCol().W(gpui.Px(500)).P4().RoundedLg().Bg(gpui.Rgb(BgSurface)).Gap2().
			Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(TextDim)).ChildText("History")).
			Child(historySection))
	}
	return root
}

func RequestedBackend() web.WebBackendPreference {
	search := js.Global().Get("location").Get("search").String()
	for _, parameter := range strings.Split(strings.TrimLeft(search, "?"), "&") {
		switch parameter {
		case "backend=webgpu":
			return web.WebBackendPreferenceWebGpu
		case "backend=webgl":
			return web.WebBackendPreferenceWebGl
		}
	}
	return web.WebBackendPreferenceAuto
}

func runApplication(onFinishLaunching func(cx *gpui.App)) {
	application = platform.ApplicationWithWebBackend(RequestedBackend()).RunEmbedded(onFinishLaunching)
}

func runHelloWeb() {
	runApplication(func(cx *gpui.App) {
		if err := cx.TextSystem().AddFonts([][]byte{IBMPlexSansRegular}); err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("failed to load application fonts: %v", err))
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(640), gpui.Px(560)), cx)
		if _, err := cx.OpenWindow(gpui.WindowOptions{
			WindowBounds: gpui.Some(gpui.WindowBoundsWindowed(bounds)),
		}, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWeb] {
			return cx.New(NewHelloWeb)
		}); err != nil {
			panic(fmt.Sprintf("failed to open window: %v", err))
		}
		cx.Activate(true)
	})
}

func main() {
	platform.WebInit()
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	switch name {
	case "dynamic-fonts":
		RunDynamicFonts()
	case "font-fallback-test":
		registerFontFallbackTest()
	case "queue-test":
		registerQueueTest()
	case "http-client-test":
		registerHTTPClientTest()
	default:
		runHelloWeb()
	}
	select {}
}
