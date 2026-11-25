package gpui

import (
	"slices"
	"testing"
)

type inputTestConfigurationView struct {
	EntityInputHandlerDefaults[inputTestConfigurationView]
	focusHandle      FocusHandle
	configuration    TextInputConfiguration
	acceptsTextInput bool
	handlerViews     *[]Entity[inputTestConfigurationView]
}

func newInputTestConfigurationView(cx *Context[inputTestConfigurationView], configuration TextInputConfiguration) *inputTestConfigurationView {
	v := &inputTestConfigurationView{
		focusHandle:      cx.FocusHandle(),
		configuration:    configuration,
		acceptsTextInput: true,
		handlerViews:     &[]Entity[inputTestConfigurationView]{},
	}
	BindEntityInputHandlerDefaults(&v.EntityInputHandlerDefaults, v)
	return v
}

func (v *inputTestConfigurationView) Render(_ *Window, cx *Context[inputTestConfigurationView]) IntoElement {
	view := cx.Entity()
	*v.handlerViews = append(*v.handlerViews, view)
	focusHandle := v.focusHandle
	return NewDiv().SizeFull().TrackFocus(v.focusHandle).Child(&windowTestCanvas{
		sizeFull: true,
		paint: func(bounds Bounds[Pixels], window *Window, cx *App) {
			window.HandleInput(focusHandle, NewElementInputHandler(bounds, view), cx)
		},
	})
}

func (v *inputTestConfigurationView) AcceptsTextInput(*Window, *Context[inputTestConfigurationView]) bool {
	return v.acceptsTextInput
}

func (v *inputTestConfigurationView) TextForRange(Range, *Option[Range], *Window, *Context[inputTestConfigurationView]) (string, bool) {
	return "", false
}

func (v *inputTestConfigurationView) SelectedTextRange(bool, *Window, *Context[inputTestConfigurationView]) (UTF16Selection, bool) {
	return UTF16Selection{}, false
}

func (v *inputTestConfigurationView) MarkedTextRange(*Window, *Context[inputTestConfigurationView]) (Range, bool) {
	return Range{}, false
}

func (v *inputTestConfigurationView) UnmarkText(*Window, *Context[inputTestConfigurationView]) {}

func (v *inputTestConfigurationView) ReplaceTextInRange(Option[Range], string, *Window, *Context[inputTestConfigurationView]) {
}

func (v *inputTestConfigurationView) ReplaceAndMarkTextInRange(Option[Range], string, Option[Range], *Window, *Context[inputTestConfigurationView]) {
}

func (v *inputTestConfigurationView) BoundsForRange(Range, Bounds[Pixels], *Window, *Context[inputTestConfigurationView]) (Bounds[Pixels], bool) {
	return Bounds[Pixels]{}, false
}

func (v *inputTestConfigurationView) CharacterIndexForPoint(Point[Pixels], *Window, *Context[inputTestConfigurationView]) (int, bool) {
	return 0, false
}

func (v *inputTestConfigurationView) TextInputConfiguration(*Window, *Context[inputTestConfigurationView]) TextInputConfiguration {
	return v.configuration
}

func inputTestReleaseHandlerViews(cx *TestAppContext, view Entity[inputTestConfigurationView]) {
	var views []Entity[inputTestConfigurationView]
	cx.Update(func(cx *App) { views = *view.Read(cx).handlerViews })
	for _, v := range views {
		v.Release()
	}
}

func inputTestRoot(cx *TestAppContext, window WindowHandle[inputTestConfigurationView]) Entity[inputTestConfigurationView] {
	view, err := window.Root(cx)
	windowTestMust(err)
	return view
}

func TestInput_TextInputConfigurationAndFocusStateAreForwardedOnChange(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		custom := TextInputConfiguration{Autocorrect: true, InputAction: TextInputActionSend}
		windowHandle := cx.AddWindow(func(_ *Window, cx *Context[inputTestConfigurationView]) *inputTestConfigurationView {
			return newInputTestConfigurationView(cx, custom)
		})
		view := inputTestRoot(cx, windowHandle)
		testWindow := cx.testWindow(windowHandle.AnyWindowHandle)
		window := windowHandle.AnyWindowHandle
		defer func() {
			var focusHandle FocusHandle
			cx.Update(func(cx *App) { focusHandle = view.Read(cx).focusHandle })
			inputTestReleaseHandlerViews(cx, view)
			view.Release()
			focusHandle.Release()
		}()
		draw := func() {
			divTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.Draw(cx).Clear(cx) })
		}
		checkConfigurations := func(want ...TextInputConfiguration) {
			t.Helper()
			if got := testWindow.TextInputConfigurations(); !slices.Equal(got, want) {
				t.Fatalf("configurations = %v, want %v", got, want)
			}
		}
		checkChanges := func(want ...TextInputStateChange) {
			t.Helper()
			if got := testWindow.TextInputStateChanges(); !slices.Equal(got, want) {
				t.Fatalf("state changes = %v, want %v", got, want)
			}
		}

		draw()
		checkConfigurations(TextInputConfiguration{})
		checkChanges()

		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			window.Focus(view.Read(cx).focusHandle, cx)
		})
		draw()
		checkConfigurations(TextInputConfiguration{}, custom)
		checkChanges(TextInputStateChangeFocusGained)

		draw()
		checkConfigurations(TextInputConfiguration{}, custom)
		checkChanges(TextInputStateChangeFocusGained)

		updated := custom
		updated.Suggestions = true
		view.Update(cx, func(view *inputTestConfigurationView, cx *Context[inputTestConfigurationView]) {
			view.configuration = updated
			cx.Notify()
		})
		draw()
		checkConfigurations(TextInputConfiguration{}, custom, updated)
		checkChanges(TextInputStateChangeFocusGained)

		divTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.Blur(cx) })
		draw()
		checkConfigurations(TextInputConfiguration{}, custom, updated, TextInputConfiguration{})
		checkChanges(TextInputStateChangeFocusGained, TextInputStateChangeFocusLost)
	})
}

func TestInput_TextNotificationsRequireAFocusedInputAndTrackFieldChanges(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		windowHandle := cx.AddWindow(func(_ *Window, cx *Context[inputTestConfigurationView]) *inputTestConfigurationView {
			return newInputTestConfigurationView(cx, TextInputConfiguration{})
		})
		view := inputTestRoot(cx, windowHandle)
		testWindow := cx.testWindow(windowHandle.AnyWindowHandle)
		window := windowHandle.AnyWindowHandle
		firstFocus := view.ReadWith(cx, func(view *inputTestConfigurationView, _ *App) FocusHandle { return view.focusHandle })
		var secondFocus FocusHandle
		defer func() {
			inputTestReleaseHandlerViews(cx, view)
			view.Release()
			firstFocus.Release()
			secondFocus.Release()
		}()
		checkChanges := func(want ...TextInputStateChange) {
			t.Helper()
			if got := testWindow.TextInputStateChanges(); !slices.Equal(got, want) {
				t.Fatalf("state changes = %v, want %v", got, want)
			}
		}

		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			window.Focus(firstFocus, cx)
			window.Draw(cx).Clear(cx)
			window.NotifyTextInputChanged(firstFocus)
			window.NotifyTextSelectionChanged(firstFocus)
		})

		installedHandler := testWindow.TakeInputHandler()
		if installedHandler == nil {
			t.Fatal("no input handler installed")
		}
		if !installedHandler.QueryAcceptsFocusedTextInput() {
			t.Fatal("handler should accept focused text input")
		}
		if id, ok := installedHandler.FocusID(); !ok || id != firstFocus.id {
			t.Fatalf("focus id = %v, %v", id, ok)
		}
		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			secondFocus = view.UpdateR(cx, func(view *inputTestConfigurationView, cx *Context[inputTestConfigurationView]) FocusHandle {
				view.focusHandle = cx.FocusHandle()
				cx.Notify()
				return view.focusHandle
			})
			window.Focus(secondFocus, cx)
		})
		if installedHandler.QueryAcceptsFocusedTextInput() {
			t.Fatal("stale handler should not accept focused text input")
		}
		testWindow.SetInputHandler(installedHandler)
		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.NotifyTextInputChanged(firstFocus)
		})
		checkChanges(
			TextInputStateChangeFocusGained,
			TextInputStateChangeContentChanged,
			TextInputStateChangeSelectionChanged,
		)

		installedHandler = testWindow.TakeInputHandler()
		if installedHandler == nil {
			t.Fatal("no input handler installed")
		}
		if id, ok := installedHandler.FocusID(); !ok || id != secondFocus.id {
			t.Fatalf("focus id = %v, %v", id, ok)
		}
		testWindow.SetInputHandler(installedHandler)
		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.NotifyTextSelectionChanged(secondFocus)
		})

		view.Update(cx, func(view *inputTestConfigurationView, cx *Context[inputTestConfigurationView]) {
			view.acceptsTextInput = false
			cx.Notify()
		})
		divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.NotifyTextInputChanged(secondFocus)
			window.NotifyTextSelectionChanged(secondFocus)
			ordinaryFocus := cx.FocusHandle()
			defer ordinaryFocus.Release()
			window.Focus(ordinaryFocus, cx)
			window.Draw(cx).Clear(cx)
			window.NotifyTextInputChanged(ordinaryFocus)
		})
		checkChanges(
			TextInputStateChangeFocusGained,
			TextInputStateChangeContentChanged,
			TextInputStateChangeSelectionChanged,
			TextInputStateChangeSelectionChanged,
			TextInputStateChangeFocusLost,
		)
	})
}
