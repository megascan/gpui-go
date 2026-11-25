//go:build linux

package xrm

import "testing"

func TestXrm_BoolTrue(t *testing.T) {
	for _, in := range []string{"1", "10", "true", "TRUE", "on", "ON", "yes", "YES"} {
		if v, ok := toBool(in); !ok || !v {
			t.Errorf("toBool(%q) = %v, %v", in, v, ok)
		}
	}
}

func TestXrm_BoolFalse(t *testing.T) {
	for _, in := range []string{"0", "false", "FALSE", "off", "OFF", "no", "NO"} {
		if v, ok := toBool(in); !ok || v {
			t.Errorf("toBool(%q) = %v, %v", in, v, ok)
		}
	}
}

func TestXrm_BoolNone(t *testing.T) {
	for _, in := range []string{"", "abc"} {
		if _, ok := toBool(in); ok {
			t.Errorf("toBool(%q) parsed", in)
		}
	}
}

func TestXrm_ParseI32Fail(t *testing.T) {
	db := NewFromData([]byte("a:"))
	if s, ok := db.GetString("a", "a"); !ok || s != "" {
		t.Fatalf("GetString = %q, %v", s, ok)
	}
	if _, _, err := db.GetInt32("a", "a"); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestXrm_ParseI32Success(t *testing.T) {
	for in, want := range map[string]int32{"a: 0": 0, "a: 1": 1, "a: -1": -1, "a: 100": 100} {
		v, ok, err := NewFromData([]byte(in)).GetInt32("a", "a")
		if err != nil || !ok || v != want {
			t.Errorf("%q: %v %v %v", in, v, ok, err)
		}
	}
}

func TestXrm_Float(t *testing.T) {
	v, ok, err := NewFromData([]byte("Xft.dpi: 144")).GetFloat32("Xft.dpi", "Xft.dpi")
	if err != nil || !ok || v != 144 {
		t.Fatalf("GetFloat32 = %v %v %v", v, ok, err)
	}
}

func TestXrm_Matches(t *testing.T) {
	none := "\x00none"
	tests := []struct{ data, resource, class, want string }{
		{"", "", "", none},
		{"First.second: 1", "First.second", "First.second.third", none},
		{"", "First.second", "", none},
		{"First.second: 1", "First.third", "", none},
		{"First.second: 1", "First", "", none},
		{"First: 1", "First.second", "", none},
		{"First.?.fourth: 1", "First.second.third.fourth", "", none},
		{"First*?.third: 1", "First.third", "", none},
		{"First: 1", "first", "", none},
		{"First: 1", "", "first", none},
		{"First: 1\nFirst: 2\nFirst: 3\n", "First", "", "3"},
		{"First: 1\nSecond: 2\nSecond: 3\nThird: 4\n", "Second", "", "3"},
		{"First: 1", "First", "", "1"},
		{"First.second: 1", "First.second", "", "1"},
		{"?.second: 1", "First.second", "", "1"},
		{"First.?.third: 1", "First.second.third", "", "1"},
		{"First.?.?.fourth: 1", "First.second.third.fourth", "", "1"},
		{"*second: 1", "First.second", "", "1"},
		{".second: 1", "First.second", "", none},
		{"*third: 1", "First.second.third", "", "1"},
		{"First*second: 1", "First.second", "", "1"},
		{"First*third: 1", "First.second.third", "", "1"},
		{"First*fourth: 1", "First.second.third.fourth", "", "1"},
		{"First*?.third: 1", "First.second.third", "", "1"},
		{"First: 1", "Second", "First", "1"},
		{"First.second: 1", "First.third", "first.second", "1"},
		{"First.second.third: 1", "First.third.third", "first.second.fourth", "1"},
		{"First*third*fifth: 1", "First.second.third.fourth.third.fifth", "", "1"},
		{"First: x\\\ny", "First", "", "xy"},
		{"! First: x", "First", "", none},
		{"# First: x", "First", "", none},
		{"First:", "First", "", ""},
		{"First: ", "First", "", ""},
		{"First: \t ", "First", "", ""},
		{"*.bar: 1", "foo.foo.bar", "", "1"},
		{"...bar: 1", "foo.bar", "", none},
		{"...bar: 1", "foo.foo.foo.bar", "", none},
		{"***bar: 1", "foo.bar", "", "1"},
		{".*.bar: 1", "foo.bar", "", "1"},
		{".*.bar: 1", "foo.foo.bar", "", "1"},
		{"..*bar: 1", "foo.foo.foo.foo.bar", "", "1"},
		{"a.*.z: 1", "a.b.c.d.e.f.z", "", "1"},
		{"a...z: 1", "a.z", "", "1"},
		{"a...z: 1", "a.b.z", "", none},
		{"First: 1\nSecond: 2\n", "First", "", "1"},
		{"First: 1\nSecond: 2\n", "Second", "", "2"},
		{"a*c.e: 1", "a.b.c.d.c.e", "", "1"},
		{"a*c.e: 1", "a.b.c.c.e", "", "1"},
		{"a*?.e: 1", "a.b.c.e", "", "1"},
		{"a*c*e: 1", "a.b.c.d.c.d.e.d.e", "", "1"},
		{"First.second.third: 1\nFirst*third: 2\n", "First.second.third", "", "1"},
		{"First*third: 2\nFirst.second.third: 1\n", "First.second.third", "", "1"},
		{"First.second.third: 1\nFirst*third: 2\n", "x.x.x", "First.second.third", "1"},
		{"First*third: 2\nFirst.second.third: 1\n", "x.x.x", "First.second.third", "1"},
		{"First.second: 1\nFirst.third: 2\n", "First.second", "First.third", "1"},
		{"First.third: 2\nFirst.second: 1\n", "First.second", "First.third", "1"},
		{"First.second.third: 1\nFirst.?.third: 2\n", "First.second.third", "", "1"},
		{"First.?.third: 2\nFirst.second.third: 1\n", "First.second.third", "", "1"},
		{"First.second.third: 1\nFirst.?.third: 2\n", "x.x.x", "First.second.third", "1"},
		{"First.?.third: 2\nFirst.second.third: 1\n", "x.x.x", "First.second.third", "1"},
		{"First.second: 1\nFirst*second: 2\n", "First.second", "", "1"},
		{"First*second: 2\nFirst.second: 1\n", "First.second", "", "1"},
		{"xmh*Paned*activeForeground: red\n*incorporate.Foreground: blue\nxmh.toc*Command*activeForeground: green\nxmh.toc*?.Foreground: white\nxmh.toc*Command.activeForeground: black", "xmh.toc.messagefunctions.incorporate.activeForeground", "Xmh.Paned.Box.Command.Foreground", "black"},
		{"urxvt*background: [95]#000", "urxvt.background", "", "[95]#000"},
		{"urxvt*scrollBar_right:true", "urxvt.scrollBar_right", "", "true"},
		{"urxvt*cutchars:    '\"'()*<>[]{|}", "urxvt.cutchars", "", "'\"'()*<>[]{|}"},
		{"urxvt.keysym.Control-Shift-Up: perl:font:increment", "urxvt.keysym.Control-Shift-Up", "", "perl:font:increment"},
		{"rofi.normal: #000000, #000000, #000000, #000000", "rofi.normal", "", "#000000, #000000, #000000, #000000"},
		{"*foo.bar: 1", "bar", "", none},
		{"First.Second.Third: 1\nFirst.Second: 2", "First.Second.Third", "First.Second", "1"},
		{"First.Second.Third: 1\nFirst.Second: 2", "First.Second", "First.Second.Third", "1"},
	}
	for _, tc := range tests {
		var entries []entry
		parseDatabase([]byte(tc.data), &entries, func([]byte, *[]entry) { t.Fatal("unreachable") })
		got, ok := matchEntry(entries, tc.resource, tc.class)
		if tc.want == none {
			if ok {
				t.Errorf("%q / %q / %q: got %q, want none", tc.data, tc.resource, tc.class, got)
			}
			continue
		}
		if !ok || string(got) != tc.want {
			t.Errorf("%q / %q / %q: got %q (%v), want %q", tc.data, tc.resource, tc.class, got, ok, tc.want)
		}
	}
}

func TestXrm_ParseEscapes(t *testing.T) {
	tests := []struct{ data, want string }{
		{"a: \\101", "A"},
		{"a: \\1x", "\\1x"},
		{"a: x\\ny", "x\ny"},
		{"a: \\\\", "\\"},
		{"a: \\", "\\"},
		{"a: \\ b", " b"},
	}
	for _, tc := range tests {
		got, ok := NewFromData([]byte(tc.data)).GetBytes("a", "a")
		if !ok || string(got) != tc.want {
			t.Errorf("%q: got %q, want %q", tc.data, got, tc.want)
		}
	}
}
