package scan

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/internal/cmd/paritycheck/matrix"
)

func names(items []Item, test bool) []string {
	var out []string
	for _, it := range items {
		if it.Test == test {
			out = append(out, it.Kind+" "+it.Name)
		}
	}
	return out
}

func TestScan_Lexer(t *testing.T) {
	src := "// fn hidden() {}\n/* nested /* fn x() */ */ let s = \"pub fn y() { \\\" }\"; let r = r#\"pub fn z() {\"#; 'a' 'b: lt '\\n'\nfoo"
	toks := lex(src)
	for _, tk := range toks {
		if tk.kind == tIdent && (tk.text == "hidden" || tk.text == "x" || tk.text == "y" || tk.text == "z") {
			t.Fatalf("comment or string leaked token %q", tk.text)
		}
	}
	last := toks[len(toks)-1]
	if last.text != "foo" || last.line != 3 {
		t.Fatalf("last token %+v, want foo on line 3", last)
	}
	var lifetimes int
	for _, tk := range toks {
		if tk.kind == tLifetime {
			lifetimes++
		}
	}
	if lifetimes != 1 {
		t.Fatalf("lifetimes = %d, want 1", lifetimes)
	}
}

const fixture = `
pub struct Foo;
struct Hidden;
pub(crate) fn crate_fn() {}
pub fn free() {}
pub const C: u32 = 1;
pub type Alias = Foo;
pub trait Tr {
    fn required(&self);
    fn provided() {}
}
trait Private {
    fn nope(&self);
}
impl Foo {
    pub fn method(&self) {}
    pub fn assoc() -> Self { Foo }
    pub const K: u8 = 0;
    fn private(&self) {}
}
impl Tr for Foo {
    fn required(&self) {}
}
impl Hidden {
    pub fn on_private(&self) {}
}
impl Point<Pixels> {
    pub fn scale(&self) {}
}
impl<T> Point<T> {
    pub fn generic(&self) {}
}
#[macro_export]
macro_rules! exported {
    () => {};
}
#[cfg(test)]
pub fn test_only() {}
#[cfg(any(test, feature = "test-support"))]
pub fn test_support() {}
#[cfg(target_os = "linux")]
pub fn linux_only() {}
mod tests {
    pub fn helper() {}
    #[test]
    fn plain() {}
    #[gpui::test(iterations = 10)]
    async fn gpui_test(cx: &mut TestAppContext) {}
}
bitflags! {
    pub struct Flags: u32 { const A = 1; }
}
proptest! {
    #[test]
    fn in_macro(x in 0..1) {}
}
`

func TestScan_Rules(t *testing.T) {
	fr := parseFile("c", "c/src/lib.rs", fixture)
	got := names(fr.items, false)
	for _, ii := range fr.implItems {
		if ii.typ != "Hidden" {
			got = append(got, ii.item.Kind+" "+ii.item.Name)
		}
	}
	want := []string{
		"struct Foo", "fn free", "const C", "type Alias", "trait Tr",
		"trait-method Tr::required", "trait-method Tr::provided",
		"macro exported", "fn test_support", "fn linux_only",
		"method Foo::method", "assoc-fn Foo::assoc", "assoc-const Foo::K",
		"method Point<Pixels>::scale", "method Point::generic",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("items\n got %q\nwant %q", got, want)
	}
	tests := names(fr.items, true)
	if !slices.Equal(tests, []string{"test plain", "gpui::test(iterations = 10) gpui_test"}) {
		t.Fatalf("tests = %q", tests)
	}
	if len(fr.warnings) != 2 {
		t.Fatalf("warnings = %v, want bitflags and proptest", fr.warnings)
	}
	for _, it := range fr.items {
		if it.Name == "linux_only" && !slices.Equal(it.Cfg, []string{"linux"}) {
			t.Fatalf("linux_only cfg = %q", it.Cfg)
		}
	}
}

func TestScan_TestFiles(t *testing.T) {
	for _, rel := range []string{"gpui/src/tests.rs", "gpui/src/foo_tests.rs", "gpui/tests/a.rs", "gpui/benches/b.rs", "gpui/examples/c/main.rs"} {
		if !isTestFile(rel) {
			t.Errorf("%s should be a test file", rel)
		}
	}
	fr := parseFile("gpui", "gpui/examples/x.rs", "pub fn main() {}\n#[test]\nfn t() {}\n")
	if got := names(fr.items, false); len(got) != 0 {
		t.Fatalf("example items = %q", got)
	}
	if got := names(fr.items, true); len(got) != 1 {
		t.Fatalf("example tests = %q", got)
	}
}

func TestScan_ProcMacros(t *testing.T) {
	src := "#[proc_macro_derive(Action, attributes(action))]\npub fn derive_action(i: TokenStream) -> TokenStream { i }\n#[proc_macro]\npub fn register(i: TokenStream) -> TokenStream { i }\npub fn helper() {}\n"
	fr := parseFile("gpui_macros", "gpui_macros/src/lib.rs", src)
	got := names(fr.items, false)
	if !slices.Equal(got, []string{"proc-macro-derive Action", "proc-macro register"}) {
		t.Fatalf("proc macros = %q", got)
	}
}

func TestScan_ExternBlockItemsAreDeclaredNotCounted(t *testing.T) {
	src := "#[link(name = \"AppKit\", kind = \"framework\")]\nunsafe extern \"C\" {\n    pub static NSAppearanceNameAqua: *mut AnyObject;\n    pub(super) static kTISPropertyInputSourceID: CFStringRef;\n    pub fn CVDisplayLinkStart(link: *mut c_void) -> i32;\n    safe fn LMGetKbdType() -> u16;\n    pub type Opaque;\n}\npub fn after() {}\n"
	fr := parseFile("gpui_macos", "gpui_macos/src/platform.rs", src)
	if got := names(fr.items, false); !slices.Equal(got, []string{"fn after"}) {
		t.Fatalf("items = %q", got)
	}
	for _, n := range []string{"NSAppearanceNameAqua", "kTISPropertyInputSourceID", "CVDisplayLinkStart", "LMGetKbdType", "Opaque", "after"} {
		if !fr.decls[n] {
			t.Fatalf("%s is not declared", n)
		}
	}
}

func refDir(t *testing.T) string {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "_ref", "gpui_macros", "src", "styles.rs")); err != nil {
		t.Skip("_ref/ is absent")
	}
	return root
}

func TestScan_StylesMatchTSV(t *testing.T) {
	root := refDir(t)
	data, err := os.ReadFile(filepath.Join(root, "_ref", "gpui_macros", "src", "styles.rs"))
	if err != nil {
		t.Fatal(err)
	}
	styles, err := ExpandStyles(string(data))
	if err != nil {
		t.Fatal(err)
	}
	tsv, err := styledNames(filepath.Join(root, matrix.StyledTSV))
	if err != nil {
		t.Fatal(err)
	}
	if len(styles) != len(tsv) || len(styles) != 3099 {
		t.Fatalf("expanded %d methods, tsv has %d, want 3099", len(styles), len(tsv))
	}
	for _, s := range styles {
		if !tsv[s.Name] {
			t.Errorf("%s (%s) not in tsv", s.Name, s.Macro)
		}
	}
}

func TestScan_ReproducesSection2(t *testing.T) {
	root := refDir(t)
	res, err := Scan(filepath.Join(root, "_ref"))
	if err != nil {
		t.Fatal(err)
	}
	expect, err := crateTotals(filepath.Join(root, matrix.ParityFile))
	if err != nil {
		t.Fatal(err)
	}
	for c, want := range expect {
		if got := res.Crates[c]; got != want {
			t.Errorf("%s: got %+v, section 2 has %+v", c, got, want)
		}
	}
}

func TestScan_CompareReportsMissingAndGone(t *testing.T) {
	res := &Result{
		Items: []Item{{Crate: "c", File: "c/src/lib.rs", Name: "Foo", Kind: "struct"}, {Crate: "c", File: "c/src/lib.rs", Name: "bar", Kind: "fn"}},
		Files: map[string]bool{"c/src/lib.rs": true},
		Decls: map[string]map[string]bool{"c/src/lib.rs": {"Foo": true, "bar": true}},
	}
	m := &matrix.Matrix{Areas: []*matrix.Area{{Name: "a", Rows: []matrix.Row{
		{Heading: "c/src/lib.rs", Name: "Foo", Kind: "struct"},
		{Heading: "c/src/lib.rs", Name: "Gone", Kind: "fn"},
		{Heading: "c/src/lib.rs", Name: "Foo.field", Kind: "field"},
		{Heading: "c/src/old.rs", Name: "X", Kind: "fn"},
	}}}}
	rep := Compare(res, m, nil, nil)
	if len(rep.Missing) != 1 || rep.Missing[0].Name != "bar" {
		t.Fatalf("missing = %+v", rep.Missing)
	}
	if len(rep.Gone) != 1 || rep.Gone[0].Name != "Gone" {
		t.Fatalf("gone = %+v", rep.Gone)
	}
	if len(rep.NoFile) != 1 {
		t.Fatalf("nofile = %+v", rep.NoFile)
	}
	if rep := Compare(res, m, nil, []string{"other"}); len(rep.Missing)+len(rep.Gone)+len(rep.NoFile) != 0 {
		t.Fatalf("out-of-scope area reported %+v", rep)
	}
}

func TestScan_RescanMatrix(t *testing.T) {
	root := refDir(t)
	m, err := matrix.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	res, err := Scan(filepath.Join(root, "_ref"))
	if err != nil {
		t.Fatal(err)
	}
	if err := report(root, res, m, []string{"foundation", "sumtree", "scheduler"}, &buf); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "6706 public items, 869 tests") {
		t.Fatalf("totals line missing:\n%s", buf.String())
	}
}

func TestScan_AreasFromArgs(t *testing.T) {
	got := areasFromArgs([]string{"-rescan", "-area", "a, b", "-done", "x"})
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("got %q", got)
	}
	if got := areasFromArgs([]string{"--area=c"}); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("got %q", got)
	}
}
