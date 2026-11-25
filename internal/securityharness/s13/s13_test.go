//go:build securityharness

package s13

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/rand/v2"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/jsonschema"
)

const seed = 0x5513

var alphabet = []string{"a", "b", "-", "--", "->", ">", "<", "(", ")", "!", "&&", "||", "==", "!=", "=", " ", "\t", " ", "~", "?", "\"", "ctrl", "shift", "secondary", "cmd", "alt", "fn", "A", "é", "‮", "\xff", "\x00", "日本", "_", "1", "́"}

func randomInput(rng *rand.Rand, max int) string {
	var b strings.Builder
	n := rng.IntN(max + 1)
	for range n {
		b.WriteString(alphabet[rng.IntN(len(alphabet))])
	}
	return b.String()
}

func fuzz(t *testing.T, f func(string)) {
	t.Helper()
	t.Logf("seed=%#x", seed)
	rng := rand.New(rand.NewPCG(seed, seed))
	for i := range 200000 {
		s := randomInput(rng, 24)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC iteration %d input %q: %v", i, s, r)
				}
			}()
			f(s)
		}()
	}
}

func TestS13_KeystrokeParseFuzz(t *testing.T) {
	fuzz(t, func(s string) {
		for _, os := range []gpui.OperatingSystem{gpui.OperatingSystemMac, gpui.OperatingSystemLinux, gpui.OperatingSystemWindows} {
			k, err := gpui.KeystrokeParseWithOs(s, os)
			if err == nil {
				_ = k.Unparse(os)
				_ = k.Display(os).String()
			}
		}
		_, _ = gpui.KeystrokeParse(s)
	})
}

func TestS13_PredicateParseFuzz(t *testing.T) {
	ctx, _ := gpui.KeyContextParse("a b=c Editor")
	contexts := []gpui.KeyContext{ctx, ctx}
	fuzz(t, func(s string) {
		p, err := gpui.KeyBindingContextPredicateParse(s)
		if err == nil {
			_ = fmt.Sprint(p)
			_ = p.Eval(contexts)
			_, _ = p.DepthOf(contexts)
			_ = p.IsSuperset(p)
		}
	})
}

func deepPredicate(kind string, n int) string {
	switch kind {
	case "parens":
		return strings.Repeat("(", n) + "a" + strings.Repeat(")", n)
	case "not":
		return strings.Repeat("!", n) + "a"
	case "unclosed":
		return strings.Repeat("(", n)
	}
	return ""
}

func predicateDepthBody(arg string) (string, error) {
	parts := strings.Split(arg, ":")
	kind := parts[0]
	n, _ := strconv.Atoi(parts[1])
	if len(parts) > 2 {
		mb, _ := strconv.Atoi(parts[2])
		debug.SetMaxStack(mb << 20)
	}
	src := deepPredicate(kind, n)
	start := time.Now()
	_, err := gpui.KeyBindingContextPredicateParse(src)
	if err != nil {
		return "", fmt.Errorf("%d bytes parsed to error in %s: %v", len(src), time.Since(start), err)
	}
	return fmt.Sprintf("%d bytes parsed in %s", len(src), time.Since(start)), nil
}

func TestS13_PredicateDeepNesting10k(t *testing.T) {
	for _, kind := range []string{"parens", "not"} {
		o, parent := bombchild.Run(t, childLimit, 60*time.Second, kind+":10000", predicateDepthBody)
		if !parent {
			return
		}
		bombchild.ExpectSafe(t, o, "predicate depth 10000 "+kind)
	}
}

func TestS13_PredicateDeepNesting4M(t *testing.T) {
	for _, kind := range []string{"parens", "not", "unclosed"} {
		o, parent := bombchild.Run(t, childLimit, 60*time.Second, kind+":4000000", predicateDepthBody)
		if !parent {
			return
		}
		bombchild.ExpectSafe(t, o, "predicate depth 4e6 "+kind)
	}
}

func TestS13_PredicateDepthThreshold(t *testing.T) {
	for _, n := range []int{25000, 50000, 100000, 200000} {
		o, parent := bombchild.Run(t, childLimit, 60*time.Second, fmt.Sprintf("parens:%d:64", n), predicateDepthBody)
		if !parent {
			return
		}
		t.Logf("THRESHOLD parens n=%d at 64 MiB max stack: %s", n, o.Verdict)
	}
}

func TestS13_KeyContextParseNonIdentifier(t *testing.T) {
	for _, in := range []string{"(", "a .", "Editor mode=vim :", "a=b ="} {
		o, parent := bombchild.Run(t, childLimit, 60*time.Second, in, func(arg string) (string, error) {
			start := time.Now()
			ctx, err := gpui.KeyContextParse(arg)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("parsed %q to %v in %s", arg, ctx, time.Since(start)), nil
		})
		if !parent {
			return
		}
		bombchild.ExpectSafe(t, o, fmt.Sprintf("KeyContextParse(%q)", in))
	}
}

func TestS13_ParseRgbaEdgeCases(t *testing.T) {
	cases := []string{"", "#", "#f", "#ff", "#fff", "#ffff", "#fffff", "#ffffff", "#fffffff", "#ffffffff", "#fffffffff",
		"# fff", " #fff ", "#+f+f+f", "#+0+0+0+0", "#ééé", "#éé", "#\xff\xff\xff", "#‮fff", "fff#", "##fff", "#gg0000",
		"#-1-1-1", "#ff\x00000", strings.Repeat("#", 1<<16), "#" + strings.Repeat("f", 1<<20)}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ParseRgba(%q) panicked: %v", c, r)
				}
			}()
			v, err := gpui.ParseRgba(c)
			var j gpui.Rgba
			jerr := json.Unmarshal([]byte(strconv.Quote(c)), &j)
			if (err == nil) != (jerr == nil) {
				t.Errorf("ParseRgba(%q) err=%v but json err=%v", c, err, jerr)
			}
			if err == nil {
				t.Logf("ParseRgba(%q) = %v", trunc(c), v)
			}
		}()
	}
}

func TestS13_ParseRgbaFuzz(t *testing.T) {
	t.Logf("seed=%#x", seed)
	rng := rand.New(rand.NewPCG(seed, 2))
	chars := []byte("#0123456789abcdefABCDEF+- \t\xff\xc3\xa9g")
	for i := range 300000 {
		n := rng.IntN(12)
		b := make([]byte, n)
		for j := range b {
			b[j] = chars[rng.IntN(len(chars))]
		}
		s := string(b)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("iteration %d ParseRgba(%q) panicked: %v", i, s, r)
				}
			}()
			_, _ = gpui.ParseRgba(s)
		}()
	}
}

func TestS13_FontFeaturesJSON(t *testing.T) {
	cases := []string{`{}`, `{"liga":true}`, `{"liga":4294967296}`, `{"liga":4294967297}`, `{"liga":18446744073709551615}`,
		`{"liga":18446744073709551616}`, `{"liga":-1}`, `{"liga":1e9}`, `{"liga":1.5}`, `{"liga":"1"}`, `{"liga":{}}`, `{"liga":[]}`,
		`{"lig":1}`, `{"ligaa":1}`, `{"li\u0000a":1}`, `{"lîga":1}`, `{"liga":null}`, `[]`, `null`, `{"liga":1,"liga":0}`, `{"liga"`, `{`}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("FontFeatures %s panicked: %v", c, r)
				}
			}()
			var f gpui.FontFeatures
			err := json.Unmarshal([]byte(c), &f)
			t.Logf("FontFeatures %s -> %v err=%v", c, f.TagValueList(), err)
			if c == `{"liga":4294967296}` && err == nil {
				for _, ft := range f.TagValueList() {
					if ft.Tag == "liga" && ft.Value != 4294967295 {
						t.Errorf("INFO-WRAP: liga 4294967296 silently stored as %d (uint32 wrap, not clamped or rejected)", ft.Value)
					}
				}
			}
		}()
	}
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse
}

func TestS13_InternTablesRetainDistinctInputs(t *testing.T) {
	const n = 100000
	before := heapInUse()
	for i := range n {
		var f gpui.FontFeatures
		if err := json.Unmarshal(fmt.Appendf(nil, `{"liga":%d,"calt":%d}`, i, i^0x5a5a), &f); err != nil {
			t.Fatal(err)
		}
		var fb gpui.FontFallbacks
		if err := json.Unmarshal(fmt.Appendf(nil, `["Font %d with a moderately long family name to look like settings"]`, i), &fb); err != nil {
			t.Fatal(err)
		}
	}
	after := heapInUse()
	retained := int64(after) - int64(before)
	t.Logf("retained after %d distinct FontFeatures+FontFallbacks unmarshals (all values dropped, GC run): %d bytes (%.0f B/input)", n, retained, float64(retained)/n)
	if retained > 8<<20 {
		t.Errorf("LEAK: global intern tables retain %d bytes for %d dropped values (no eviction)", retained, n)
	}
}

func TestS13_BuildActionHostileJSON(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		inputs := map[string]string{
			"deep-array":  strings.Repeat("[", 200000) + strings.Repeat("]", 200000),
			"deep-object": strings.Repeat(`{"a":`, 200000) + "1" + strings.Repeat("}", 200000),
			"string":      `"editor::NewLine"`,
			"object":      `{"action":"x"}`,
			"invalid":     `{`,
			"nul":         "\x00",
			"big-number":  `1e999999`,
		}
		for name, in := range inputs {
			for _, action := range []string{"zed::Unbind", "zed::NoAction", "no::Such", ""} {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("BuildAction(%q, %s) panicked: %v", action, name, r)
						}
					}()
					start := time.Now()
					err := cx.UpdateR(func(cx *gpui.App) error {
						_, err := cx.BuildAction(action, gpui.Some(jsontext.Value(in)))
						return err
					})
					t.Logf("BuildAction(%q, %s): err=%v in %s", action, name, trunc(fmt.Sprint(err)), time.Since(start))
				}()
			}
		}
	})
}

type namedCycle struct {
	Next *namedCycle
	Kids []namedCycle
	M    map[string]namedCycle
}

type recSlice []recSlice

type recMap map[string]recMap

type recPtrStruct struct {
	L recSlice
}

func TestS13_SchemaNamedStructCycle(t *testing.T) {
	done := make(chan *jsonschema.Schema, 1)
	go func() { done <- jsonschema.NewGenerator().Reflect(reflect.TypeFor[namedCycle]()) }()
	select {
	case s := <-done:
		if s == nil {
			t.Fatal("nil schema")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("schema generation for a named struct cycle did not terminate")
	}
}

func TestS13_SchemaRecursiveNonStruct(t *testing.T) {
	for _, name := range []string{"slice", "map", "nested"} {
		o, parent := bombchild.Run(t, childLimit, 60*time.Second, name, func(arg string) (string, error) {
			var ty reflect.Type
			switch arg {
			case "slice":
				ty = reflect.TypeFor[recSlice]()
			case "map":
				ty = reflect.TypeFor[recMap]()
			default:
				ty = reflect.TypeFor[recPtrStruct]()
			}
			s := jsonschema.NewGenerator().Reflect(ty)
			return fmt.Sprintf("schema type %v", s.Type), nil
		})
		if !parent {
			return
		}
		if !o.Safe() {
			t.Logf("INFO SEC-038 (unfixed by design, TECH_DEBT SEC-038): jsonschema Reflect recursive %s: %s", name, o)
		}
	}
}

func trunc(s string) string {
	if len(s) > 80 {
		return fmt.Sprintf("%s...(%d bytes)", s[:80], len(s))
	}
	return s
}
