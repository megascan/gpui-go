package matrix

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

const fixtureParity = `# Parity

## 2. Totals

| Area file | Upstream scope | Milestone | Go paths owned | Files | Public items | Tests |
|---|---|---|---|---|---|---|
| [parity/a.md](parity/a.md) | a | M0 | root | 1 | 2 | 1 |

| Total | | | | | 2 | 1 |

## 5. Deviations and limits

| Id | Limit | Upstream | Source |
|---|---|---|---|
| L-GLYPH | x | y | z |

## 6. Examples
`

const fixtureArea = `# Parity: a

## src/a.rs

Go target: ` + "`a.go`" + ` (package ` + "`gpui`" + `), tests in ` + "`a_test.go`" + `

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| ` + "`Foo`" + ` | struct | 1 |  | ` + "`Foo`" + ` | done |
| ` + "`Foo::bar`" + ` | method | 2 |  | ` + "`(Foo).Bar`" + ` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| ` + "`test_foo`" + ` | test | 9 |  | ` + "`TestA_Foo`" + ` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-NAME | x | y | z |
`

const fixtureGo = `package gpui

type Foo struct{ Field int }

func (f *Foo) Bar() {}
`

const fixtureGoTest = `package gpui

import "testing"

func TestA_Foo(t *testing.T) {}
`

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func baseTree() map[string]string {
	return map[string]string{
		"go.mod":                              "module example.com/m\n\ngo 1.27\n",
		"docs/port/PARITY.md":                 fixtureParity,
		"docs/port/parity/a.md":               fixtureArea,
		"docs/port/parity/styled-methods.tsv": "# rust_method\tgo_method\tmacro\tsource\nw_0p5\tW0p5\tstyle_helpers\tstyles.rs:1\n",
		"a.go":                                fixtureGo,
		"a_test.go":                           fixtureGoTest,
	}
}

func newFixture(t *testing.T, edits map[string]string) string {
	t.Helper()
	root := t.TempDir()
	tree := baseTree()
	for k, v := range edits {
		if v == "" {
			delete(tree, k)
			continue
		}
		tree[k] = v
	}
	writeTree(t, root, tree)
	return root
}

func symsetFromSource(t *testing.T, src string) symset {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	s := newSymset()
	collect(f, s)
	return s
}
