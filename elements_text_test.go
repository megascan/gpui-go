package gpui

import "testing"

func TestText_IntoElementFor(t *testing.T) {
	_ = NewDiv().ChildText("static str")
	_ = NewDiv().ChildText("String")
	_ = NewDiv().ChildText("Cow")
	_ = NewDiv().Child(SharedString("SharedString"))
}

func makeTextStableID(happy bool) Text {
	s := SharedString("sad")
	if happy {
		s = "happy"
	}
	return TextMacro(s)
}

func makeTextUnstableID(happy bool) Text {
	if happy {
		return TextMacro("happy")
	}
	return TextMacro("sad")
}

func TestText_TextMacroID(t *testing.T) {
	if makeTextStableID(false).id != makeTextStableID(true).id {
		t.Fatal("one call to TextMacro must give one id")
	}
	if makeTextUnstableID(false).id == makeTextUnstableID(true).id {
		t.Fatal("two calls to TextMacro must give two ids")
	}
}
