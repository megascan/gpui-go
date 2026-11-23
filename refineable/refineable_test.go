package refineable

import (
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
)

type eqByLen string

func (a eqByLen) Eq(b eqByLen) bool { return len(a) == len(b) }

func TestRefineable_Eq(t *testing.T) {
	if !Eq(1, 1) || Eq(1, 2) || !Eq([]int{1}, []int{1}) || !Eq(eqByLen("ab"), eqByLen("cd")) {
		t.Fatal("Eq")
	}
	if !OptionEq(gpuiutil.None[int](), gpuiutil.None[int]()) || OptionEq(gpuiutil.Some(0), gpuiutil.None[int]()) || !OptionEq(gpuiutil.Some([]int{2}), gpuiutil.Some([]int{2})) {
		t.Fatal("OptionEq")
	}
}

func TestRefineable_DebugStruct(t *testing.T) {
	d := NewDebugStruct("A")
	d.Field("x", 1)
	d.Field("y", gpuiutil.Some(2))
	if got := d.Finish(); got != "A { x: 1, y: Some(2) }" {
		t.Fatal(got)
	}
	if got := NewDebugStruct("B").Finish(); got != "B" {
		t.Fatal(got)
	}
}
