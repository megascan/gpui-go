//go:build linux

package wayland

import (
	"os"
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
)

func TestClient_RejectsRequiredGlobalsBelowSupportedVersions(t *testing.T) {
	if _, err := wlSeatVersion(4); err == nil {
		t.Fatal("wl_seat 4 accepted")
	}
	if _, err := wlOutputVersion(1); err == nil {
		t.Fatal("wl_output 1 accepted")
	}
	if v, err := wlSeatVersion(5); err != nil || v != 5 {
		t.Fatalf("minimum seat version: %d, %v", v, err)
	}
	if v, err := wlOutputVersion(2); err != nil || v != 2 {
		t.Fatalf("minimum output version: %d, %v", v, err)
	}
}

type fakeDataOffer struct {
	name      string
	finished  bool
	destroyed bool
}

func newFakeDataOffer(name string) *fakeDataOffer { return &fakeDataOffer{name: name} }

func (o *fakeDataOffer) Finish()  { o.finished = true }
func (o *fakeDataOffer) Destroy() { o.destroyed = true }

type fakeDrag = dragState[*fakeDataOffer, string]

func completeDragURIRead(drag *fakeDrag, generation uint64, dataOffer *fakeDataOffer, window string) (string, gpui.PlatformInput, bool) {
	return drag.completeURIRead(generation, dataOffer, window, gpui.Point[gpui.Pixels]{}, gpui.ExternalPaths{"/tmp/file"})
}

func recordEntered(enteredWindows *[]string) func(string, gpui.PlatformInput, bool) {
	return func(window string, input gpui.PlatformInput, ok bool) {
		if !ok {
			return
		}
		if _, entered := input.(gpui.FileDropEventEntered); entered {
			*enteredWindows = append(*enteredWindows, window)
		}
	}
}

func TestClient_LeaveDiscardsPendingDragURIRead(t *testing.T) {
	var drag fakeDrag
	dataOffer := newFakeDataOffer("A")
	generation := drag.beginURIRead()
	var enteredWindows []string
	record := recordEntered(&enteredWindows)

	record(drag.handleLeave())
	record(completeDragURIRead(&drag, generation, dataOffer, "window A"))

	if len(enteredWindows) != 0 || !dataOffer.destroyed || drag.dataOffer.IsSome() || drag.window.IsSome() {
		t.Fatalf("entered %v destroyed %v offer %v window %v", enteredWindows, dataOffer.destroyed, drag.dataOffer.IsSome(), drag.window.IsSome())
	}
}

func TestClient_DropThenLeaveDiscardsPendingDragURIRead(t *testing.T) {
	var drag fakeDrag
	dataOffer := newFakeDataOffer("A")
	generation := drag.beginURIRead()
	var enteredWindows []string
	record := recordEntered(&enteredWindows)

	record(drag.handleDrop())
	if drag.uriReadGeneration == generation {
		t.Fatal("drop did not invalidate the uri read")
	}
	record(drag.handleLeave())
	record(completeDragURIRead(&drag, generation, dataOffer, "window A"))

	if len(enteredWindows) != 0 || dataOffer.finished || !dataOffer.destroyed || drag.dataOffer.IsSome() || drag.window.IsSome() {
		t.Fatalf("entered %v finished %v destroyed %v", enteredWindows, dataOffer.finished, dataOffer.destroyed)
	}
}

func TestClient_StaleCompletionPreservesNewerDragDestination(t *testing.T) {
	var drag fakeDrag
	dataOfferA := newFakeDataOffer("A")
	generationA := drag.beginURIRead()
	var enteredWindows []string
	record := recordEntered(&enteredWindows)
	record(drag.handleLeave())

	dataOfferB := newFakeDataOffer("B")
	generationB := drag.beginURIRead()
	record(completeDragURIRead(&drag, generationB, dataOfferB, "window B"))
	record(completeDragURIRead(&drag, generationA, dataOfferA, "window A"))

	if !slices.Equal(enteredWindows, []string{"window B"}) {
		t.Fatalf("entered %v", enteredWindows)
	}
	if !dataOfferA.destroyed || dataOfferB.destroyed {
		t.Fatalf("destroyed A %v B %v", dataOfferA.destroyed, dataOfferB.destroyed)
	}
	if offer, ok := drag.dataOffer.Get(); !ok || offer.name != "B" {
		t.Fatalf("data offer %v", drag.dataOffer)
	}
	if window, ok := drag.window.Get(); !ok || window != "window B" {
		t.Fatalf("window %v", drag.window)
	}
}

type fakeIMECursorRectangleSink struct {
	cursorRectangles [][4]int32
	commitCount      int
}

func (s *fakeIMECursorRectangleSink) setIMECursorRectangle(x, y, width, height int32) {
	s.cursorRectangles = append(s.cursorRectangles, [4]int32{x, y, width, height})
}

func (s *fakeIMECursorRectangleSink) commitIMEState() { s.commitCount++ }

func TestClient_BuildsTerminatedURIListForEachDraggedPath(t *testing.T) {
	paths := gpui.NewFileDragPaths([]gpui.FileDragPath{{Path: "/tmp/first"}, {Path: "/tmp/nested directory", IsDir: true}})
	if got := fileURIList(paths); got != "file:///tmp/first\r\nfile:///tmp/nested%20directory\r\n" {
		t.Fatalf("got %q", got)
	}
}

func TestClient_SkipsPathsThatHaveNoFileURL(t *testing.T) {
	paths := gpui.NewFileDragPaths([]gpui.FileDragPath{{Path: "relative/path"}, {Path: "/tmp/absolute"}})
	if got := fileURIList(paths); got != "file:///tmp/absolute\r\n" {
		t.Fatalf("got %q", got)
	}
}

func TestClient_BuildsEmptyURIListWithoutConvertiblePaths(t *testing.T) {
	if got := fileURIList(gpui.FileDragPaths{}); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := fileURIList(gpui.NewFileDragPaths([]gpui.FileDragPath{{Path: "relative"}})); got != "" {
		t.Fatalf("got %q", got)
	}
}

func imeCursorBounds(x float32) gpui.Bounds[gpui.Pixels] {
	return gpui.NewBounds(gpui.NewPoint(gpui.Pixels(x), gpui.Pixels(20.25)), gpui.NewSize(gpui.Pixels(1.0), gpui.Pixels(18.75)))
}

func TestClient_CachesCursorRectangleCommittedAfterDone(t *testing.T) {
	textInput := &fakeIMECursorRectangleSink{}
	var last gpui.Option[imeCursorRectangle]
	initialBounds := imeCursorBounds(10.0)
	updatedBounds := imeCursorBounds(20.0)

	updateIMECursorRectangle(textInput, &last, initialBounds)
	setIMECursorRectangleAfterDone(textInput, &last, updatedBounds, true)
	updateIMECursorRectangle(textInput, &last, updatedBounds)

	if textInput.commitCount != 2 || len(textInput.cursorRectangles) != 2 {
		t.Fatalf("commits %d rectangles %d", textInput.commitCount, len(textInput.cursorRectangles))
	}
}

func TestClient_SkipsUnchangedCursorRectangleAfterDone(t *testing.T) {
	textInput := &fakeIMECursorRectangleSink{}
	var last gpui.Option[imeCursorRectangle]
	bounds := imeCursorBounds(10.0)

	updateIMECursorRectangle(textInput, &last, bounds)
	setIMECursorRectangleAfterDone(textInput, &last, bounds, true)

	if textInput.commitCount != 1 || len(textInput.cursorRectangles) != 1 {
		t.Fatalf("commits %d rectangles %d", textInput.commitCount, len(textInput.cursorRectangles))
	}
}

func TestClient_SkipsCursorRectanglesWithUnchangedProtocolCoordinates(t *testing.T) {
	textInput := &fakeIMECursorRectangleSink{}
	var last gpui.Option[imeCursorRectangle]

	updateIMECursorRectangle(textInput, &last, imeCursorBounds(10.25))
	updateIMECursorRectangle(textInput, &last, imeCursorBounds(10.75))

	if textInput.commitCount != 1 || !slices.Equal(textInput.cursorRectangles, [][4]int32{{10, 20, 1, 18}}) {
		t.Fatalf("commits %d rectangles %v", textInput.commitCount, textInput.cursorRectangles)
	}
}

func TestClientFileURLFromPathMatchesRustURL(t *testing.T) {
	cases := map[string]string{
		"/tmp/a b (1)/../ü":   "file:///tmp/a%20b%20(1)/../%C3%BC",
		"/":                   "file:///",
		"/x/./y//z/":          "file:///x/y/z",
		"/q?#%{}`<>\"\\!*'[]": "file:///q%3F%23%25%7B%7D%60%3C%3E%22%5C!*'[]",
	}
	for path, want := range cases {
		if got, ok := fileURLFromPath(path); !ok || got != want {
			t.Errorf("fileURLFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestClient_AttachClose(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" || os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("needs WAYLAND_DISPLAY and GPUI_TEST_DISPLAY=1; run under dbus-run-session")
	}
	loop, err := common.NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	linuxCommon, _, _ := common.NewLinuxCommon(loop.GetSignal())
	defer linuxCommon.Close()
	conn, err := Attach(loop.Handle(), linuxCommon, gpui.GraphicalEnvironmentDetect(), gpui.None[int](), gpui.None[string]())
	if err != nil {
		t.Fatal(err)
	}
	ptr := conn.ptr()
	state := conn.state()
	if len(state.registrations) == 0 {
		t.Fatal("Attach registered no event sources")
	}
	conn.Close()
	if _, ok := ptr.TryGetClient(); ok {
		t.Fatal("client state outlived Close")
	}
	if state.registrations != nil {
		t.Fatal("event sources still registered after Close")
	}
	conn.Close()
}
