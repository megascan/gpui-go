package gpui

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

func a11yTestNode() a11y.Node {
	return a11y.NewNode(a11y.RoleGenericContainer)
}

func a11yTestNewBuilder() *a11yNodeBuilder {
	builder := newA11yNodeBuilder()
	builder.beginFrame(nil)
	return &builder
}

func a11yTestNewA11y() *windowA11y {
	flag := new(atomic.Bool)
	flag.Store(true)
	a := newWindowA11y(flag, false, None[SharedString]())
	a.beginFrame()
	return &a
}

func a11yTestShouldPanicInDebug(t *testing.T, expected string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if !gpuiutil.DebugAssertions {
			if r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
			return
		}
		if r == nil {
			t.Fatalf("expected a panic containing %q", expected)
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, expected) {
			t.Fatalf("panic %v does not contain %q", r, expected)
		}
	}()
	f()
}

func a11yTestAssert(t *testing.T, cond bool) {
	t.Helper()
	if !cond {
		t.Fatal("assertion failed")
	}
}

func TestA11y_AccessibilityEnabledIsIndependentOfActivation(t *testing.T) {
	for _, forceDisabled := range []bool{false, true} {
		for _, active := range []bool{false, true} {
			flag := new(atomic.Bool)
			flag.Store(active)
			a := newWindowA11y(flag, forceDisabled, None[SharedString]())
			a.syncActiveFlag()
			if a.isEnabled() != !forceDisabled {
				t.Fatalf("isEnabled = %v", a.isEnabled())
			}
			if a.isActive() != (!forceDisabled && active) {
				t.Fatalf("isActive = %v", a.isActive())
			}
		}
	}
}

func TestA11y_ActiveDescendantHonoredWhenContainerFocused(t *testing.T) {
	builder := a11yTestNewBuilder()
	container, item := a11y.NodeID(1), a11y.NodeID(2)
	a11yTestAssert(t, builder.push(container, a11yTestNode()))
	builder.setFocus(container)
	a11yTestAssert(t, builder.push(item, a11yTestNode()))
	a11yTestAssert(t, builder.focusIsAncestorOfCurrent())
	builder.setActiveDescendant(item)
	builder.pop()
	builder.pop()
	update := builder.finalize()
	if update.Focus != item {
		t.Fatalf("focus = %v", update.Focus)
	}
}

func TestA11y_ActiveDescendantHonoredForDeepDescendant(t *testing.T) {
	builder := a11yTestNewBuilder()
	container, group, item := a11y.NodeID(1), a11y.NodeID(2), a11y.NodeID(3)
	a11yTestAssert(t, builder.push(container, a11yTestNode()))
	builder.setFocus(container)
	a11yTestAssert(t, builder.push(group, a11yTestNode()))
	a11yTestAssert(t, builder.push(item, a11yTestNode()))
	a11yTestAssert(t, builder.focusIsAncestorOfCurrent())
	builder.setActiveDescendant(item)
	builder.pop()
	builder.pop()
	builder.pop()
	update := builder.finalize()
	if update.Focus != item {
		t.Fatalf("focus = %v", update.Focus)
	}
}

func TestA11y_ActiveDescendantIgnoredWhenFocusInOtherSubtree(t *testing.T) {
	builder := a11yTestNewBuilder()
	focusedContainer, focusedLeaf := a11y.NodeID(1), a11y.NodeID(2)
	otherContainer, otherItem := a11y.NodeID(3), a11y.NodeID(4)
	a11yTestAssert(t, builder.push(focusedContainer, a11yTestNode()))
	a11yTestAssert(t, builder.push(focusedLeaf, a11yTestNode()))
	builder.setFocus(focusedLeaf)
	builder.pop()
	builder.pop()
	a11yTestAssert(t, builder.push(otherContainer, a11yTestNode()))
	a11yTestAssert(t, builder.push(otherItem, a11yTestNode()))
	a11yTestAssert(t, !builder.focusIsAncestorOfCurrent())
	builder.pop()
	builder.pop()
	update := builder.finalize()
	if update.Focus != focusedLeaf {
		t.Fatalf("focus = %v", update.Focus)
	}
}

func TestA11y_ActiveDescendantIgnoredWhenNothingFocused(t *testing.T) {
	builder := a11yTestNewBuilder()
	container, item := a11y.NodeID(1), a11y.NodeID(2)
	a11yTestAssert(t, builder.push(container, a11yTestNode()))
	a11yTestAssert(t, builder.push(item, a11yTestNode()))
	a11yTestAssert(t, !builder.focusIsAncestorOfCurrent())
	builder.pop()
	builder.pop()
	update := builder.finalize()
	if update.Focus != rootNodeID {
		t.Fatalf("focus = %v", update.Focus)
	}
}

func TestA11y_RegularFocusUsedWhenNoActiveDescendant(t *testing.T) {
	builder := a11yTestNewBuilder()
	focused := a11y.NodeID(1)
	a11yTestAssert(t, builder.push(focused, a11yTestNode()))
	builder.setFocus(focused)
	builder.pop()
	update := builder.finalize()
	if update.Focus != focused {
		t.Fatalf("focus = %v", update.Focus)
	}
}

func TestA11y_FocusIsAncestorExcludesSelfAndNonAncestors(t *testing.T) {
	builder := a11yTestNewBuilder()
	container, item := a11y.NodeID(1), a11y.NodeID(2)
	a11yTestAssert(t, builder.push(container, a11yTestNode()))
	builder.setFocus(container)
	a11yTestAssert(t, !builder.focusIsAncestorOfCurrent())
	a11yTestAssert(t, builder.push(item, a11yTestNode()))
	a11yTestAssert(t, builder.focusIsAncestorOfCurrent())
	builder.pop()
	builder.pop()
}

func TestA11y_MultipleActiveDescendantClaimsPanicInDebug(t *testing.T) {
	a11yTestShouldPanicInDebug(t, "active descendant claimed by multiple nodes", func() {
		builder := a11yTestNewBuilder()
		builder.setActiveDescendant(1)
		builder.setActiveDescendant(2)
	})
}

func TestA11y_SettingFocusTwicePanicsInDebug(t *testing.T) {
	a11yTestShouldPanicInDebug(t, "set_focus called more than once", func() {
		builder := a11yTestNewBuilder()
		builder.setFocus(1)
		builder.setFocus(2)
	})
}

func TestA11y_SetFocusWithoutSetFocusable(t *testing.T) {
	a11yTestShouldPanicInDebug(t, "was not registered with set_focusable", func() {
		a := a11yTestNewA11y()
		node := a11y.NodeID(1)
		a11yTestAssert(t, a.nodes.push(node, a11yTestNode()))
		a.setFocus(node)
	})
}

func TestA11y_SetActiveDescendantOnFocusedNode(t *testing.T) {
	a11yTestShouldPanicInDebug(t, "on the focused node", func() {
		a := a11yTestNewA11y()
		node := a11y.NodeID(1)
		a11yTestAssert(t, a.nodes.push(node, a11yTestNode()))
		a.setFocusable(node, FocusID(0))
		a.setFocus(node)
		a.setActiveDescendant(node)
	})
}

func TestA11y_TwoSiblingsClaimingActiveDescendant(t *testing.T) {
	a11yTestShouldPanicInDebug(t, "active descendant claimed by multiple nodes", func() {
		a := a11yTestNewA11y()
		container, first, second := a11y.NodeID(1), a11y.NodeID(2), a11y.NodeID(3)
		a11yTestAssert(t, a.nodes.push(container, a11yTestNode()))
		a.setFocusable(container, FocusID(0))
		a.setFocus(container)
		a11yTestAssert(t, a.nodes.push(first, a11yTestNode()))
		a.setActiveDescendant(first)
		a.nodes.pop()
		a11yTestAssert(t, a.nodes.push(second, a11yTestNode()))
		a.setActiveDescendant(second)
		a.nodes.pop()
		a.nodes.pop()
	})
}

func TestA11y_ActiveDescendantInUnfocusedSubtreeKeepsRealFocus(t *testing.T) {
	a := a11yTestNewA11y()
	na, nb, nc := a11y.NodeID(1), a11y.NodeID(2), a11y.NodeID(3)
	a11yTestAssert(t, a.nodes.push(na, a11yTestNode()))
	a.setFocusable(na, FocusID(0))
	a.setFocus(na)
	a.nodes.pop()
	a11yTestAssert(t, a.nodes.push(nb, a11yTestNode()))
	a11yTestAssert(t, a.nodes.push(nc, a11yTestNode()))
	a.setActiveDescendant(nc)
	a.nodes.pop()
	a.nodes.pop()
	update := a.endFrame(frameDebugInfo{})
	if update.Focus != na {
		t.Fatalf("focus = %v", update.Focus)
	}
	if _, ok := a.debugTreeJSON(); !ok {
		t.Fatal("no debug tree captured")
	}
}
