package matrix

import (
	"strings"
	"testing"
)

const foundRow = "| `Foo::bar` | method | 2 |  | `(Foo).Bar` | todo |"

func withStatus(sym, status string) string {
	return strings.Replace(fixtureArea, "| `Foo::bar` | method | 2 |  | `(Foo).Bar` | todo |", "| `Foo::bar` | method | 2 |  | `"+sym+"` | "+status+" |", 1)
}

func mustLoadRoot(t *testing.T, root string) *Matrix {
	t.Helper()
	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func hasProblem(ps []Problem, sub string) bool {
	for _, p := range ps {
		if strings.Contains(p.Msg, sub) {
			return true
		}
	}
	return false
}

func TestLoad_ParsesRowsTargetsAndRegistry(t *testing.T) {
	m := mustLoadRoot(t, newFixture(t, nil))
	a := m.Area("a")
	if a == nil || len(a.Rows) != 3 {
		t.Fatalf("want area a with 3 rows, got %+v", m.Areas)
	}
	if a.Rows[0].Target != "." || a.Rows[0].Status != "done" || a.Rows[0].Test {
		t.Errorf("first row: %+v", a.Rows[0])
	}
	if !a.Rows[2].Test || a.Rows[2].Symbol != "TestA_Foo" {
		t.Errorf("test row: %+v", a.Rows[2])
	}
	if !m.IsRegistered("L-GLYPH") || !m.IsRegistered("D-NAME") || m.IsRegistered("D-NOPE") {
		t.Errorf("registry: %v %v", m.Registered, a.Deviations)
	}
	if !m.Styled["W0p5"] {
		t.Errorf("styled methods: %v", m.Styled)
	}
}

func TestLoad_RejectsBadStatus(t *testing.T) {
	_, err := Load(newFixture(t, map[string]string{"docs/port/parity/a.md": withStatus("(Foo).Bar", "todo dev:D-NAME")}))
	if err == nil || !strings.Contains(err.Error(), "bad status") {
		t.Fatalf("want bad status error, got %v", err)
	}
}

func TestCheck_IDRegistration(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		problem string
	}{
		{"registered deviation", "dev:D-NAME", ""},
		{"registered limit", "n/a:L-GLYPH", ""},
		{"unregistered id", "dev:D-MISSING", "not registered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mustLoadRoot(t, newFixture(t, map[string]string{"docs/port/parity/a.md": withStatus("(Foo).Bar", tt.status)}))
			rep, err := Check(m, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if tt.problem == "" && len(rep.Problems) != 0 {
				t.Fatalf("unexpected problems: %v", rep.Problems)
			}
			if tt.problem != "" && !hasProblem(rep.Problems, tt.problem) {
				t.Fatalf("want %q, got %v", tt.problem, rep.Problems)
			}
		})
	}
}

func TestCheck_Options(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		opt     Options
		wantMsg string
	}{
		{"todo under all-done", "todo", Options{AllDone: true}, "todo row under -all-done"},
		{"todo under done prefix", "todo", Options{Done: []string{"src/"}}, "todo row under -done"},
		{"done prefix not matched", "done", Options{Done: []string{"zzz/"}}, "matches no upstream file"},
		{"blocked fails all-done", "blocked:M1-x", Options{AllDone: true}, "blocked row under -all-done"},
		{"blocked tolerated", "blocked:M1-x", Options{AllDone: true, AllowBlocked: true}, ""},
		{"owner fails all-done", "blocked:owner", Options{AllDone: true, AllowBlocked: true}, "blocked:owner is not allowed"},
		{"unknown area", "done", Options{Areas: []string{"nope"}}, "unknown area"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mustLoadRoot(t, newFixture(t, map[string]string{"docs/port/parity/a.md": withStatus("(Foo).Bar", tt.status)}))
			rep, err := Check(m, tt.opt)
			if tt.wantMsg == "" {
				if err != nil || len(rep.Problems) != 0 {
					t.Fatalf("want clean, got err=%v problems=%v", err, rep.Problems)
				}
				return
			}
			if err != nil {
				if !strings.Contains(err.Error(), tt.wantMsg) {
					t.Fatalf("err %v, want %q", err, tt.wantMsg)
				}
				return
			}
			if !hasProblem(rep.Problems, tt.wantMsg) {
				t.Fatalf("want %q, got %v", tt.wantMsg, rep.Problems)
			}
		})
	}
}

func TestCheck_OwnerWarningOutsideAllDone(t *testing.T) {
	m := mustLoadRoot(t, newFixture(t, map[string]string{"docs/port/parity/a.md": withStatus("(Foo).Bar", "blocked:owner")}))
	rep, err := Check(m, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Problems) != 0 || len(rep.Warnings) != 1 {
		t.Fatalf("want one warning, got problems=%v warnings=%v", rep.Problems, rep.Warnings)
	}
}

func TestCheck_DoneSymbolAndCounts(t *testing.T) {
	m := mustLoadRoot(t, newFixture(t, map[string]string{"docs/port/parity/a.md": withStatus("(Foo).Missing", "done")}))
	rep, err := Check(m, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasProblem(rep.Problems, "not found in .") {
		t.Fatalf("want missing symbol, got %v", rep.Problems)
	}

	short := strings.Replace(fixtureParity, "| 2 | 1 |", "| 3 | 1 |", 1)
	m = mustLoadRoot(t, newFixture(t, map[string]string{"docs/port/PARITY.md": short}))
	rep, err = Check(m, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasProblem(rep.Problems, "item rows") {
		t.Fatalf("want count shortfall, got %v", rep.Problems)
	}
}

func TestSymbolIn(t *testing.T) {
	set := symsetFromSource(t, `package p
type Foo struct{ Field int }
func (f *Foo) Bar() {}
type Gen[S any] struct{}
func (g *Gen[S]) Get() {}
type Iface interface{ Name() string }
func NewFoo() *Foo { return nil }
const Limit = 3
`)
	tests := map[string]bool{
		"Foo":           true,
		"NewFoo":        true,
		"Limit":         true,
		"(Foo).Bar":     true,
		"(*Foo).Bar":    true,
		"(Gen[S]).Get":  true,
		"Foo.Bar":       true,
		"Foo.Field":     true,
		"Iface.Name":    true,
		"Foo.Missing":   false,
		"Missing":       false,
		"-":             false,
		"":              false,
		"(Foo).Missing": false,
		"`Foo`":         true,
	}
	for sym, want := range tests {
		if got := symbolIn(set, sym); got != want {
			t.Errorf("symbolIn(%q) = %v, want %v", sym, got, want)
		}
	}
}
