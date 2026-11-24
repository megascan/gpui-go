//go:build linux

package x11

import (
	"math/bits"

	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xext"
)

type buttonOrScrollKind uint8

const (
	buttonOrScrollButton buttonOrScrollKind = iota
	buttonOrScrollScroll
)

type buttonOrScroll struct {
	kind   buttonOrScrollKind
	button gpui.MouseButton
	scroll scrollDirection
}

type scrollDirection uint8

const (
	scrollDirectionUp scrollDirection = iota
	scrollDirectionDown
	scrollDirectionLeft
	scrollDirectionRight
)

func buttonOrScrollFromEventDetail(detail uint32) (buttonOrScroll, bool) {
	button := func(b gpui.MouseButton) (buttonOrScroll, bool) {
		return buttonOrScroll{kind: buttonOrScrollButton, button: b}, true
	}
	scroll := func(d scrollDirection) (buttonOrScroll, bool) {
		return buttonOrScroll{kind: buttonOrScrollScroll, scroll: d}, true
	}
	switch detail {
	case 1:
		return button(gpui.MouseButtonLeft)
	case 2:
		return button(gpui.MouseButtonMiddle)
	case 3:
		return button(gpui.MouseButtonRight)
	case 4:
		return scroll(scrollDirectionUp)
	case 5:
		return scroll(scrollDirectionDown)
	case 6:
		return scroll(scrollDirectionLeft)
	case 7:
		return scroll(scrollDirectionRight)
	case 8:
		return button(gpui.MouseButtonNavigate(gpui.NavigationDirectionBack))
	case 9:
		return button(gpui.MouseButtonNavigate(gpui.NavigationDirectionForward))
	}
	return buttonOrScroll{}, false
}

func modifiersFromState(state uint16) gpui.Modifiers {
	return gpui.Modifiers{
		Control:  state&xproto.KeyButMaskControl != 0,
		Alt:      state&xproto.KeyButMaskMod1 != 0,
		Shift:    state&xproto.KeyButMaskShift != 0,
		Platform: state&xproto.KeyButMaskMod4 != 0,
		Function: false,
	}
}

func modifiersFromXinputInfo(modifierInfo xext.XinputModifierInfo) gpui.Modifiers {
	effective := uint16(modifierInfo.Effective)
	return gpui.Modifiers{
		Control:  effective&xproto.ModMaskControl == xproto.ModMaskControl,
		Alt:      effective&xproto.ModMask1 == xproto.ModMask1,
		Shift:    effective&xproto.ModMaskShift == xproto.ModMaskShift,
		Platform: effective&xproto.ModMask4 == xproto.ModMask4,
		Function: false,
	}
}

func pressedButtonFromMask(buttonMask uint32) gpui.Option[gpui.MouseButton] {
	switch {
	case buttonMask&2 == 2:
		return gpui.Some(gpui.MouseButtonLeft)
	case buttonMask&4 == 4:
		return gpui.Some(gpui.MouseButtonMiddle)
	case buttonMask&8 == 8:
		return gpui.Some(gpui.MouseButtonRight)
	}
	return gpui.None[gpui.MouseButton]()
}

func getValuatorAxisIndex(valuatorMask []uint32, valuatorNumber uint16) (int, bool) {
	if bitIsSetInVec(valuatorMask, valuatorNumber) {
		return int(popcountUptoBitIndex(valuatorMask, valuatorNumber)), true
	}
	return 0, false
}

func popcountUptoBitIndex(bitVec []uint32, bitIndex uint16) uint32 {
	arrayIndex := int(bitIndex) / 32
	var popcount uint32
	if arrayIndex < len(bitVec) {
		popcount = uint32(bits.OnesCount32(keepBitsUpto(bitVec[arrayIndex], bitIndex%32)))
	}
	if arrayIndex == 0 {
		return popcount
	}
	var leading uint32
	for i := 0; i < arrayIndex && i < len(bitVec); i++ {
		leading += uint32(bits.OnesCount32(bitVec[i]))
	}
	return popcount + leading
}

func bitIsSetInVec(bitVec []uint32, bitIndex uint16) bool {
	arrayIndex := int(bitIndex) / 32
	return arrayIndex < len(bitVec) && bitIsSet(bitVec[arrayIndex], bitIndex%32)
}

func bitIsSet(bits uint32, bitIndex uint16) bool {
	return bits&(1<<bitIndex) != 0
}

func keepBitsUpto(bits uint32, bitIndex uint16) uint32 {
	switch {
	case bitIndex == 0:
		return 0
	case bitIndex >= 32:
		return ^uint32(0)
	}
	return bits & ((1 << bitIndex) - 1)
}
