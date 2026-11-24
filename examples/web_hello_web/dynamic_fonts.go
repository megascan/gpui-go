//go:build js && wasm

package main

import (
	"fmt"
	"io"
	"net/http"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	FontURL = "https://fonts.gstatic.com/s/notosansdevanagari/v30/TuGoUUFzXI5FBtUq5a8bjKYTZjtRU6Sgv3NaV_SNmI0b8QQCQmHn6B2OHjbL_08AlXQly-A.ttf"
	Sample  = "नमस्ते 😀 दुनिया 👨‍👩‍👧‍👦 नमस्ते ❤️"
)

type DynamicFonts struct {
	status          string
	download        *gpui.Task[struct{}]
	missingGlyphsCB gpui.Subscription
}

func NewDynamicFonts(cx *gpui.Context[DynamicFonts]) *DynamicFonts {
	view := cx.WeakEntity()
	subscription := cx.OnMissingGlyphs(func(missing []gpui.MissingGlyph, cx *gpui.App) {
		for _, glyph := range missing {
			for _, character := range glyph.Grapheme() {
				if character >= 'ऀ' && character <= 'ॿ' {
					if err := view.Update(cx, func(view *DynamicFonts, cx *gpui.Context[DynamicFonts]) {
						view.LoadFont(cx)
					}); err != nil {
						js.Global().Get("console").Call("error", err.Error())
					}
					return
				}
			}
		}
	})
	return &DynamicFonts{
		status:          "Waiting for a missing Devanagari glyph…",
		missingGlyphsCB: subscription,
	}
}

func (d *DynamicFonts) LoadFont(cx *gpui.Context[DynamicFonts]) {
	if d.download != nil {
		return
	}
	d.status = "Downloading Noto Sans Devanagari (219 KB)…"
	cx.Notify()
	httpClient := cx.HTTPClient()
	d.download = cx.Spawn(func(view gpui.WeakEntity[DynamicFonts], cx *gpui.AsyncApp) (struct{}, error) {
		bytes, err := fetchFont(cx, httpClient)
		if err == nil {
			err = cx.UpdateR(func(app *gpui.App) error {
				return app.TextSystem().AddFonts([][]byte{bytes})
			})
		}
		return struct{}{}, view.Update(cx, func(view *DynamicFonts, cx *gpui.Context[DynamicFonts]) {
			if err != nil {
				view.status = fmt.Sprintf("Font loading failed: %v. Reload to retry.", err)
			} else {
				view.status = "Font loaded. The same text is now shaped by Cosmic Text."
			}
			cx.RefreshWindows()
			cx.Notify()
		})
	})
}

type bodyResult struct {
	bytes []byte
	err   error
}

func fetchFont(cx *gpui.AsyncApp, httpClient gpui.HTTPClient) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, FontURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := httpClient.Send(request).Await(cx)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("Font request failed: HTTP %d", response.StatusCode)
	}
	body, err := readBodyOffTask(cx, response.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("Font response was empty")
	}
	return body, nil
}

func readBodyOffTask(cx scheduler.Awaiter, body io.ReadCloser) ([]byte, error) {
	sender, receiver := scheduler.Oneshot[bodyResult]()
	go func() {
		defer body.Close()
		bytes, err := io.ReadAll(body)
		_ = sender.Send(bodyResult{bytes: bytes, err: err})
	}()
	result, err := receiver.Recv(cx)
	if err != nil {
		return nil, err
	}
	return result.bytes, result.err
}

func (d *DynamicFonts) Render(_ *gpui.Window, _ *gpui.Context[DynamicFonts]) gpui.IntoElement {
	return gpui.NewDiv().
		SizeFull().
		Flex().FlexCol().
		Gap4().P8().
		Bg(gpui.Rgb(0x202030)).
		TextColor(gpui.Rgb(0xffffff)).
		FontFamily("IBM Plex Sans").
		Child(gpui.NewDiv().Text2xl().ChildText("On-demand font loading")).
		ChildText("Only IBM Plex Sans is bundled. Missing Devanagari triggers the font download.").
		Child(gpui.NewDiv().TextSize(gpui.Px(40)).ChildText(Sample)).
		Child(gpui.NewDiv().TextSize(gpui.Px(32)).ChildText("Emoji still uses Canvas: 😀 👨‍👩‍👧‍👦")).
		ChildText(d.status).
		ChildText("Source: Google Fonts CDN · Noto Sans Devanagari · SIL Open Font License").
		ChildText("Use the browser Network panel to throttle or block the font request, then reload.")
}

func RunDynamicFonts() {
	runApplication(func(cx *gpui.App) {
		if err := cx.TextSystem().AddFonts([][]byte{IBMPlexSansRegular}); err != nil {
			js.Global().Get("console").Call("error", err.Error())
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(900), gpui.Px(500)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[DynamicFonts] {
			return cx.New(NewDynamicFonts)
		}); err != nil {
			js.Global().Get("console").Call("error", err.Error())
		}
		cx.Activate(true)
	})
}
