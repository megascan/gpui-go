//go:build linux

package wayland

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/scheduler"
)

const unwrapNone = "called `Option::unwrap()` on a `None` value"

const knownSeatCapabilities = wlproto.WlSeatCapabilityPointer | wlproto.WlSeatCapabilityKeyboard | wlproto.WlSeatCapabilityTouch

func (p WaylandClientStatePtr) seatListener(seat *wlproto.WlSeat) wlproto.WlSeatListener {
	return wlproto.WlSeatListener{
		Capabilities: func(capabilities wlproto.WlSeatCapability) {
			if capabilities&^knownSeatCapabilities != 0 {
				return
			}
			state := p.GetClient()
			if capabilities&wlproto.WlSeatCapabilityKeyboard != 0 {
				keyboard := seat.GetKeyboard()
				keyboard.SetListener(p.keyboardListener())

				if state.textInput != nil {
					textInput := state.textInput
					state.textInput = nil
					textInput.Destroy()
					state.imePreEdit = gpui.None[string]()
					state.composing = false
				}

				state.textInput = nil
				if manager := state.globals.TextInputManager; manager != nil {
					textInput := manager.GetTextInput(seat)
					textInput.SetListener(p.textInputListener(textInput))
					state.textInput = textInput
				}

				if state.wlKeyboard != nil {
					state.wlKeyboard.Release()
				}

				state.wlKeyboard = keyboard
			}
			if capabilities&wlproto.WlSeatCapabilityPointer != 0 {
				pointer := seat.GetPointer()
				pointer.SetListener(p.pointerListener(pointer))

				if state.cursorShapeDevice != nil {
					device := state.cursorShapeDevice
					state.cursorShapeDevice = nil
					device.Destroy()
				}

				state.cursorShapeDevice = nil
				if manager := state.globals.CursorShapeManager; manager != nil {
					state.cursorShapeDevice = manager.GetPointer(pointer)
				}

				state.pinchGesture = nil
				if manager := state.globals.GestureManager; manager != nil {
					pinch := manager.GetPinchGesture(pointer)
					pinch.SetListener(p.pinchGestureListener())
					state.pinchGesture = pinch
				}

				if state.wlPointer != nil {
					state.wlPointer.Release()
				}

				state.wlPointer = pointer
			}
		},
	}
}

func (p WaylandClientStatePtr) keyboardListener() wlproto.WlKeyboardListener {
	return wlproto.WlKeyboardListener{
		RepeatInfo: func(rate, delay int32) {
			state := p.GetClient()
			state.repeat.charactersPerSecond = uint32(rate)
			state.repeat.delay = time.Duration(uint64(delay)) * time.Millisecond
		},
		Keymap: func(format wlproto.WlKeyboardKeymapFormat, fd int, size uint32) {
			if format != wlproto.WlKeyboardKeymapFormatNoKeymap && format != wlproto.WlKeyboardKeymapFormatXkbV1 {
				unix.Close(fd)
				return
			}
			state := p.GetClient()
			if format != wlproto.WlKeyboardKeymapFormatXkbV1 {
				unix.Close(fd)
				slog.Error(fmt.Sprintf("Received keymap format %v, expected XkbV1", format))
				return
			}
			xkbContext, err := common.NewXkbContext()
			if err != nil {
				unix.Close(fd)
				slog.Error(fmt.Sprintf("Failed to process Wayland keymap: %v", err))
				return
			}
			defer xkbContext.Close()
			keymap, err := xkb.NewKeymapFromFd(xkbContext, fd, int(size), xkb.KeymapFormatTextV1, xkb.KeymapCompileNoFlags)
			if err != nil {
				slog.Error(err.Error())
			}
			if keymap == nil {
				panic("Failed to create keymap")
			}
			if state.keymapState != nil {
				state.keymapState.Close()
			}
			state.keymapState = xkb.NewState(keymap)
			keymap.Close()
			if state.composeState != nil {
				state.composeState.Close()
			}
			state.composeState = nil
			if compose, ok := common.GetXkbComposeState(xkbContext); ok {
				state.composeState = compose
			}

			p.HandleKeyboardLayoutChange()
		},
		Enter: func(serial uint32, surface *wlproto.WlSurface, keys []byte) {
			state := p.GetClient()
			state.keyboardFocusedWindow = surfaceWindow(state, surface)
			state.enterToken = true

			if window, ok := state.keyboardFocusedWindow.Get(); ok {
				window.SetFocused(true)
			}
		},
		Leave: func(serial uint32, surface *wlproto.WlSurface) {
			state := p.GetClient()
			keyboardFocusedWindow := surfaceWindow(state, surface)
			state.keyboardFocusedWindow = gpui.None[WaylandWindowStatePtr]()
			state.enterToken = false
			state.repeat.currentID++
			state.restoreCursorAfterHide()

			if window, ok := keyboardFocusedWindow.Get(); ok {
				if state.composeState != nil {
					state.composeState.Reset()
				}
				state.preEditText = gpui.None[string]()
				window.HandleIME(IMEInputDeleteText{})
				window.SetFocused(false)
			}
		},
		Modifiers: func(serial, modsDepressed, modsLatched, modsLocked, group uint32) {
			state := p.GetClient()
			focusedWindow := state.keyboardFocusedWindow

			keymapState := state.keymapState
			if keymapState == nil {
				panic(unwrapNone)
			}
			oldLayout := keymapState.SerializeLayout(xkb.StateLayoutEffective)
			keymapState.UpdateMask(xkb.ModMask(modsDepressed), xkb.ModMask(modsLatched), xkb.ModMask(modsLocked), 0, 0, xkb.LayoutIndex(group))
			state.Modifiers = common.ModifiersFromXkb(keymapState)
			state.Capslock = common.CapslockFromXkb(keymapState)

			input := gpui.ModifiersChangedEvent{
				Modifiers: state.Modifiers,
				Capslock:  state.Capslock,
			}

			if window, ok := focusedWindow.Get(); ok {
				window.HandleInput(input)
			}

			if xkb.LayoutIndex(group) != oldLayout {
				p.HandleKeyboardLayoutChange()
			}
		},
		Key: func(serial, time_, key uint32, keyState wlproto.WlKeyboardKeyState) {
			if keyState > wlproto.WlKeyboardKeyStateRepeated {
				return
			}
			p.handleKey(serial, key, keyState)
		},
	}
}

func (p WaylandClientStatePtr) handleKey(serial, key uint32, keyState wlproto.WlKeyboardKeyState) {
	state := p.GetClient()
	if keyState == wlproto.WlKeyboardKeyStatePressed {
		var focusedSurface *wlproto.WlSurface
		if window, ok := state.keyboardFocusedWindow.Get(); ok {
			focusedSurface = window.Surface()
		}
		state.serialTracker.UpdateWithFocusedSurface(SerialKindKeyPress, serial, focusedSurface)
	}

	focusedWindow, ok := state.keyboardFocusedWindow.Get()
	if !ok {
		return
	}

	keymapState := state.keymapState
	if keymapState == nil {
		panic(unwrapNone)
	}
	keycode := xkb.Keycode(key + minKeycode)
	keysym := keymapState.KeyGetOneSym(keycode)

	switch {
	case keyState == wlproto.WlKeyboardKeyStatePressed && !keysym.IsModifierKey():
		keystroke := common.KeystrokeFromXkb(keymapState, state.Modifiers, keycode)
		if compose := state.composeState; compose != nil {
			state.composeState = nil
			compose.Feed(keysym)
			switch compose.Status() {
			case xkb.StatusComposing:
				keystroke.KeyChar = gpui.None[string]()
				if text, ok := compose.Utf8(); ok {
					state.preEditText = gpui.Some(text)
				} else {
					state.preEditText = gpui.OptionFrom(common.KeystrokeUnderlyingDeadKey(keysym))
				}
				preEdit := state.preEditText.UnwrapOrDefault()
				focusedWindow.HandleIME(IMEInputSetMarkedText{Text: preEdit})
				state = p.GetClient()
			case xkb.StatusComposed:
				state.preEditText = gpui.None[string]()
				keystroke.KeyChar = gpui.OptionFrom(compose.Utf8())
				if composed, ok := compose.Keysym(); ok {
					keystroke.Key = xkb.KeysymGetName(composed)
				}
			case xkb.StatusCancelled:
				preEdit := state.preEditText.Take()
				newPreEdit := gpui.OptionFrom(common.KeystrokeUnderlyingDeadKey(keysym))
				state.preEditText = newPreEdit
				if text, ok := preEdit.Get(); ok {
					focusedWindow.HandleIME(IMEInputInsertText{Text: text})
				}
				if currentKey, ok := newPreEdit.Get(); ok {
					focusedWindow.HandleIME(IMEInputSetMarkedText{Text: currentKey})
				}
				compose.Feed(keysym)
				state = p.GetClient()
			}
			state.composeState = compose
		}
		input := gpui.KeyDownEvent{
			Keystroke:            keystroke,
			IsHeld:               false,
			PreferCharacterInput: false,
		}

		state.repeat.currentID++
		state.repeat.currentKeycode = gpui.Some(keycode)

		rate := state.repeat.charactersPerSecond
		repeatInterval := time.Second / time.Duration(max(rate, 1))
		id := state.repeat.currentID
		heldInput := gpui.KeyDownEvent{
			Keystroke:            keystroke,
			IsHeld:               true,
			PreferCharacterInput: false,
		}
		state.loopHandle.InsertTimerAfter(state.repeat.delay, func(eventTimestamp time.Time) common.TimeoutAction {
			client, ok := p.TryGetClient()
			if !ok {
				return common.TimeoutActionDrop()
			}
			isRepeating := id == client.repeat.currentID &&
				client.repeat.currentKeycode.IsSome() &&
				client.keyboardFocusedWindow.IsSome()

			if !isRepeating || rate == 0 {
				return common.TimeoutActionDrop()
			}

			focusedWindow := client.keyboardFocusedWindow.Unwrap()

			focusedWindow.HandleInput(heldInput)

			return common.TimeoutActionToInstant(eventTimestamp.Add(repeatInterval))
		})

		focusedWindow.HandleInput(input)
	case keyState == wlproto.WlKeyboardKeyStateReleased && !keysym.IsModifierKey():
		input := gpui.KeyUpEvent{
			Keystroke: common.KeystrokeFromXkb(keymapState, state.Modifiers, keycode),
		}

		if state.repeat.currentKeycode == gpui.Some(keycode) {
			state.repeat.currentKeycode = gpui.None[xkb.Keycode]()
		}

		focusedWindow.HandleInput(input)
	}
}

func surfaceWindow(state *WaylandClientState, surface *wlproto.WlSurface) gpui.Option[WaylandWindowStatePtr] {
	if surface == nil {
		return gpui.None[WaylandWindowStatePtr]()
	}
	return gpui.OptionFrom(getWindow(state, &surface.Proxy))
}

func (p WaylandClientStatePtr) textInputListener(textInput *wlproto.ZwpTextInputV3) wlproto.ZwpTextInputV3Listener {
	return wlproto.ZwpTextInputV3Listener{
		Enter: func(*wlproto.WlSurface) {
			p.GetClient()
			p.EnableIME()
		},
		Leave: func(*wlproto.WlSurface) {
			p.GetClient()
			p.DisableIME()
		},
		CommitString: func(text gpui.Option[string]) {
			state := p.GetClient()
			state.composing = false
			window, ok := state.keyboardFocusedWindow.Get()
			if !ok {
				return
			}

			if commitText, ok := text.Get(); ok {
				if len(commitText) == 1 {
					window.HandleInput(gpui.KeyDownEvent{
						Keystroke: gpui.Keystroke{
							Modifiers: gpui.Modifiers{},
							Key:       commitText,
							KeyChar:   gpui.Some(commitText),
						},
						IsHeld:               false,
						PreferCharacterInput: false,
					})
				} else {
					window.HandleIME(IMEInputInsertText{Text: commitText})
				}
			}
		},
		PreeditString: func(text gpui.Option[string], cursorBegin, cursorEnd int32) {
			state := p.GetClient()
			state.composing = true
			state.imePreEdit = text
		},
		Done: func(serial uint32) {
			state := p.GetClient()
			lastSerial := state.serialTracker.Get(SerialKindInputMethod)
			state.serialTracker.Update(SerialKindInputMethod, serial)
			window, ok := state.keyboardFocusedWindow.Get()
			if !ok {
				return
			}

			if text, ok := state.imePreEdit.Take().Get(); ok {
				window.HandleIME(IMEInputSetMarkedText{Text: text})
				if area, ok := window.GetIMEArea(); ok {
					state := p.GetClient()
					setIMECursorRectangleAfterDone(textInputSink{textInput}, &state.lastIMECursorRectangle, area, lastSerial.AsRaw() == serial)
				}
			} else {
				state.composing = false
				window.HandleIME(IMEInputDeleteText{})
			}
		},
	}
}

func linuxButtonToGPUI(button uint32) (gpui.MouseButton, bool) {
	const (
		btnLeft    uint32 = 0x110
		btnRight   uint32 = 0x111
		btnMiddle  uint32 = 0x112
		btnSide    uint32 = 0x113
		btnExtra   uint32 = 0x114
		btnForward uint32 = 0x115
		btnBack    uint32 = 0x116
	)

	switch button {
	case btnLeft:
		return gpui.MouseButtonLeft, true
	case btnRight:
		return gpui.MouseButtonRight, true
	case btnMiddle:
		return gpui.MouseButtonMiddle, true
	case btnBack, btnSide:
		return gpui.MouseButtonNavigate(gpui.NavigationDirectionBack), true
	case btnForward, btnExtra:
		return gpui.MouseButtonNavigate(gpui.NavigationDirectionForward), true
	}
	return 0, false
}

func isKnownAxis(axis wlproto.WlPointerAxis) bool {
	return axis == wlproto.WlPointerAxisVerticalScroll || axis == wlproto.WlPointerAxisHorizontalScroll
}

func (s *WaylandClientState) scrollAxis(axis wlproto.WlPointerAxis) wlproto.WlPointerAxis {
	if s.Modifiers.Shift {
		return wlproto.WlPointerAxisHorizontalScroll
	}
	return axis
}

func (s *WaylandClientState) axisModifier(axis wlproto.WlPointerAxis) float32 {
	switch axis {
	case wlproto.WlPointerAxisVerticalScroll:
		return s.verticalModifier
	case wlproto.WlPointerAxisHorizontalScroll:
		return s.horizontalModifier
	}
	return 1.0
}

func (p WaylandClientStatePtr) pointerListener(wlPointer *wlproto.WlPointer) wlproto.WlPointerListener {
	return wlproto.WlPointerListener{
		Enter: func(serial uint32, surface *wlproto.WlSurface, surfaceX, surfaceY float64) {
			state := p.GetClient()
			position := gpui.NewPoint(gpui.Px(float32(surfaceX)), gpui.Px(float32(surfaceY)))
			state.serialTracker.Update(SerialKindMouseEnter, serial)
			state.MouseLocation = gpui.Some(position)
			state.buttonPressed = gpui.None[gpui.MouseButton]()

			window, ok := surfaceWindow(state, surface).Get()
			if !ok {
				return
			}
			state.mouseFocusedWindow = gpui.Some(window)

			state.enterToken = false
			state.restoreCursorAfterHide()
			if style, ok := state.cursorStyle.Get(); ok {
				if state.cursorShapeDevice != nil {
					state.cursorShapeDevice.SetShape(serial, toShape(style))
				} else {
					scale := window.PrimaryOutputScale()
					state.cursor.SetIcon(wlPointer, serial, common.CursorStyleToIconNames(style), scale)
				}
			}
			modifiers := state.Modifiers
			window.SetHovered(true)
			window.HandleInput(gpui.MouseMoveEvent{
				Position:      position,
				PressedButton: gpui.None[gpui.MouseButton](),
				Modifiers:     modifiers,
			})
		},
		Leave: func(serial uint32, surface *wlproto.WlSurface) {
			state := p.GetClient()
			focusedWindow, ok := state.mouseFocusedWindow.Get()
			if !ok {
				return
			}
			input := gpui.MouseExitEvent{
				Position:      state.MouseLocation.Unwrap(),
				PressedButton: state.buttonPressed,
				Modifiers:     state.Modifiers,
			}
			state.mouseFocusedWindow = gpui.None[WaylandWindowStatePtr]()
			state.MouseLocation = gpui.None[gpui.Point[gpui.Pixels]]()
			state.buttonPressed = gpui.None[gpui.MouseButton]()
			state.cursorHiddenWindow = gpui.None[WaylandWindowStatePtr]()

			focusedWindow.HandleInput(input)
			focusedWindow.SetHovered(false)
		},
		Motion: func(time_ uint32, surfaceX, surfaceY float64) {
			state := p.GetClient()
			if state.mouseFocusedWindow.IsNone() {
				return
			}
			state.MouseLocation = gpui.Some(gpui.NewPoint(gpui.Px(float32(surfaceX)), gpui.Px(float32(surfaceY))))
			state.restoreCursorAfterHide()

			window, ok := state.mouseFocusedWindow.Get()
			if !ok {
				return
			}
			if window.IsBlocked() {
				defaultStyle := gpui.CursorStyleArrow
				if state.cursorStyle != gpui.Some(defaultStyle) {
					serial := state.serialTracker.Get(SerialKindMouseEnter)
					state.cursorStyle = gpui.Some(defaultStyle)

					if state.cursorShapeDevice != nil {
						state.cursorShapeDevice.SetShape(serial.AsRaw(), toShape(defaultStyle))
					} else {
						pointer := state.wlPointer
						if pointer == nil {
							panic("window is focused by pointer")
						}
						scale := window.PrimaryOutputScale()
						state.cursor.SetIcon(pointer, serial.AsRaw(), common.CursorStyleToIconNames(defaultStyle), scale)
					}
				}
			}
			if keyboardWindow, ok := state.keyboardFocusedWindow.Get(); ok && window.PtrEq(keyboardWindow) {
				state.enterToken = false
			}
			input := gpui.MouseMoveEvent{
				Position:      state.MouseLocation.Unwrap(),
				PressedButton: state.buttonPressed,
				Modifiers:     state.Modifiers,
			}
			window.HandleInput(input)
		},
		Button: func(serial, time_, button uint32, buttonState wlproto.WlPointerButtonState) {
			if buttonState != wlproto.WlPointerButtonStatePressed && buttonState != wlproto.WlPointerButtonStateReleased {
				return
			}
			p.handleButton(serial, button, buttonState)
		},
		AxisSource: func(axisSource wlproto.WlPointerAxisSource) {
			if axisSource > wlproto.WlPointerAxisSourceWheelTilt {
				return
			}
			p.GetClient().axisSource = axisSource
		},
		Axis: func(time_ uint32, axis wlproto.WlPointerAxis, value float64) {
			if !isKnownAxis(axis) {
				return
			}
			state := p.GetClient()
			if state.axisSource == wlproto.WlPointerAxisSourceWheel {
				return
			}
			axis = state.scrollAxis(axis)
			axisModifier := state.axisModifier(axis)
			state.scrollEventReceived = true
			if state.continuousScrollDelta.IsNone() {
				state.continuousScrollDelta = gpui.Some(gpui.NewPoint(gpui.Px(0), gpui.Px(0)))
			}
			scrollDelta := state.continuousScrollDelta.AsMut()
			const modifier float32 = 3.0
			switch axis {
			case wlproto.WlPointerAxisVerticalScroll:
				scrollDelta.Y += gpui.Px(float32(value) * modifier * axisModifier)
			case wlproto.WlPointerAxisHorizontalScroll:
				scrollDelta.X += gpui.Px(float32(value) * modifier * axisModifier)
			default:
				panic("internal error: entered unreachable code")
			}
		},
		AxisDiscrete: func(axis wlproto.WlPointerAxis, discrete int32) {
			if !isKnownAxis(axis) {
				return
			}
			state := p.GetClient()
			state.scrollEventReceived = true
			axis = state.scrollAxis(axis)
			axisModifier := state.axisModifier(axis)

			scrollDelta := state.discreteScrollDeltaOrInsert()
			switch axis {
			case wlproto.WlPointerAxisVerticalScroll:
				scrollDelta.Y += float32(discrete) * axisModifier * common.ScrollLines
			case wlproto.WlPointerAxisHorizontalScroll:
				scrollDelta.X += float32(discrete) * axisModifier * common.ScrollLines
			default:
				panic("internal error: entered unreachable code")
			}
		},
		AxisValue120: func(axis wlproto.WlPointerAxis, value120 int32) {
			if !isKnownAxis(axis) {
				return
			}
			state := p.GetClient()
			state.scrollEventReceived = true
			axis = state.scrollAxis(axis)
			axisModifier := state.axisModifier(axis)

			scrollDelta := state.discreteScrollDeltaOrInsert()
			wheelPercent := float32(value120) / 120.0
			switch axis {
			case wlproto.WlPointerAxisVerticalScroll:
				scrollDelta.Y += wheelPercent * axisModifier * common.ScrollLines
			case wlproto.WlPointerAxisHorizontalScroll:
				scrollDelta.X += wheelPercent * axisModifier * common.ScrollLines
			default:
				panic("internal error: entered unreachable code")
			}
		},
		Frame: func() {
			state := p.GetClient()
			if !state.scrollEventReceived {
				return
			}
			state.scrollEventReceived = false
			continuous := state.continuousScrollDelta.Take()
			discrete := state.discreteScrollDelta.Take()
			if continuous, ok := continuous.Get(); ok {
				if window, ok := state.mouseFocusedWindow.Get(); ok {
					input := gpui.ScrollEvent{
						Position:   state.MouseLocation.Unwrap(),
						Delta:      gpui.ScrollDeltaPixels(continuous),
						Modifiers:  state.Modifiers,
						TouchPhase: gpui.TouchPhaseMoved,
						Source:     gpui.ScrollSourcePointer,
					}
					window.HandleInput(input)
				}
			} else if discrete, ok := discrete.Get(); ok {
				if window, ok := state.mouseFocusedWindow.Get(); ok {
					input := gpui.ScrollEvent{
						Position:   state.MouseLocation.Unwrap(),
						Delta:      gpui.ScrollDeltaLines(discrete),
						Modifiers:  state.Modifiers,
						TouchPhase: gpui.TouchPhaseMoved,
						Source:     gpui.ScrollSourcePointer,
					}
					window.HandleInput(input)
				}
			}
		},
	}
}

func (s *WaylandClientState) discreteScrollDeltaOrInsert() *gpui.Point[float32] {
	if s.discreteScrollDelta.IsNone() {
		s.discreteScrollDelta = gpui.Some(gpui.NewPoint[float32](0.0, 0.0))
	}
	return s.discreteScrollDelta.AsMut()
}

func (p WaylandClientStatePtr) handleButton(serial, rawButton uint32, buttonState wlproto.WlPointerButtonState) {
	state := p.GetClient()
	if buttonState == wlproto.WlPointerButtonStatePressed {
		var focusedSurface *wlproto.WlSurface
		if window, ok := state.mouseFocusedWindow.Get(); ok {
			focusedSurface = window.Surface()
		}
		state.serialTracker.UpdateWithFocusedSurface(SerialKindMousePress, serial, focusedSurface)
	}
	button, ok := linuxButtonToGPUI(rawButton)
	if !ok {
		return
	}
	if state.mouseFocusedWindow.IsNone() {
		return
	}
	switch buttonState {
	case wlproto.WlPointerButtonStatePressed:
		if window, ok := state.keyboardFocusedWindow.Get(); ok {
			if state.composing && state.textInput != nil {
				p.DisableIME()
				p.EnableIME()
				window.HandleIME(IMEInputUnmarkText{})
				state = p.GetClient()
			} else if text, ok := state.preEditText.Get(); ok && state.composeState != nil {
				state.preEditText = gpui.None[string]()
				state.composeState.Reset()
				window.HandleIME(IMEInputInsertText{Text: text})
				state = p.GetClient()
			}
		}
		clickElapsed := time.Since(state.click.lastClick)

		if clickElapsed < common.DoubleClickInterval &&
			state.click.lastMouseButton == gpui.Some(button) &&
			common.IsWithinClickDistance(state.click.lastLocation, state.MouseLocation.Unwrap()) {
			state.click.currentCount++
		} else {
			state.click.currentCount = 1
		}

		state.click.lastClick = time.Now()
		state.click.lastMouseButton = gpui.Some(button)
		state.click.lastLocation = state.MouseLocation.Unwrap()

		state.buttonPressed = gpui.Some(button)

		if window, ok := state.mouseFocusedWindow.Get(); ok {
			firstMouse := state.enterToken
			state.enterToken = false
			input := gpui.MouseDownEvent{
				Button:     button,
				Position:   state.MouseLocation.Unwrap(),
				Modifiers:  state.Modifiers,
				ClickCount: state.click.currentCount,
				FirstMouse: firstMouse,
			}
			window.HandleInput(input)
		}
	case wlproto.WlPointerButtonStateReleased:
		state.buttonPressed = gpui.None[gpui.MouseButton]()

		if window, ok := state.mouseFocusedWindow.Get(); ok {
			input := gpui.MouseUpEvent{
				Button:     button,
				Position:   state.MouseLocation.Unwrap(),
				Modifiers:  state.Modifiers,
				ClickCount: state.click.currentCount,
			}
			window.HandleInput(input)
		}
	}
}

func (p WaylandClientStatePtr) pinchGestureListener() wlproto.ZwpPointerGesturePinchV1Listener {
	pinchPosition := func(state *WaylandClientState) gpui.Point[gpui.Pixels] {
		return state.MouseLocation.UnwrapOr(gpui.NewPoint(gpui.Px(0.0), gpui.Px(0.0)))
	}
	return wlproto.ZwpPointerGesturePinchV1Listener{
		Begin: func(serial, time_ uint32, surface *wlproto.WlSurface, fingers uint32) {
			state := p.GetClient()
			window, ok := state.mouseFocusedWindow.Get()
			if !ok {
				return
			}
			state.pinchScale = 1.0
			input := gpui.PinchEvent{
				Position:  pinchPosition(state),
				Delta:     0.0,
				Modifiers: state.Modifiers,
				Phase:     gpui.TouchPhaseStarted,
			}
			window.HandleInput(input)
		},
		Update: func(time_ uint32, dx, dy, scale, rotation float64) {
			state := p.GetClient()
			window, ok := state.mouseFocusedWindow.Get()
			if !ok {
				return
			}
			newAbsoluteScale := float32(scale)
			previousScale := state.pinchScale
			zoomDelta := newAbsoluteScale - previousScale
			state.pinchScale = newAbsoluteScale

			input := gpui.PinchEvent{
				Position:  pinchPosition(state),
				Delta:     zoomDelta,
				Modifiers: state.Modifiers,
				Phase:     gpui.TouchPhaseMoved,
			}
			window.HandleInput(input)
		},
		End: func(serial, time_ uint32, cancelled int32) {
			state := p.GetClient()
			window, ok := state.mouseFocusedWindow.Get()
			if !ok {
				return
			}
			state.pinchScale = 1.0
			input := gpui.PinchEvent{
				Position:  pinchPosition(state),
				Delta:     0.0,
				Modifiers: state.Modifiers,
				Phase:     gpui.TouchPhaseEnded,
			}
			window.HandleInput(input)
		},
	}
}

// WpFractionalScaleV1Event is a wp_fractional_scale_v1 event.
type WpFractionalScaleV1Event interface{ isWpFractionalScaleV1Event() }

// WpFractionalScaleV1EventPreferredScale is wp_fractional_scale_v1.preferred_scale.
type WpFractionalScaleV1EventPreferredScale struct{ Scale uint32 }

func (WpFractionalScaleV1EventPreferredScale) isWpFractionalScaleV1Event() {}

// FractionalScaleListener is the listener of a window's wp_fractional_scale_v1.
func (p WaylandClientStatePtr) FractionalScaleListener(surfaceID ObjectID) wlproto.WpFractionalScaleV1Listener {
	return wlproto.WpFractionalScaleV1Listener{
		PreferredScale: func(scale uint32) {
			window, ok := getWindow(p.GetClient(), surfaceID)
			if !ok {
				return
			}
			window.HandleFractionalScaleEvent(WpFractionalScaleV1EventPreferredScale{Scale: scale})
		},
	}
}

// ZxdgToplevelDecorationV1Event is a zxdg_toplevel_decoration_v1 event.
type ZxdgToplevelDecorationV1Event interface{ isZxdgToplevelDecorationV1Event() }

// ZxdgToplevelDecorationV1EventConfigure is zxdg_toplevel_decoration_v1.configure.
type ZxdgToplevelDecorationV1EventConfigure struct {
	Mode wlproto.ZxdgToplevelDecorationV1Mode
}

func (ZxdgToplevelDecorationV1EventConfigure) isZxdgToplevelDecorationV1Event() {}

// ToplevelDecorationListener is the listener of a window's zxdg_toplevel_decoration_v1.
func (p WaylandClientStatePtr) ToplevelDecorationListener(surfaceID ObjectID) wlproto.ZxdgToplevelDecorationV1Listener {
	return wlproto.ZxdgToplevelDecorationV1Listener{
		Configure: func(mode wlproto.ZxdgToplevelDecorationV1Mode) {
			window, ok := getWindow(p.GetClient(), surfaceID)
			if !ok {
				return
			}
			window.HandleToplevelDecorationEvent(ZxdgToplevelDecorationV1EventConfigure{Mode: mode})
		},
	}
}

func (p WaylandClientStatePtr) dataDeviceListener() wlproto.WlDataDeviceListener {
	return wlproto.WlDataDeviceListener{
		DataOffer: func(dataOffer *wlproto.WlDataOffer) {
			dataOffer.SetListener(p.dataOfferListener(dataOffer))
			state := p.GetClient()
			state.dataOffers = append(state.dataOffers, NewDataOffer(dataOffer))
			if len(state.dataOffers) > 2 {
				removed := state.dataOffers[0]
				state.dataOffers = state.dataOffers[1:]
				removed.Inner.Destroy()
			}
		},
		Selection: func(dataOffer *wlproto.WlDataOffer) {
			state := p.GetClient()
			if dataOffer == nil {
				state.clipboard.SetOffer(gpui.None[DataOffer[*wlproto.WlDataOffer]]())
				return
			}
			offer := gpui.None[DataOffer[*wlproto.WlDataOffer]]()
			for _, wrapper := range state.dataOffers {
				if wrapper.Inner == dataOffer {
					offer = gpui.Some(wrapper)
					break
				}
			}
			state.clipboard.SetOffer(offer)
		},
		Enter: func(serial uint32, surface *wlproto.WlSurface, x, y float64, dataOffer *wlproto.WlDataOffer) {
			p.handleDragEnter(serial, surface, x, y, dataOffer)
		},
		Motion: func(time_ uint32, x, y float64) {
			state := p.GetClient()
			dragWindow, ok := state.drag.window.Get()
			if !ok {
				return
			}
			position := gpui.NewPoint(gpui.Px(float32(x)), gpui.Px(float32(y)))
			state.drag.position = position

			dragWindow.HandleInput(gpui.FileDropEventPending{Position: position})
		},
		Leave: func() {
			state := p.GetClient()
			if dragWindow, input, ok := state.drag.handleLeave(); ok {
				dragWindow.HandleInput(input)
			}
		},
		Drop: func() {
			state := p.GetClient()
			if dragWindow, input, ok := state.drag.handleDrop(); ok {
				dragWindow.HandleInput(input)
			}
		},
	}
}

func (p WaylandClientStatePtr) handleDragEnter(serial uint32, surface *wlproto.WlSurface, x, y float64, dataOffer *wlproto.WlDataOffer) {
	state := p.GetClient()
	state.serialTracker.Update(SerialKindDataDevice, serial)
	if dataOffer == nil {
		return
	}
	dragWindow, ok := surfaceWindow(state, surface).Get()
	if !ok {
		return
	}
	uriReadGeneration := state.drag.beginURIRead()

	const actions = wlproto.WlDataDeviceManagerDndActionCopy
	dataOffer.SetActions(actions, actions)

	var pipe [2]int
	if err := unix.Pipe2(pipe[:], unix.O_CLOEXEC); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	dataOffer.Receive(fileListMimeType, pipe[1])
	fd := pipe[0]
	unix.Close(pipe[1])

	readTask := state.Common.BackgroundExecutor.Spawn(func(*scheduler.Scope) (string, error) {
		buffer, err := common.ReadFdWithTimeout(fd, common.PipeReadTimeout)
		if err != nil {
			return "", err
		}
		if !utf8.Valid(buffer) {
			return "", errors.New("invalid utf-8 sequence in drag and drop pipe")
		}
		return string(buffer), nil
	})

	state.Common.ForegroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		fileList, err := readTask.Await(s)
		if err != nil {
			slog.Error(fmt.Sprintf("error reading drag and drop pipe: %v", err))
			return struct{}{}, nil
		}

		paths := gpui.ExternalPaths{}
		for _, line := range rustLines(fileList) {
			path, err := fileURLToPath(line)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			paths = append(paths, path)
		}
		position := gpui.NewPoint(gpui.Px(float32(x)), gpui.Px(float32(y)))

		if len(paths) == 0 {
			dataOffer.Destroy()
			return struct{}{}, nil
		}
		client, ok := p.TryGetClient()
		if !ok {
			dataOffer.Destroy()
			return struct{}{}, nil
		}
		window, input, ok := client.drag.completeURIRead(uriReadGeneration, dataOffer, dragWindow, position, paths)

		if ok {
			window.HandleInput(input)
		}
		return struct{}{}, nil
	}).Detach()
}

func rustLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

func fileURLToPath(line string) (string, error) {
	parsed, err := url.Parse(line)
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() {
		return "", fmt.Errorf("relative URL without a base: %q", line)
	}
	if !strings.EqualFold(parsed.Scheme, "file") || parsed.Opaque != "" || (parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost")) || !strings.HasPrefix(parsed.Path, "/") {
		return "", fmt.Errorf("Failed turn %q into a file path", parsed.String())
	}
	return parsed.Path, nil
}

func (p WaylandClientStatePtr) dataOfferListener(dataOffer *wlproto.WlDataOffer) wlproto.WlDataOfferListener {
	return wlproto.WlDataOfferListener{
		Offer: func(mimeType string) {
			state := p.GetClient()
			if mimeType == fileListMimeType {
				serial := state.serialTracker.Get(SerialKindDataDevice)
				dataOffer.Accept(serial.AsRaw(), gpui.Some(mimeType))
			}

			for i := range state.dataOffers {
				if state.dataOffers[i].Inner == dataOffer {
					state.dataOffers[i].AddMimeType(mimeType)
					break
				}
			}
		},
	}
}

func (p WaylandClientStatePtr) dataSourceListener(source *wlproto.WlDataSource, kind DataSourceKind) wlproto.WlDataSourceListener {
	endDrag := func() {
		state := p.GetClient()
		if kind != DataSourceKindDrag {
			if kind == DataSourceKindClipboard {
				source.Destroy()
			}
			return
		}
		externalDrag := state.externalDrag
		if externalDrag == nil {
			return
		}
		state.externalDrag = nil
		externalDrag.source.Destroy()

		externalDrag.window.HandleInput(gpui.FileDropEventEnded{})
	}
	return wlproto.WlDataSourceListener{
		Send: func(mimeType string, fd int) {
			state := p.GetClient()
			switch kind {
			case DataSourceKindClipboard:
				state.clipboard.Send(mimeType, fd)
			case DataSourceKindDrag:
				externalDrag := state.externalDrag
				if externalDrag == nil {
					unix.Close(fd)
					return
				}
				state.clipboard.SendBytes(fd, append([]byte(nil), externalDrag.bytes...))
			default:
				unix.Close(fd)
			}
		},
		Cancelled: endDrag,
		DndFinished: func() {
			if kind != DataSourceKindDrag {
				p.GetClient()
				return
			}
			endDrag()
		},
	}
}

func (p WaylandClientStatePtr) primarySelectionDeviceListener() wlproto.ZwpPrimarySelectionDeviceV1Listener {
	return wlproto.ZwpPrimarySelectionDeviceV1Listener{
		DataOffer: func(offer *wlproto.ZwpPrimarySelectionOfferV1) {
			offer.SetListener(p.primarySelectionOfferListener())
			state := p.GetClient()
			oldOffer := state.primaryDataOffer
			newOffer := NewDataOffer(offer)
			state.primaryDataOffer = &newOffer
			if oldOffer != nil {
				oldOffer.Inner.Destroy()
			}
		},
		Selection: func(dataOffer *wlproto.ZwpPrimarySelectionOfferV1) {
			state := p.GetClient()
			if dataOffer != nil {
				offer := gpui.None[DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]]()
				if state.primaryDataOffer != nil {
					offer = gpui.Some(*state.primaryDataOffer)
				}
				state.clipboard.SetPrimaryOffer(offer)
			} else {
				state.clipboard.SetPrimaryOffer(gpui.None[DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]]())
			}
		},
	}
}

func (p WaylandClientStatePtr) primarySelectionOfferListener() wlproto.ZwpPrimarySelectionOfferV1Listener {
	return wlproto.ZwpPrimarySelectionOfferV1Listener{
		Offer: func(mimeType string) {
			state := p.GetClient()
			if state.primaryDataOffer != nil {
				state.primaryDataOffer.AddMimeType(mimeType)
			}
		},
	}
}

func (p WaylandClientStatePtr) primarySelectionSourceListener(selectionSource *wlproto.ZwpPrimarySelectionSourceV1) wlproto.ZwpPrimarySelectionSourceV1Listener {
	return wlproto.ZwpPrimarySelectionSourceV1Listener{
		Send: func(mimeType string, fd int) {
			state := p.GetClient()
			state.clipboard.SendPrimary(mimeType, fd)
		},
		Cancelled: func() {
			p.GetClient()
			selectionSource.Destroy()
		},
	}
}
