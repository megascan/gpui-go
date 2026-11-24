//go:build darwin && !ios

package macos

import (
	"runtime"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

func labelledTree(label string, toggled bool) a11y.TreeUpdate {
	root := a11y.NewNode(a11y.RoleWindow)
	root.SetChildren([]a11y.NodeID{1})
	button := a11y.NewNode(a11y.RoleButton)
	button.SetLabel(label)
	if toggled {
		button.SetToggled(a11y.ToggledTrue)
	} else {
		button.SetToggled(a11y.ToggledFalse)
	}
	return a11y.TreeUpdate{
		Nodes:  []a11y.NodeEntry{{ID: 0, Node: root}, {ID: 1, Node: button}},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}
}

func TestNode_TextAlignmentValues(t *testing.T) {
	center, right := nsTextAlignment(a11y.TextAlignCenter), nsTextAlignment(a11y.TextAlignRight)
	if runtime.GOARCH == "amd64" {
		if center != 2 || right != 1 {
			t.Fatalf("amd64 center/right = %d/%d", center, right)
		}
	} else if center != 1 || right != 2 {
		t.Fatalf("arm64 center/right = %d/%d", center, right)
	}
	if nsTextAlignment(a11y.TextAlignLeft) != 0 || nsTextAlignment(a11y.TextAlignJustify) != 3 {
		t.Fatal("left/justify")
	}
}

func TestNode_WrapperValueAndTitle(t *testing.T) {
	tree := consumer.NewTree(labelledTree("OK", true), false)
	root := tree.State().Root()
	if _, ok := (nodeWrapper{&root}).title(); ok {
		t.Fatal("root window must not expose a title")
	}
	button, ok := tree.State().NodeByTreeLocalID(1, a11y.TreeIDRoot())
	if !ok {
		t.Fatal("button missing")
	}
	if title, ok := (nodeWrapper{&button}).title(); !ok || title != "OK" {
		t.Fatalf("title = %q, %v", title, ok)
	}
	if v := (nodeWrapper{&button}).value(); v.kind != valueBool || !v.bool {
		t.Fatalf("value = %+v", v)
	}
}

func TestEvent_UpdateQueuesTitleAndValueChanges(t *testing.T) {
	tree := consumer.NewTree(labelledTree("OK", false), false)
	generator := newEventGenerator(nil)
	tree.UpdateAndProcessChanges(labelledTree("Cancel", true), generator)
	var notifications []string
	for _, event := range generator.events {
		if generic, ok := event.(queuedEventGeneric); ok {
			notifications = append(notifications, generic.notification)
		}
	}
	want := []string{"NSAccessibilityValueChangedNotification", "NSAccessibilityTitleChangedNotification"}
	if len(notifications) != len(want) {
		t.Fatalf("notifications = %v", notifications)
	}
	for i := range want {
		if notifications[i] != want[i] {
			t.Fatalf("notifications = %v", notifications)
		}
	}
}

func TestPatch_MethodTypeEncoding(t *testing.T) {
	if got := methodTypeEncoding("@", nil); got != "@@:" {
		t.Fatalf("encoding = %q", got)
	}
	if countArgs("accessibilityHitTest:") != 1 || countArgs("accessibilityFocusedUIElement") != 0 {
		t.Fatal("countArgs")
	}
}
