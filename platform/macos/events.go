//go:build darwin && !ios

package macos

import (
	"unicode/utf16"
	"unicode/utf8"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const escapeKey uint16 = 0x1b

const (
	backspaceKey   uint32 = 0x7f
	spaceKey       uint32 = ' '
	enterKey       uint32 = 0x0d
	numpadEnterKey uint32 = 0x03
	tabKey         uint32 = 0x09
	shiftTabKey    uint32 = 0x19
)

const (
	backspaceKeyEquivalent uint32 = 0x08
	deleteKeyEquivalent    uint32 = 0x7f
)

func keyToMenuEquivalent(key string) string {
	var code uint32
	switch key {
	case "space":
		code = spaceKey
	case "backspace":
		code = backspaceKeyEquivalent
	case "escape":
		code = uint32(escapeKey)
	case "up":
		code = appkit.NSUpArrowFunctionKey
	case "down":
		code = appkit.NSDownArrowFunctionKey
	case "left":
		code = appkit.NSLeftArrowFunctionKey
	case "right":
		code = appkit.NSRightArrowFunctionKey
	case "pageup":
		code = appkit.NSPageUpFunctionKey
	case "pagedown":
		code = appkit.NSPageDownFunctionKey
	case "home":
		code = appkit.NSHomeFunctionKey
	case "end":
		code = appkit.NSEndFunctionKey
	case "delete":
		code = deleteKeyEquivalent
	case "insert":
		code = appkit.NSHelpFunctionKey
	case "f1":
		code = appkit.NSF1FunctionKey
	case "f2":
		code = appkit.NSF2FunctionKey
	case "f3":
		code = appkit.NSF3FunctionKey
	case "f4":
		code = appkit.NSF4FunctionKey
	case "f5":
		code = appkit.NSF5FunctionKey
	case "f6":
		code = appkit.NSF6FunctionKey
	case "f7":
		code = appkit.NSF7FunctionKey
	case "f8":
		code = appkit.NSF8FunctionKey
	case "f9":
		code = appkit.NSF9FunctionKey
	case "f10":
		code = appkit.NSF10FunctionKey
	case "f11":
		code = appkit.NSF11FunctionKey
	case "f12":
		code = appkit.NSF12FunctionKey
	case "f13":
		code = appkit.NSF13FunctionKey
	case "f14":
		code = appkit.NSF14FunctionKey
	case "f15":
		code = appkit.NSF15FunctionKey
	case "f16":
		code = appkit.NSF16FunctionKey
	case "f17":
		code = appkit.NSF17FunctionKey
	case "f18":
		code = appkit.NSF18FunctionKey
	case "f19":
		code = appkit.NSF19FunctionKey
	case "f20":
		code = appkit.NSF20FunctionKey
	case "f21":
		code = appkit.NSF21FunctionKey
	case "f22":
		code = appkit.NSF22FunctionKey
	case "f23":
		code = appkit.NSF23FunctionKey
	case "f24":
		code = appkit.NSF24FunctionKey
	case "f25":
		code = appkit.NSF25FunctionKey
	case "f26":
		code = appkit.NSF26FunctionKey
	case "f27":
		code = appkit.NSF27FunctionKey
	case "f28":
		code = appkit.NSF28FunctionKey
	case "f29":
		code = appkit.NSF29FunctionKey
	case "f30":
		code = appkit.NSF30FunctionKey
	case "f31":
		code = appkit.NSF31FunctionKey
	case "f32":
		code = appkit.NSF32FunctionKey
	case "f33":
		code = appkit.NSF33FunctionKey
	case "f34":
		code = appkit.NSF34FunctionKey
	case "f35":
		code = appkit.NSF35FunctionKey
	default:
		return key
	}
	return string(utf16.Decode([]uint16{uint16(code)}))
}

func readModifiers(nativeEvent appkit.NSEvent) gpui.Modifiers {
	modifiers := nativeEvent.ModifierFlags()
	control := modifiers.Contains(appkit.NSEventModifierFlagControl)
	alt := modifiers.Contains(appkit.NSEventModifierFlagOption)
	shift := modifiers.Contains(appkit.NSEventModifierFlagShift)
	command := modifiers.Contains(appkit.NSEventModifierFlagCommand)
	function := modifiers.Contains(appkit.NSEventModifierFlagFunction)

	return gpui.Modifiers{
		Control:  control,
		Alt:      alt,
		Shift:    shift,
		Platform: command,
		Function: function,
	}
}

func mouseButtonFromNumber(n int) (gpui.MouseButton, bool) {
	switch n {
	case 0:
		return gpui.MouseButtonLeft, true
	case 1:
		return gpui.MouseButtonRight, true
	case 2:
		return gpui.MouseButtonMiddle, true
	case 3:
		return gpui.MouseButtonNavigateBack, true
	case 4:
		return gpui.MouseButtonNavigateForward, true
	default:
		return 0, false
	}
}

func eventPosition(nativeEvent appkit.NSEvent, windowHeight gpui.Pixels) gpui.Point[gpui.Pixels] {
	location := nativeEvent.LocationInWindow()
	return gpui.NewPoint(
		gpui.Px(float32(location.X)),
		windowHeight-gpui.Px(float32(location.Y)),
	)
}

func touchPhaseFromNative(phase appkit.NSEventPhase) gpui.TouchPhase {
	switch phase {
	case appkit.NSEventPhaseMayBegin, appkit.NSEventPhaseBegan:
		return gpui.TouchPhaseStarted
	case appkit.NSEventPhaseEnded:
		return gpui.TouchPhaseEnded
	default:
		return gpui.TouchPhaseMoved
	}
}

func platformInputFromNative(event objc.ID, windowHeight gpui.Option[gpui.Pixels]) gpui.PlatformInput {
	if event == 0 {
		return nil
	}
	nativeEvent := appkit.NSEvent(event)
	eventType := nativeEvent.Type()

	switch eventType {
	case 0, 21, 32, 33, 35, 36, 37:
		return nil
	}

	height, hasHeight := windowHeight.Get()

	switch eventType {
	case appkit.NSEventTypeFlagsChanged:
		return gpui.ModifiersChangedEvent{
			Modifiers: readModifiers(nativeEvent),
			Capslock: gpui.Capslock{
				On: nativeEvent.ModifierFlags().Contains(appkit.NSEventModifierFlagCapsLock),
			},
		}
	case appkit.NSEventTypeKeyDown:
		return gpui.KeyDownEvent{
			Keystroke:            parseKeystroke(nativeEvent),
			IsHeld:               nativeEvent.IsARepeat(),
			PreferCharacterInput: false,
		}
	case appkit.NSEventTypeKeyUp:
		return gpui.KeyUpEvent{
			Keystroke: parseKeystroke(nativeEvent),
		}
	case appkit.NSEventTypeLeftMouseDown, appkit.NSEventTypeRightMouseDown, appkit.NSEventTypeOtherMouseDown:
		button, ok := mouseButtonFromNumber(nativeEvent.ButtonNumber())
		if !ok || !hasHeight {
			return nil
		}
		return gpui.MouseDownEvent{
			Button:     button,
			Position:   eventPosition(nativeEvent, height),
			Modifiers:  readModifiers(nativeEvent),
			ClickCount: nativeEvent.ClickCount(),
			FirstMouse: false,
		}
	case appkit.NSEventTypeLeftMouseUp, appkit.NSEventTypeRightMouseUp, appkit.NSEventTypeOtherMouseUp:
		button, ok := mouseButtonFromNumber(nativeEvent.ButtonNumber())
		if !ok || !hasHeight {
			return nil
		}
		return gpui.MouseUpEvent{
			Button:     button,
			Position:   eventPosition(nativeEvent, height),
			Modifiers:  readModifiers(nativeEvent),
			ClickCount: nativeEvent.ClickCount(),
		}
	case appkit.NSEventTypePressure:
		stage := nativeEvent.Stage()
		pressure := nativeEvent.Pressure()
		if !hasHeight {
			return nil
		}
		var pressureStage gpui.PressureStage
		switch stage {
		case 1:
			pressureStage = gpui.PressureStageNormal
		case 2:
			pressureStage = gpui.PressureStageForce
		default:
			pressureStage = gpui.PressureStageZero
		}
		return gpui.MousePressureEvent{
			Stage:     pressureStage,
			Pressure:  pressure,
			Modifiers: readModifiers(nativeEvent),
			Position:  eventPosition(nativeEvent, height),
		}
	case appkit.NSEventTypeSwipe:
		if nativeEvent.Phase() != appkit.NSEventPhaseEnded {
			return nil
		}
		var direction gpui.MouseButton
		switch x := nativeEvent.DeltaX(); {
		case x > 0:
			direction = gpui.MouseButtonNavigateBack
		case x < 0:
			direction = gpui.MouseButtonNavigateForward
		default:
			return nil
		}
		if !hasHeight {
			return nil
		}
		return gpui.MouseDownEvent{
			Button:     direction,
			Position:   eventPosition(nativeEvent, height),
			Modifiers:  readModifiers(nativeEvent),
			ClickCount: 1,
			FirstMouse: false,
		}
	case appkit.NSEventTypeMagnify:
		if !hasHeight {
			return nil
		}
		phase := touchPhaseFromNative(nativeEvent.Phase())
		magnification := float32(nativeEvent.Magnification())
		return gpui.PinchEvent{
			Position:  eventPosition(nativeEvent, height),
			Delta:     magnification,
			Modifiers: readModifiers(nativeEvent),
			Phase:     phase,
		}
	case appkit.NSEventTypeScrollWheel:
		if !hasHeight {
			return nil
		}
		phase := touchPhaseFromNative(nativeEvent.Phase())
		rawData := gpui.NewPoint(
			float32(nativeEvent.ScrollingDeltaX()),
			float32(nativeEvent.ScrollingDeltaY()),
		)
		var delta gpui.ScrollDelta
		if nativeEvent.HasPreciseScrollingDeltas() {
			delta = gpui.ScrollDeltaPixels(gpui.NewPoint(gpui.Px(rawData.X), gpui.Px(rawData.Y)))
		} else {
			delta = gpui.ScrollDeltaLines(rawData)
		}
		return gpui.ScrollEvent{
			Position:   eventPosition(nativeEvent, height),
			Delta:      delta,
			TouchPhase: phase,
			Modifiers:  readModifiers(nativeEvent),
			Source:     gpui.ScrollSourcePointer,
		}
	case appkit.NSEventTypeLeftMouseDragged, appkit.NSEventTypeRightMouseDragged, appkit.NSEventTypeOtherMouseDragged:
		pressedButton, ok := mouseButtonFromNumber(nativeEvent.ButtonNumber())
		if !ok || !hasHeight {
			return nil
		}
		return gpui.MouseMoveEvent{
			PressedButton: gpui.Some(pressedButton),
			Position:      eventPosition(nativeEvent, height),
			Modifiers:     readModifiers(nativeEvent),
		}
	case appkit.NSEventTypeMouseMoved:
		if !hasHeight {
			return nil
		}
		return gpui.MouseMoveEvent{
			Position:      eventPosition(nativeEvent, height),
			PressedButton: gpui.None[gpui.MouseButton](),
			Modifiers:     readModifiers(nativeEvent),
		}
	case appkit.NSEventTypeMouseExited:
		if !hasHeight {
			return nil
		}
		return gpui.MouseExitEvent{
			Position:      eventPosition(nativeEvent, height),
			PressedButton: gpui.None[gpui.MouseButton](),
			Modifiers:     readModifiers(nativeEvent),
		}
	default:
		return nil
	}
}

func parseKeystroke(nativeEvent appkit.NSEvent) gpui.Keystroke {
	characters := nsStringToStr(nativeEvent.CharactersIgnoringModifiers())
	keyChar := gpui.None[string]()
	var firstChar uint32
	hasFirst := false
	if r, size := utf8.DecodeRuneInString(characters); size > 0 {
		firstChar = uint32(uint16(r))
		hasFirst = true
	}
	modifiers := nativeEvent.ModifierFlags()

	control := modifiers.Contains(appkit.NSEventModifierFlagControl)
	alt := modifiers.Contains(appkit.NSEventModifierFlagOption)
	shift := modifiers.Contains(appkit.NSEventModifierFlagShift)
	command := modifiers.Contains(appkit.NSEventModifierFlagCommand)
	function := modifiers.Contains(appkit.NSEventModifierFlagFunction) &&
		(!hasFirst || !(firstChar >= appkit.NSUpArrowFunctionKey && firstChar <= appkit.NSModeSwitchFunctionKey))

	var key string
	switch {
	case hasFirst && firstChar == spaceKey:
		keyChar = gpui.Some(" ")
		key = "space"
	case hasFirst && firstChar == tabKey:
		keyChar = gpui.Some("\t")
		key = "tab"
	case hasFirst && (firstChar == enterKey || firstChar == numpadEnterKey):
		keyChar = gpui.Some("\n")
		key = "enter"
	case hasFirst && firstChar == backspaceKey:
		key = "backspace"
	case hasFirst && firstChar == uint32(escapeKey):
		key = "escape"
	case hasFirst && firstChar == shiftTabKey:
		key = "tab"
	case hasFirst && firstChar == appkit.NSUpArrowFunctionKey:
		key = "up"
	case hasFirst && firstChar == appkit.NSDownArrowFunctionKey:
		key = "down"
	case hasFirst && firstChar == appkit.NSLeftArrowFunctionKey:
		key = "left"
	case hasFirst && firstChar == appkit.NSRightArrowFunctionKey:
		key = "right"
	case hasFirst && firstChar == appkit.NSPageUpFunctionKey:
		key = "pageup"
	case hasFirst && firstChar == appkit.NSPageDownFunctionKey:
		key = "pagedown"
	case hasFirst && firstChar == appkit.NSHomeFunctionKey:
		key = "home"
	case hasFirst && firstChar == appkit.NSEndFunctionKey:
		key = "end"
	case hasFirst && firstChar == appkit.NSDeleteFunctionKey:
		key = "delete"
	case hasFirst && firstChar == appkit.NSHelpFunctionKey:
		key = "insert"
	case hasFirst && firstChar == appkit.NSF1FunctionKey:
		key = "f1"
	case hasFirst && firstChar == appkit.NSF2FunctionKey:
		key = "f2"
	case hasFirst && firstChar == appkit.NSF3FunctionKey:
		key = "f3"
	case hasFirst && firstChar == appkit.NSF4FunctionKey:
		key = "f4"
	case hasFirst && firstChar == appkit.NSF5FunctionKey:
		key = "f5"
	case hasFirst && firstChar == appkit.NSF6FunctionKey:
		key = "f6"
	case hasFirst && firstChar == appkit.NSF7FunctionKey:
		key = "f7"
	case hasFirst && firstChar == appkit.NSF8FunctionKey:
		key = "f8"
	case hasFirst && firstChar == appkit.NSF9FunctionKey:
		key = "f9"
	case hasFirst && firstChar == appkit.NSF10FunctionKey:
		key = "f10"
	case hasFirst && firstChar == appkit.NSF11FunctionKey:
		key = "f11"
	case hasFirst && firstChar == appkit.NSF12FunctionKey:
		key = "f12"
	case hasFirst && firstChar == appkit.NSF13FunctionKey:
		key = "f13"
	case hasFirst && firstChar == appkit.NSF14FunctionKey:
		key = "f14"
	case hasFirst && firstChar == appkit.NSF15FunctionKey:
		key = "f15"
	case hasFirst && firstChar == appkit.NSF16FunctionKey:
		key = "f16"
	case hasFirst && firstChar == appkit.NSF17FunctionKey:
		key = "f17"
	case hasFirst && firstChar == appkit.NSF18FunctionKey:
		key = "f18"
	case hasFirst && firstChar == appkit.NSF19FunctionKey:
		key = "f19"
	case hasFirst && firstChar == appkit.NSF20FunctionKey:
		key = "f20"
	case hasFirst && firstChar == appkit.NSF21FunctionKey:
		key = "f21"
	case hasFirst && firstChar == appkit.NSF22FunctionKey:
		key = "f22"
	case hasFirst && firstChar == appkit.NSF23FunctionKey:
		key = "f23"
	case hasFirst && firstChar == appkit.NSF24FunctionKey:
		key = "f24"
	case hasFirst && firstChar == appkit.NSF25FunctionKey:
		key = "f25"
	case hasFirst && firstChar == appkit.NSF26FunctionKey:
		key = "f26"
	case hasFirst && firstChar == appkit.NSF27FunctionKey:
		key = "f27"
	case hasFirst && firstChar == appkit.NSF28FunctionKey:
		key = "f28"
	case hasFirst && firstChar == appkit.NSF29FunctionKey:
		key = "f29"
	case hasFirst && firstChar == appkit.NSF30FunctionKey:
		key = "f30"
	case hasFirst && firstChar == appkit.NSF31FunctionKey:
		key = "f31"
	case hasFirst && firstChar == appkit.NSF32FunctionKey:
		key = "f32"
	case hasFirst && firstChar == appkit.NSF33FunctionKey:
		key = "f33"
	case hasFirst && firstChar == appkit.NSF34FunctionKey:
		key = "f34"
	case hasFirst && firstChar == appkit.NSF35FunctionKey:
		key = "f35"
	default:
		keyCode := nativeEvent.KeyCode()
		charsIgnoringModifiers := charsForModifiedKey(keyCode, noMod)
		charsWithShift := charsForModifiedKey(keyCode, shiftMod)
		alwaysUseCmdLayout := alwaysUseCommandLayout()

		if command || alwaysUseCmdLayout {
			charsWithCmd := charsForModifiedKey(keyCode, cmdMod)
			charsWithBoth := charsForModifiedKey(keyCode, cmdMod|shiftMod)

			if charsWithBoth != charsWithCmd {
				charsWithShift = charsWithBoth
			} else if asciiToUpper(charsWithCmd) != charsWithCmd {
				charsWithShift = asciiToUpper(charsWithCmd)
			}
			charsIgnoringModifiers = charsWithCmd
		}

		if !control && !command && !function {
			mods := noMod
			if shift {
				mods |= shiftMod
			}
			if alt {
				mods |= optionMod
			}

			keyChar = gpui.Some(charsForModifiedKey(keyCode, mods))
		}

		if shift && allASCIILower(charsIgnoringModifiers) {
			key = charsIgnoringModifiers
		} else if shift {
			shift = false
			key = charsWithShift
		} else {
			key = charsIgnoringModifiers
		}
	}

	return gpui.Keystroke{
		Modifiers: gpui.Modifiers{
			Control:  control,
			Alt:      alt,
			Shift:    shift,
			Platform: command,
			Function: function,
		},
		Key:     key,
		KeyChar: keyChar,
	}
}

func asciiToUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}

func allASCIILower(s string) bool {
	for _, c := range s {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func alwaysUseCommandLayout() bool {
	if isASCII(charsForModifiedKey(0, noMod)) {
		return false
	}

	return isASCII(charsForModifiedKey(0, cmdMod))
}

const (
	noMod     uint32 = 0
	cmdMod    uint32 = 1
	shiftMod  uint32 = 2
	optionMod uint32 = 8
)

func charsForModifiedKey(code uint16, modifiers uint32) string {
	const cgSpaceKey uint16 = 49
	const kUCKeyActionDown uint16 = 0
	const kUCKeyTranslateNoDeadKeysMask uint32 = 0

	mustLoadCarbon()
	keyboardType := uint32(lmGetKbdType())
	const bufferSize = 4
	var deadKeyState uint32
	var buffer [bufferSize]uint16
	var bufferLen uintptr

	keyboard := tisCopyCurrentKeyboardLayoutInputSource()
	if keyboard == 0 {
		return ""
	}
	layoutData := cf.TypeRef(tisGetInputSourceProperty(keyboard, kTISPropertyUnicodeKeyLayoutData))
	if layoutData == 0 {
		objc.Release(keyboard)
		return ""
	}
	keyboardLayout := cfDataGetBytePtr(layoutData)

	ucKeyTranslate(
		keyboardLayout,
		code,
		kUCKeyActionDown,
		modifiers,
		keyboardType,
		kUCKeyTranslateNoDeadKeysMask,
		&deadKeyState,
		bufferSize,
		&bufferLen,
		&buffer[0],
	)
	if deadKeyState != 0 {
		ucKeyTranslate(
			keyboardLayout,
			cgSpaceKey,
			kUCKeyActionDown,
			modifiers,
			keyboardType,
			kUCKeyTranslateNoDeadKeysMask,
			&deadKeyState,
			bufferSize,
			&bufferLen,
			&buffer[0],
		)
	}
	objc.Release(keyboard)
	return stringFromUTF16OrDefault(buffer[:min(int(bufferLen), bufferSize)])
}

func stringFromUTF16OrDefault(units []uint16) string {
	for i := 0; i < len(units); i++ {
		u := units[i]
		switch {
		case utf16.IsSurrogate(rune(u)) && u < 0xdc00:
			if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
				return ""
			}
			i++
		case u >= 0xdc00 && u <= 0xdfff:
			return ""
		}
	}
	return string(utf16.Decode(units))
}
