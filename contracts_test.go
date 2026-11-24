package gpui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/scheduler"
)

var (
	_ AppContext        = (*App)(nil)
	_ AppContext        = (*Context[counter])(nil)
	_ AppContext        = (*AsyncApp)(nil)
	_ AppContext        = AppRefMut{}
	_ BorrowAppContext  = (*App)(nil)
	_ BorrowAppContext  = (*Context[counter])(nil)
	_ scheduler.Awaiter = (*AsyncApp)(nil)

	_ InteractiveElement         = (*Div)(nil)
	_ Styler                     = (*Div)(nil)
	_ ParentElement              = (*Div)(nil)
	_ IntoElement                = (*Div)(nil)
	_ IntoElement                = (*Stateful[*Div])(nil)
	_ StatefulInteractiveElement = (*Stateful[*Div])(nil)
	_ IntoElement                = AnyElement{}
	_ IntoElement                = Entity[counter]{}

	_ Element[struct{}, int]            = (*quadElement)(nil)
	_ ElementA11y                       = (*quadElement)(nil)
	_ ElementA11ySyntheticChildren[int] = (*quadElement)(nil)
	_ Render[counter]                   = (*counter)(nil)
	_ RenderOnce                        = button{}
	_ Action                            = noop{}
	_ Focusable                         = (*counter)(nil)
)

type counter struct {
	count int
	focus FocusHandle
}

func (c *counter) Render(window *Window, cx *Context[counter]) IntoElement {
	return NewDiv().Flex().SizeFull().ChildText("count")
}

func (c *counter) FocusHandle(cx *App) FocusHandle {
	return c.focus.Clone()
}

type changed struct{ by int }

type button struct{ label string }

func (b button) Render(window *Window, cx *App) IntoElement {
	return NewDiv().ID("button").OnClick(func(*ClickEvent, *Window, *App) {}).ChildText(b.label)
}

type quadElement struct{ bounds Bounds[Pixels] }

func (q *quadElement) ElementID() (ElementID, bool)    { return ElementID{}, false }
func (q *quadElement) SourceLocation() *SourceLocation { return nil }

func (q *quadElement) RequestLayout(id *GlobalElementID, insp *InspectorElementID, w *Window, cx *App) (LayoutID, struct{}) {
	return w.RequestLayout(Style{}, nil, cx), struct{}{}
}

func (q *quadElement) Prepaint(id *GlobalElementID, insp *InspectorElementID, bounds Bounds[Pixels], rl *struct{}, w *Window, cx *App) int {
	w.InsertHitbox(bounds, HitboxBehavior(0))
	return 1
}

func (q *quadElement) Paint(id *GlobalElementID, insp *InspectorElementID, bounds Bounds[Pixels], rl *struct{}, pp *int, w *Window, cx *App) {
	w.PaintQuad(PaintQuad{})
}

func (q *quadElement) A11yRole() (a11y.Role, bool)                                 { return a11y.Role(0), false }
func (q *quadElement) WriteA11yInfo(node *a11y.Node)                               {}
func (q *quadElement) ElementA11ySyntheticChildren(pp *int, b *A11ySubtreeBuilder) {}

type noop struct{}

func (noop) Name() string                { return "test::Noop" }
func (noop) BoxedClone() Action          { return noop{} }
func (noop) PartialEq(other Action) bool { _, ok := other.(noop); return ok }

type settings struct{ theme string }

var _ = func(cx *App, acx *AsyncApp, vcx VisualContext, cell *AppCell, t *testing.T) {
	e := cx.New(func(cx *Context[counter]) *counter { return &counter{} })
	defer e.Release()
	var _ Entity[counter] = AppContextNew(cx, func(cx *Context[counter]) *counter { return &counter{} })
	var _ Entity[counter] = acx.New(func(cx *Context[counter]) *counter { return &counter{} })

	r := AppContextReserveEntity[counter](cx)
	var _ EntityID = r.EntityID()
	var _ Entity[counter] = AppContextInsertEntity(cx, r, func(cx *Context[counter]) *counter { return &counter{} })
	var _ Reservation[counter] = cx.ReserveEntity[counter]()

	e.Update(cx, func(c *counter, cx *Context[counter]) {
		c.count++
		cx.Notify()
		cx.Emit(changed{by: 1})
	})
	var _ int = e.UpdateR(cx, func(c *counter, cx *Context[counter]) int { return c.count })
	var _ int = AppContextUpdateEntity(acx, e, func(c *counter, cx *Context[counter]) int { return c.count })
	var _ int = AppContextReadEntity(cx, e, func(c *counter, cx *App) int { return c.count })
	var _ int = e.ReadWith(cx, func(c *counter, cx *App) int { return c.count })
	var _ *counter = e.Read(cx)
	var _ EntityID = e.EntityID()
	var _ AnyEntity = e.IntoAny()
	var _ error = e.UpdateIn(vcx, func(c *counter, w *Window, cx *Context[counter]) {})
	var _, _ = e.UpdateInR(vcx, func(c *counter, w *Window, cx *Context[counter]) int { return 0 })

	weak := e.Downgrade()
	if strong, ok := weak.Upgrade(); ok {
		strong.Release()
	}
	var _ error = weak.Update(cx, func(c *counter, cx *Context[counter]) {})

	b := AppContextAsMut(cx, e)
	b.Get().count++
	b.Release()

	var _, _ = AppContextUpdateWindow(cx, AnyWindowHandle{}, func(root AnyView, w *Window, cx *App) int { return 0 })
	var _, _ = AppContextWithWindow(cx, e.EntityID(), func(w *Window, cx *App) bool { return true })
	var _, _ = AppContextReadWindow(cx, WindowHandle[counter]{}, func(root Entity[counter], cx *App) int { return 0 })
	var _ string = AppContextReadGlobal(cx, func(s *settings, cx *App) string { return s.theme })
	var _ string = cx.ReadGlobal(func(s *settings, cx *App) string { return s.theme })

	task := AppContextBackgroundSpawn(cx, func(s *scheduler.Scope) (int, error) { return 42, nil })
	var _ *Task[int] = cx.BackgroundSpawn(func(s *scheduler.Scope) (int, error) { return 0, nil })
	fg := cx.Spawn(func(acx *AsyncApp) (int, error) {
		v, err := task.Await(acx)
		acx.Update(func(cx *App) {})
		return v + acx.UpdateR(func(cx *App) int { return 1 }), err
	})
	fg.Detach()

	SetGlobal(cx, settings{theme: "dark"})
	UpdateGlobal(cx, func(s *settings, cx *App) { s.theme = "light" })
	var _ int = UpdateGlobalR(cx, func(s *settings, cx *App) int { return len(s.theme) })
	UpdateDefaultGlobal(cx, func(s *settings, cx *App) {})

	sub := cx.Observe(e, func(e Entity[counter], cx *App) {})
	sub.Detach()
	cx.Defer(func(cx *App) {})

	var _ Entity[counter] = cx.New(func(ccx *Context[counter]) *counter {
		ccx.Observe(e, func(this *counter, e Entity[counter], cx *Context[counter]) {}).Unsubscribe()
		ccx.Subscribe(e, func(this *counter, e Entity[counter], ev *changed, cx *Context[counter]) {}).Detach()
		ccx.Spawn(func(this WeakEntity[counter], acx *AsyncApp) (struct{}, error) {
			return struct{}{}, this.Update(acx, func(c *counter, cx *Context[counter]) {})
		}).Detach()
		var _ func(*ClickEvent, *Window, *App) = ccx.Listener(func(this *counter, ev *ClickEvent, w *Window, cx *Context[counter]) {})
		UpdateGlobal(ccx, func(s *settings, cx *Context[counter]) {})
		var _ Entity[counter] = ccx.New(func(cx *Context[counter]) *counter { return &counter{} })
		return &counter{}
	})

	var _, _ = VisualContextNewWindowEntity(vcx, func(w *Window, cx *Context[counter]) *counter { return &counter{} })
	var _, _ = VisualContextUpdateWindowEntity(vcx, e, func(c *counter, w *Window, cx *Context[counter]) int { return 0 })
	var _, _ = VisualContextReplaceRootView(vcx, func(w *Window, cx *Context[counter]) *counter { return &counter{} })
	var _ error = VisualContextFocus(vcx, e)
	var _ AnyWindowHandle = vcx.WindowHandle()

	var _, _ = cx.OpenWindow(WindowOptions{}, func(w *Window, cx *App) Entity[counter] {
		return cx.New(func(cx *Context[counter]) *counter { return &counter{} })
	})

	ref := cell.Borrow()
	var _ *App = ref.Get()
	ref.Release()
	mut := cell.BorrowMut()
	defer mut.Release()
	if m, err := cell.TryBorrowMut(); err == nil {
		m.Release()
	}

	var w *Window
	var _ ArenaClearNeeded = w.Draw(cx)
	var _ int = w.WithElementState(&GlobalElementID{}, func(state Option[int], w *Window) (int, int) {
		v, _ := state.Get()
		return v, v + 1
	})
	var _ bool = w.DispatchKeystroke(Keystroke{Key: "a", Modifiers: Modifiers{Control: true}}, cx)
	var _ DispatchEventResult = w.DispatchEvent(nil, cx)
	w.Refresh()
	w.OnNextFrame(func(w *Window, cx *App) {})
	w.RequestAnimationFrame()
	var _ ContentMask[Pixels] = ContentMask[Pixels]{Bounds: Bounds[Pixels]{}}
	var _ DispatchPhase = DispatchPhaseCapture

	var d *Div = NewDiv().Flex().SizeFull()
	var s *Stateful[*Div] = d.ID("root").OnClick(func(*ClickEvent, *Window, *App) {}).Flex()
	s = s.Child(button{label: "x"}.Render(w, cx)).Children(SharedString("a"), NewDiv())
	var _ *Div = d.When(true, func(d *Div) *Div { return d.SizeFull() })
	var _ int = d.Map(func(d *Div) int { return 0 })
	any := NewAnyElement(&quadElement{})
	any.RequestLayout(w, cx)
	var _ Size[Pixels] = any.LayoutAsRoot(SizeOf[AvailableSpace]{}, w, cx)
	any.PrepaintAt(Point[Pixels]{}, w, cx)
	any.Paint(w, cx)
	var _ AnyElement = s.IntoAnyElement()
	var _ AnyElement = e.IntoAnyElement()

	var _, _ = KeystrokeParse("ctrl-a")
	RegisterAction(ActionData{
		Name: "test::Noop",
		Type: reflect.TypeFor[noop](),
		Build: func(value jsontext.Value) (Action, error) {
			return noop{}, nil
		},
	})

	var sc *Scene
	for batch := range sc.Batches() {
		_ = batch
	}

	RunTest(t, func(tc *TestRun) {
		var _ *TestAppContext = tc.Cx()
		var _ *TestAppContext = tc.NewContext()
		_ = tc.Rand().Uint64()
		var _ BackgroundExecutor = tc.BackgroundExecutor()
	}, Seed(1), Iterations(10), Retries(2), OnFailure(func() {}), Seeds(1, 2))
}

func TestContracts_MixinBinding(t *testing.T) {
	d := &Div{}
	BindStyled(&d.Styled, d)
	BindInteractive(&d.Interactive, d)
	BindInteractiveID(&d.InteractiveID, d)
	BindParent(&d.Parent, d)
	BindFluent(&d.Fluent, d)
	if d.Styled.self != d || d.Interactive.self != d || d.InteractiveID.self != d || d.Parent.self != d || d.Fluent.self != d {
		t.Fatal("mixins not bound to their element")
	}
	s := &Stateful[*Div]{element: d}
	BindStyled(&s.Styled, s)
	BindStatefulInteractive(&s.StatefulInteractive, s)
	if s.Styled.self != s || s.StatefulInteractive.self != s {
		t.Fatal("Stateful mixins not bound")
	}
}

func TestContracts_SealedInterfaces(t *testing.T) {
	ac := reflect.TypeFor[AppContext]()
	for i := range ac.NumMethod() {
		if m := ac.Method(i); m.IsExported() {
			t.Errorf("AppContext exposes exported primitive %s", m.Name)
		}
	}
	vc := reflect.TypeFor[VisualContext]()
	exported := 0
	for i := range vc.NumMethod() {
		if vc.Method(i).IsExported() {
			exported++
		}
	}
	if exported != 1 {
		t.Errorf("VisualContext has %d exported methods, want 1 (WindowHandle)", exported)
	}
	if reflect.TypeFor[Task[int]]() != reflect.TypeFor[scheduler.Task[int]]() {
		t.Error("Task is not an alias of scheduler.Task")
	}
	if reflect.TypeFor[Role]() != reflect.TypeFor[a11y.Role]() || reflect.TypeFor[AccessibleAction]() != reflect.TypeFor[a11y.Action]() {
		t.Error("a11y aliases differ from package a11y")
	}
	shared := reflect.TypeFor[AppRef]()
	if shared.Implements(ac) || shared.Implements(reflect.TypeFor[BorrowAppContext]()) {
		t.Error("AppRef, a shared borrow, implements a mutable context interface")
	}
	if shared.NumMethod() != 2 {
		t.Errorf("AppRef has %d methods, want 2 (Get, Release)", shared.NumMethod())
	}
	mut := reflect.TypeFor[AppRefMut]()
	if !mut.Implements(ac) || !mut.Implements(reflect.TypeFor[BorrowAppContext]()) {
		t.Error("AppRefMut does not reach the App mutably")
	}
}

func TestContracts_GPUSpecsJSON(t *testing.T) {
	specs := GPUSpecs{IsSoftwareEmulated: true, DeviceName: "llvmpipe", DriverName: "llvmpipe", DriverInfo: "Mesa"}
	got, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"is_software_emulated":true,"device_name":"llvmpipe","driver_name":"llvmpipe","driver_info":"Mesa"}`
	if string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	var back GPUSpecs
	if err := json.Unmarshal(got, &back); err != nil {
		t.Fatal(err)
	}
	if back != specs {
		t.Fatalf("round trip: got %+v", back)
	}
}
