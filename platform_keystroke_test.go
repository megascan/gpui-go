package gpui

import (
	"encoding/json/v2"
	"testing"
)

func mustParse(t *testing.T, source string) Keystroke {
	t.Helper()
	k, err := KeystrokeParse(source)
	if err != nil {
		t.Fatalf("parse %q: %v", source, err)
	}
	return k
}

func TestPlatformKeystroke_Parse(t *testing.T) {
	cases := []struct {
		source string
		want   Keystroke
	}{
		{"a", Keystroke{Key: "a"}},
		{"A", Keystroke{Key: "a", Modifiers: Modifiers{Shift: true}}},
		{"ctrl-shift-Tab", Keystroke{Key: "tab", Modifiers: Modifiers{Control: true, Shift: true}}},
		{"cmd-alt-fn-x", Keystroke{Key: "x", Modifiers: Modifiers{Platform: true, Alt: true, Function: true}}},
		{"ctrl--", Keystroke{Key: "-", Modifiers: Modifiers{Control: true}}},
		{"-", Keystroke{Key: "-"}},
		{"alt-s->ß", Keystroke{Key: "s", KeyChar: Some("ß"), Modifiers: Modifiers{Alt: true}}},
		{"shift", Keystroke{Key: "shift"}},
		{"ctrl-shift", Keystroke{Key: "shift", Modifiers: Modifiers{Control: true}}},
		{"SUPER-WIN-k", Keystroke{Key: "k", Modifiers: Modifiers{Platform: true}}},
	}
	for _, c := range cases {
		if got := mustParse(t, c.source); got != c.want {
			t.Errorf("parse %q = %+v, want %+v", c.source, got, c.want)
		}
	}
	for _, bad := range []string{"a-b", "secondary-a", "ſhift-a"} {
		_, err := KeystrokeParse(bad)
		if bad == "ſhift-a" {
			if err == nil {
				t.Errorf("parse %q: want error", bad)
			}
			continue
		}
		if err == nil || err.Error() != "Invalid keystroke \""+bad+"\". "+KeystrokeParseExpectedMessage {
			t.Errorf("parse %q: err %v", bad, err)
		}
	}
	if k, err := KeystrokeParseWithOs("secondary-s", OperatingSystemMac); err != nil || k.Modifiers != ModifiersCommand() {
		t.Errorf("mac secondary: %+v %v", k, err)
	}
	if k, err := KeystrokeParseWithOs("secondary-s", OperatingSystemLinux); err != nil || k.Modifiers != ModifiersControl() {
		t.Errorf("linux secondary: %+v %v", k, err)
	}
}

func TestPlatformKeystroke_UnparseAndDisplay(t *testing.T) {
	k := mustParse(t, "fn-ctrl-alt-cmd-shift-up")
	if got := k.Unparse(OperatingSystemLinux); got != "fn-ctrl-alt-super-shift-up" {
		t.Errorf("unparse linux %q", got)
	}
	if got := k.Unparse(OperatingSystemWindows); got != "fn-ctrl-alt-win-shift-up" {
		t.Errorf("unparse windows %q", got)
	}
	if got := k.Unparse(OperatingSystemMac); got != "fn-ctrl-alt-cmd-shift-up" {
		t.Errorf("unparse mac %q", got)
	}
	if got := k.Display(OperatingSystemMac).String(); got != "^⌥⌘⇧↑" {
		t.Errorf("display mac %q", got)
	}
	if got := k.Display(OperatingSystemLinux).String(); got != "ctrl-alt-❖shift-up" {
		t.Errorf("display linux %q", got)
	}
	if got := mustParse(t, "cmd-a").Display(OperatingSystemIos).String(); got != "cmd-A" {
		t.Errorf("display ios %q", got)
	}
}

func TestPlatformKeystroke_ShouldMatch(t *testing.T) {
	typed := Keystroke{Key: "4", KeyChar: Some("$"), Modifiers: Modifiers{Alt: true, Shift: true}}
	target := keybindingKeystrokeFromKeystroke(Keystroke{Key: "$"}, false)
	if !typed.shouldMatch(target, false) || !typed.shouldMatch(target, true) {
		t.Error("key_char match")
	}
	ctrlTarget := keybindingKeystrokeFromKeystroke(Keystroke{Key: "$", Modifiers: Modifiers{Control: true}}, false)
	typedCtrl := Keystroke{Key: "4", KeyChar: Some("$"), Modifiers: Modifiers{Control: true, Alt: true}}
	if !typedCtrl.shouldMatch(ctrlTarget, false) {
		t.Error("ime modifiers match")
	}
	if typedCtrl.shouldMatch(ctrlTarget, true) {
		t.Error("windows requires no modifiers")
	}
	exact := keybindingKeystrokeFromKeystroke(typed, true)
	if !typed.shouldMatch(exact, true) {
		t.Error("exact match")
	}
	w := keybindingKeystrokeFromKeystroke(Keystroke{Key: "a", Modifiers: ModifiersShift()}, true)
	if w.key(true) != "a" || w.modifiers(true) != ModifiersShift() {
		t.Error("windows display fields")
	}
}

func TestPlatformKeystroke_IMEAndModifiers(t *testing.T) {
	if got := mustParse(t, "space").WithSimulatedIME().KeyChar; got != Some(" ") {
		t.Errorf("space %v", got)
	}
	if got := mustParse(t, "shift-a").WithSimulatedIME().KeyChar; got != Some("A") {
		t.Errorf("shift-a %v", got)
	}
	if got := mustParse(t, "f1").WithSimulatedIME().KeyChar; got.IsSome() {
		t.Errorf("f1 %v", got)
	}
	if got := mustParse(t, "ctrl-a").WithSimulatedIME().KeyChar; got.IsSome() {
		t.Errorf("ctrl-a %v", got)
	}
	if !mustParse(t, "a").IsIMEInProgress() || mustParse(t, "ctrl-a").IsIMEInProgress() {
		t.Error("ime in progress")
	}
	if !mustParse(t, "shift").IsModifierKey() {
		t.Error("modifier key")
	}
	m := ModifiersAlt().BitOr(ModifiersShift())
	if m.NumberOfModifiers() != 2 || !ModifiersShift().IsSubsetOf(m) || m.IsSubsetOf(ModifiersShift()) {
		t.Errorf("modifiers %+v", m)
	}
	if m.BitXor(ModifiersShift()) != ModifiersAlt() || m.BitAnd(ModifiersShift()) != ModifiersShift() {
		t.Errorf("xor/and %+v", m)
	}
	var k Keystroke
	if err := json.Unmarshal([]byte(`{"modifiers":{"control":true},"key":"a","key_char":"a"}`), &k); err != nil {
		t.Fatal(err)
	}
	if k != (Keystroke{Key: "a", KeyChar: Some("a"), Modifiers: ModifiersControl()}) {
		t.Errorf("json %+v", k)
	}
}
