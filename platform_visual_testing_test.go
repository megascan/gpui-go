package gpui

import (
	"runtime"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

func newVisualTestPlatformForTest() (*VisualTestPlatform, *TestPlatform) {
	dispatcher := NewTestDispatcher(0)
	inner := NewTestPlatform(NewBackgroundExecutor(dispatcher), NewForegroundExecutor(dispatcher))
	return NewVisualTestPlatform(inner, 7), inner
}

func TestVisualTestPlatform_DelegatesAndIsolates(t *testing.T) {
	p, inner := newVisualTestPlatformForTest()

	if p.Dispatcher() == nil || p.Dispatcher().Scheduler() == nil {
		t.Fatal("missing dispatcher")
	}
	if p.Dispatcher().SessionID() == p.ForegroundExecutor().inner.SessionID() {
		t.Fatal("executors must run on a cloned dispatcher session")
	}
	if len(p.Displays()) != len(inner.Displays()) {
		t.Fatal("Displays not delegated")
	}
	if p.TextSystem() != inner.TextSystem() {
		t.Fatal("TextSystem not delegated")
	}

	inner.WriteToClipboard(NewClipboardItemString("inner"))
	if _, ok := p.ReadFromClipboard(); ok {
		t.Fatal("clipboard must start empty")
	}
	p.WriteToClipboard(NewClipboardItemString("visual"))
	item, ok := p.ReadFromClipboard()
	if text, _ := item.Text(); !ok || text != "visual" {
		t.Fatalf("clipboard = %q, %v", text, ok)
	}
	if innerItem, _ := inner.ReadFromClipboard(); func() string { s, _ := innerItem.Text(); return s }() != "inner" {
		t.Fatal("inner clipboard was overwritten")
	}

	p.WriteToFindPasteboard(NewClipboardItemString("find"))
	_, ok = p.ReadFromFindPasteboard()
	if ok != (runtime.GOOS == "darwin") {
		t.Fatalf("find pasteboard ok = %v on %s", ok, runtime.GOOS)
	}

	if !p.CanSelectMixedFilesAndDirs() || p.ThermalState() != ThermalStateNominal {
		t.Fatal("constant answers differ from upstream")
	}
	if _, ok := p.GetMenus(); ok {
		t.Fatal("GetMenus must return none")
	}
}

func TestVisualTestPlatform_PromptsCredentialsAndSleep(t *testing.T) {
	p, _ := newVisualTestPlatformForTest()
	executor := p.ForegroundExecutor()
	p.Dispatcher().AllowParking()

	paths := executor.BlockTest(scheduler.Future[scheduler.Result[scheduler.Result[Option[[]string]]]](p.PromptForPaths(PathPromptOptions{})))
	if paths.Err != nil || paths.Value.Err != nil || paths.Value.Value.IsSome() {
		t.Fatalf("PromptForPaths = %+v", paths)
	}
	newPath := executor.BlockTest(scheduler.Future[scheduler.Result[scheduler.Result[Option[string]]]](p.PromptForNewPath("/tmp", None[string]())))
	if newPath.Err != nil || newPath.Value.Err != nil || newPath.Value.Value.IsSome() {
		t.Fatalf("PromptForNewPath = %+v", newPath)
	}

	if r := executor.BlockTest(scheduler.Future[scheduler.Result[struct{}]](p.WriteCredentials("u", "n", []byte("p")))); r.Err != nil {
		t.Fatal(r.Err)
	}
	creds := executor.BlockTest(scheduler.Future[scheduler.Result[Option[Credentials]]](p.ReadCredentials("u")))
	if creds.Err != nil || creds.Value.IsSome() {
		t.Fatalf("ReadCredentials = %+v", creds)
	}
	err := executor.BlockTest(scheduler.Future[scheduler.Result[*ActivityGuard]](p.PreventIdleSleep("render"))).Err
	if err == nil || err.Error() != `Idle sleep prevention for "render" is not supported in visual tests` {
		t.Fatalf("PreventIdleSleep error = %v", err)
	}

	defer func() {
		if r := recover(); r != "VisualTestPlatform::run should not be called in tests" {
			t.Fatalf("Run panic = %v", r)
		}
	}()
	p.Run(func() {})
}
