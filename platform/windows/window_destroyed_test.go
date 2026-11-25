//go:build windows

package windows

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

func captureLogs(t *testing.T) *[]string {
	t.Helper()
	var mu sync.Mutex
	var msgs []string
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		msgs = append(msgs, string(p))
		return len(p), nil
	}), nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &msgs
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestWindow_LastErrorForNilHasNoFormatVerb(t *testing.T) {
	if got := lastErrorFor("ScreenToClient", nil).Error(); got != "ScreenToClient failed" {
		t.Fatalf("lastErrorFor(nil) = %q", got)
	}
	inner := errors.New("Invalid window handle.")
	if err := lastErrorFor("ScreenToClient", inner); !errors.Is(err, inner) || strings.Contains(err.Error(), "%!") {
		t.Fatalf("lastErrorFor(err) = %v", err)
	}
}

func TestWindow_DestroyedWindowUsesCachedState(t *testing.T) {
	logs := captureLogs(t)
	cached := win32.WINDOWPLACEMENT{ShowCmd: win32.SW_SHOWMAXIMIZED}
	cached.RcNormalPosition = win32.RECT{Left: 10, Top: 20, Right: 110, Bottom: 220}
	w := &WindowsWindow{inner: &WindowsWindowInner{state: WindowsWindowState{
		scaleFactor:   1,
		destroyed:     true,
		lastPlacement: gpui.Some(cached),
		mousePosition: gpui.Point[gpui.Pixels]{X: 3, Y: 4},
	}}}
	_, maximized := w.inner.state.calculateWindowBounds()
	if !maximized {
		t.Fatal("destroyed window did not report its cached placement")
	}
	if got := w.MousePosition(); got != (gpui.Point[gpui.Pixels]{X: 3, Y: 4}) {
		t.Fatalf("MousePosition = %v, want the last known position", got)
	}
	if len(*logs) != 0 {
		t.Fatalf("destroyed window logged %v", *logs)
	}
}
