package taffy

import (
	"iter"
	"slices"
	"strings"
	"testing"
)

type printTestTree struct {
	children map[NodeID][]NodeID
	layouts  map[NodeID]Layout
}

func (p *printTestTree) ChildIds(parent NodeID) iter.Seq[NodeID] {
	return slices.Values(p.children[parent])
}

func (p *printTestTree) ChildCount(parent NodeID) int {
	return len(p.children[parent])
}

func (p *printTestTree) GetChildID(parent NodeID, index int) NodeID {
	return p.children[parent][index]
}

func (p *printTestTree) TraverseTree() {}

func (p *printTestTree) GetDebugLabel(NodeID) string {
	return "FLEX ROW"
}

func (p *printTestTree) GetFinalLayout(id NodeID) Layout {
	return p.layouts[id]
}

func TestUtilPrint_WriteTree(t *testing.T) {
	root, a, b := NewNodeID(0), NewNodeID(1), NewNodeID(2)
	l := NewLayout()
	l.Size.Width = 1.5
	tree := &printTestTree{
		children: map[NodeID][]NodeID{root: {a, b}},
		layouts:  map[NodeID]Layout{root: l, a: NewLayout(), b: NewLayout()},
	}
	var sb strings.Builder
	if err := WriteTree(&sb, tree, root); err != nil {
		t.Fatal(err)
	}
	want := "TREE\n" +
		"└──  FLEX ROW [x: 0    y: 0    w: 1.5  h: 0    overflow: l:0 r:0 t:0 b:0, border: l:0 r:0 t:0 b:0, padding: l:0 r:0 t:0 b:0] (NodeId(0))\n" +
		"    ├──  FLEX ROW [x: 0    y: 0    w: 0    h: 0    overflow: l:0 r:0 t:0 b:0, border: l:0 r:0 t:0 b:0, padding: l:0 r:0 t:0 b:0] (NodeId(1))\n" +
		"    └──  FLEX ROW [x: 0    y: 0    w: 0    h: 0    overflow: l:0 r:0 t:0 b:0, border: l:0 r:0 t:0 b:0, padding: l:0 r:0 t:0 b:0] (NodeId(2))\n"
	if got := sb.String(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}
