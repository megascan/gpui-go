//go:build windows

package windows

import (
	"fmt"
	"log/slog"
	"unicode"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// VIRTUAL_KEY is a Win32 virtual-key code.
type VIRTUAL_KEY uint16

// WindowsKeyboardLayout is the active keyboard layout.
type WindowsKeyboardLayout struct {
	id   string
	name string
}

var _ gpui.PlatformKeyboardLayout = (*WindowsKeyboardLayout)(nil)

type vkeyShifted struct {
	vkey    uint16
	shifted bool
}

// WindowsKeyboardMapper maps keystrokes between keys and virtual keys of the active layout.
type WindowsKeyboardMapper struct {
	keyToVkey     map[string]vkeyShifted
	vkeyToKey     map[uint16]string
	vkeyToShifted map[uint16]string
}

var _ gpui.PlatformKeyboardMapper = (*WindowsKeyboardMapper)(nil)

// ID returns the layout id.
func (l *WindowsKeyboardLayout) ID() string {
	return l.id
}

// Name returns the layout display name.
func (l *WindowsKeyboardLayout) Name() string {
	return l.name
}

// MapKeyEquivalent maps a keystroke to the keybinding keystroke of the active layout.
func (m *WindowsKeyboardMapper) MapKeyEquivalent(keystroke gpui.Keystroke, useKeyEquivalents bool) gpui.KeybindingKeystroke {
	found, ok := m.getVkeyFromKey(keystroke.Key, useKeyEquivalents)
	if !ok {
		return gpui.KeybindingKeystrokeFromKeystroke(keystroke)
	}
	vkey, shiftedKey := found.vkey, found.shifted
	if shiftedKey && keystroke.Modifiers.Shift {
		slog.Warn(fmt.Sprintf("Keystroke '%s' has both shift and a shifted key, this is likely a bug", keystroke.Key))
	}

	shift := shiftedKey || keystroke.Modifiers.Shift
	keystroke.Modifiers.Shift = false

	key, ok := m.vkeyToKey[vkey]
	if !ok {
		slog.Error(fmt.Sprintf("Failed to map key equivalent '%+v' to a valid key", keystroke))
		return gpui.KeybindingKeystrokeFromKeystroke(keystroke)
	}

	if shift {
		shifted, ok := m.vkeyToShifted[vkey]
		if !ok {
			slog.Error(fmt.Sprintf("Failed to map keystroke %+v with virtual key '%d' to a shifted key", keystroke, vkey))
			return gpui.KeybindingKeystrokeFromKeystroke(keystroke)
		}
		keystroke.Key = shifted
	} else {
		keystroke.Key = key
	}

	modifiers := keystroke.Modifiers
	modifiers.Shift = shift

	return gpui.NewKeybindingKeystroke(keystroke, modifiers, key)
}

// GetKeyEquivalents returns nil: Windows has no key equivalents table.
func (m *WindowsKeyboardMapper) GetKeyEquivalents() map[rune]rune {
	return nil
}

// NewWindowsKeyboardLayout reads the active keyboard layout and its name from the registry.
func NewWindowsKeyboardLayout() (*WindowsKeyboardLayout, error) {
	var buffer [win32.KL_NAMELENGTH]uint16
	if err := win32.GetKeyboardLayoutNameW(&buffer); err != nil {
		return nil, err
	}
	id := string(utf16.Decode(buffer[:len(buffer)-1]))
	entry, err := registry.OpenKey(registry.LOCAL_MACHINE, fmt.Sprintf("System\\CurrentControlSet\\Control\\Keyboard Layouts\\%s", id), registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer entry.Close()
	name, _, err := entry.GetStringValue("Layout Text")
	if err != nil {
		return nil, err
	}
	return &WindowsKeyboardLayout{id: id, name: name}, nil
}

// WindowsKeyboardLayoutUnknown is the layout used when the active one cannot be read.
func WindowsKeyboardLayoutUnknown() *WindowsKeyboardLayout {
	return &WindowsKeyboardLayout{id: "unknown", name: "unknown"}
}

// NewWindowsKeyboardMapper builds the mapper for the active layout.
func NewWindowsKeyboardMapper() *WindowsKeyboardMapper {
	keyToVkey := map[string]vkeyShifted{}
	vkeyToKey := map[uint16]string{}
	vkeyToShifted := map[uint16]string{}
	for _, vkey := range candidateVkeys {
		if key, ok := getKeyFromVkey(vkey); ok {
			keyToVkey[key] = vkeyShifted{uint16(vkey), false}
			vkeyToKey[uint16(vkey)] = key
		}
		scanCode := win32.MapVirtualKeyW(uint32(vkey), win32.MAPVK_VK_TO_VSC)
		if scanCode == 0 {
			continue
		}
		if shiftedKey, ok := getShiftedKey(vkey, scanCode); ok {
			keyToVkey[shiftedKey] = vkeyShifted{uint16(vkey), true}
			vkeyToShifted[uint16(vkey)] = shiftedKey
		}
	}
	return &WindowsKeyboardMapper{
		keyToVkey:     keyToVkey,
		vkeyToKey:     vkeyToKey,
		vkeyToShifted: vkeyToShifted,
	}
}

func (m *WindowsKeyboardMapper) getVkeyFromKey(key string, useKeyEquivalents bool) (vkeyShifted, bool) {
	if useKeyEquivalents {
		return getVkeyFromKeyWithUSLayout(key)
	}
	v, ok := m.keyToVkey[key]
	return v, ok
}

func getKeystrokeKey(vkey VIRTUAL_KEY, scanCode uint32, modifiers *gpui.Modifiers) (string, bool) {
	if modifiers.Shift && needToConvertToShiftedKey(vkey) {
		key, ok := getShiftedKey(vkey, scanCode)
		if ok {
			modifiers.Shift = false
		}
		return key, ok
	}
	return getKeyFromVkey(vkey)
}

func getKeyFromVkey(vkey VIRTUAL_KEY) (string, bool) {
	keyData := win32.MapVirtualKeyW(uint32(vkey), win32.MAPVK_VK_TO_CHAR)
	if keyData == 0 {
		return "", false
	}

	r := rune(keyData & 0xFFFF)
	if r >= 0xD800 && r <= 0xDFFF {
		return "", false
	}
	if r < 0x80 {
		r = unicode.ToLower(r)
	}
	return string(r), true
}

func needToConvertToShiftedKey(vkey VIRTUAL_KEY) bool {
	switch vkey {
	case win32.VK_OEM_3, win32.VK_OEM_MINUS, win32.VK_OEM_PLUS, win32.VK_OEM_4, win32.VK_OEM_5,
		win32.VK_OEM_6, win32.VK_OEM_1, win32.VK_OEM_7, win32.VK_OEM_COMMA, win32.VK_OEM_PERIOD,
		win32.VK_OEM_2, win32.VK_OEM_102, win32.VK_OEM_8, win32.VK_ABNT_C1, win32.VK_0, win32.VK_1,
		win32.VK_2, win32.VK_3, win32.VK_4, win32.VK_5, win32.VK_6, win32.VK_7, win32.VK_8, win32.VK_9:
		return true
	}
	return false
}

func getShiftedKey(vkey VIRTUAL_KEY, scanCode uint32) (string, bool) {
	return generateKeyChar(vkey, scanCode, false, true, false)
}

func generateKeyChar(vkey VIRTUAL_KEY, scanCode uint32, control, shift, alt bool) (string, bool) {
	var state [256]byte
	if control {
		state[win32.VK_CONTROL] = 0x80
	}
	if shift {
		state[win32.VK_SHIFT] = 0x80
	}
	if alt {
		state[win32.VK_MENU] = 0x80
	}

	var buffer [8]uint16
	n := win32.ToUnicode(uint32(vkey), scanCode, &state, buffer[:], 0x5)

	switch {
	case n > 0:
		candidate, ok := strictUTF16(buffer[:n])
		if !ok || candidate == "" || unicode.IsControl([]rune(candidate)[0]) {
			return "", false
		}
		return candidate, true
	case n < 0:
		return strictUTF16(buffer[:-n])
	}
	return "", false
}

func strictUTF16(s []uint16) (string, bool) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 0xD800 && c <= 0xDBFF:
			if i+1 >= len(s) || s[i+1] < 0xDC00 || s[i+1] > 0xDFFF {
				return "", false
			}
			i++
		case c >= 0xDC00 && c <= 0xDFFF:
			return "", false
		}
	}
	return string(utf16.Decode(s)), true
}

func getVkeyFromKeyWithUSLayout(key string) (vkeyShifted, bool) {
	v := func(vkey VIRTUAL_KEY, shifted bool) (vkeyShifted, bool) {
		return vkeyShifted{uint16(vkey), shifted}, true
	}
	switch key {
	case "`":
		return v(win32.VK_OEM_3, false)
	case "~":
		return v(win32.VK_OEM_3, true)
	case "1":
		return v(win32.VK_1, false)
	case "!":
		return v(win32.VK_1, true)
	case "2":
		return v(win32.VK_2, false)
	case "@":
		return v(win32.VK_2, true)
	case "3":
		return v(win32.VK_3, false)
	case "#":
		return v(win32.VK_3, true)
	case "4":
		return v(win32.VK_4, false)
	case "$":
		return v(win32.VK_4, true)
	case "5":
		return v(win32.VK_5, false)
	case "%":
		return v(win32.VK_5, true)
	case "6":
		return v(win32.VK_6, false)
	case "^":
		return v(win32.VK_6, true)
	case "7":
		return v(win32.VK_7, false)
	case "&":
		return v(win32.VK_7, true)
	case "8":
		return v(win32.VK_8, false)
	case "*":
		return v(win32.VK_8, true)
	case "9":
		return v(win32.VK_9, false)
	case "(":
		return v(win32.VK_9, true)
	case "0":
		return v(win32.VK_0, false)
	case ")":
		return v(win32.VK_0, true)
	case "-":
		return v(win32.VK_OEM_MINUS, false)
	case "_":
		return v(win32.VK_OEM_MINUS, true)
	case "=":
		return v(win32.VK_OEM_PLUS, false)
	case "+":
		return v(win32.VK_OEM_PLUS, true)
	case "[":
		return v(win32.VK_OEM_4, false)
	case "{":
		return v(win32.VK_OEM_4, true)
	case "]":
		return v(win32.VK_OEM_6, false)
	case "}":
		return v(win32.VK_OEM_6, true)
	case "\\":
		return v(win32.VK_OEM_5, false)
	case "|":
		return v(win32.VK_OEM_5, true)
	case ";":
		return v(win32.VK_OEM_1, false)
	case ":":
		return v(win32.VK_OEM_1, true)
	case "'":
		return v(win32.VK_OEM_7, false)
	case "\"":
		return v(win32.VK_OEM_7, true)
	case ",":
		return v(win32.VK_OEM_COMMA, false)
	case "<":
		return v(win32.VK_OEM_COMMA, true)
	case ".":
		return v(win32.VK_OEM_PERIOD, false)
	case ">":
		return v(win32.VK_OEM_PERIOD, true)
	case "/":
		return v(win32.VK_OEM_2, false)
	case "?":
		return v(win32.VK_OEM_2, true)
	}
	return vkeyShifted{}, false
}

var candidateVkeys = []VIRTUAL_KEY{
	win32.VK_OEM_3,
	win32.VK_OEM_MINUS,
	win32.VK_OEM_PLUS,
	win32.VK_OEM_4,
	win32.VK_OEM_5,
	win32.VK_OEM_6,
	win32.VK_OEM_1,
	win32.VK_OEM_7,
	win32.VK_OEM_COMMA,
	win32.VK_OEM_PERIOD,
	win32.VK_OEM_2,
	win32.VK_OEM_102,
	win32.VK_OEM_8,
	win32.VK_ABNT_C1,
	win32.VK_0,
	win32.VK_1,
	win32.VK_2,
	win32.VK_3,
	win32.VK_4,
	win32.VK_5,
	win32.VK_6,
	win32.VK_7,
	win32.VK_8,
	win32.VK_9,
}
