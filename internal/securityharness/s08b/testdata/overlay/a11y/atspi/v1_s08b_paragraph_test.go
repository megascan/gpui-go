//go:build securityharness

package atspi

import (
	"runtime"
	"testing"
	"time"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

func v1ParagraphUpdate() a11y.TreeUpdate {
	window := a11y.NewNode(a11y.RoleWindow)
	window.SetChildren([]a11y.NodeID{1})
	input := a11y.NewNode(a11y.RoleMultilineTextInput)
	input.SetBounds(a11y.Rect{X0: 0, Y0: 0, X1: 100, Y1: 40})
	input.SetChildren([]a11y.NodeID{2, 3, 4})
	input.SetTextDirection(a11y.TextDirectionLeftToRight)
	first := b3Run("ab\n", []uint8{1, 1, 1})
	empty := a11y.NewNode(a11y.RoleTextRun)
	empty.SetBounds(a11y.Rect{X0: 0, Y0: 20, X1: 0, Y1: 40})
	empty.SetValue("")
	empty.SetCharacterLengths([]uint8{})
	empty.SetCharacterPositions([]float32{})
	empty.SetCharacterWidths([]float32{})
	empty.SetNextOnLine(4)
	second := b3Run("hello", []uint8{1, 1, 1, 1, 1})
	second.SetPreviousOnLine(3)
	tree := a11y.NewTree(0)
	tree.ToolkitName = gpuiutil.Some("gpui")
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: window},
			{ID: 1, Node: input},
			{ID: 2, Node: first},
			{ID: 3, Node: empty},
			{ID: 4, Node: second},
		},
		Tree:   gpuiutil.Some(tree),
		Focus:  1,
		TreeID: a11y.TreeIDRoot(),
	}
}

func TestSecV1_S08b_ParagraphGranularityLivelock(t *testing.T) {
	appContext := NewAppContext(gpuiutil.Some("app"))
	defer runtime.KeepAlive(appContext)
	handler := &recordingHandler{}
	bounds := NewWindowBounds(a11y.Rect{X0: 0, Y0: 0, X1: 200, Y1: 200}, a11y.Rect{X0: 0, Y0: 0, X1: 200, Y1: 200})
	context := newContext(appContext, consumer.NewTree(v1ParagraphUpdate(), true), handler, bounds)
	appContext.pushAdapter(7, context)
	node := newPlatformNode(context, 7, nodeID(context, 1))
	done := make(chan string, 1)
	go func() {
		s, start, end, err := node.StringAtOffset(5, GranularityParagraph)
		done <- s + "|" + string(rune('0'+start)) + string(rune('0'+end)) + "|" + func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}()
	}()
	select {
	case r := <-done:
		t.Logf("returned %q", r)
	case <-time.After(3 * time.Second):
		locked := !context.treeLock.TryLock()
		t.Fatalf("StringAtOffset(5, Paragraph) still running after 3s (livelock); tree write lock blocked=%v", locked)
	}
}
