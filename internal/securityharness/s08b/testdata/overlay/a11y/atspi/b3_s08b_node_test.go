//go:build securityharness

package atspi

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

func b3Run(text string, lengths []uint8) a11y.Node {
	run := a11y.NewNode(a11y.RoleTextRun)
	run.SetBounds(a11y.Rect{X0: 0, Y0: 60, X1: 90, Y1: 80})
	run.SetValue(text)
	run.SetCharacterLengths(lengths)
	positions := make([]float32, len(lengths))
	widths := make([]float32, len(lengths))
	for i := range lengths {
		positions[i] = float32(10 * i)
		widths[i] = 10
	}
	run.SetCharacterPositions(positions)
	run.SetCharacterWidths(widths)
	run.SetWordStarts([]uint8{0, 3})
	return run
}

func b3Update() a11y.TreeUpdate {
	window := a11y.NewNode(a11y.RoleWindow)
	window.SetLabel("W‮indow")
	window.SetChildren([]a11y.NodeID{1, 2, 5, 6, 7})
	button := a11y.NewNode(a11y.RoleButton)
	button.SetLabel("OK")
	button.SetBounds(a11y.Rect{X0: 10, Y0: 20, X1: 110, Y1: 50})
	button.AddAction(a11y.ActionClick)
	input := a11y.NewNode(a11y.RoleMultilineTextInput)
	input.SetBounds(a11y.Rect{X0: 0, Y0: 60, X1: 100, Y1: 100})
	input.SetChildren([]a11y.NodeID{3, 4})
	input.SetTextDirection(a11y.TextDirectionLeftToRight)
	input.SetTextSelection(a11y.TextSelection{
		Anchor: a11y.TextPosition{Node: 3, CharacterIndex: 1},
		Focus:  a11y.TextPosition{Node: 4, CharacterIndex: 2},
	})
	slider := a11y.NewNode(a11y.RoleSlider)
	slider.SetNumericValue(5)
	slider.SetMinNumericValue(0)
	slider.SetMaxNumericValue(10)
	slider.SetNumericValueStep(1)
	slider.AddAction(a11y.ActionSetValue)
	link := a11y.NewNode(a11y.RoleLink)
	link.SetURL("https://example.invalid/")
	link.SetLabel("link")
	link.AddAction(a11y.ActionClick)
	list := a11y.NewNode(a11y.RoleListBox)
	list.SetChildren([]a11y.NodeID{8})
	option := a11y.NewNode(a11y.RoleListBoxOption)
	option.SetLabel("opt")
	tree := a11y.NewTree(0)
	tree.ToolkitName = gpuiutil.Some("gpui")
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: window},
			{ID: 1, Node: button},
			{ID: 2, Node: input},
			{ID: 3, Node: b3Run("hé\U0001F600 日\n", []uint8{1, 2, 4, 1, 3, 1})},
			{ID: 4, Node: b3Run("אבc", []uint8{2, 2, 1})},
			{ID: 5, Node: slider},
			{ID: 6, Node: link},
			{ID: 7, Node: list},
			{ID: 8, Node: option},
		},
		Tree:   gpuiutil.Some(tree),
		TreeID: a11y.TreeIDRoot(),
		Focus:  2,
	}
}

func b3Values(t reflect.Type) []reflect.Value {
	var raw []any
	switch t.Kind() {
	case reflect.Int32:
		raw = []any{int32(-1), int32(0), int32(1), int32(2), int32(5), int32(9), int32(100), int32(math.MaxInt32), int32(math.MinInt32)}
	case reflect.Int:
		raw = []any{-1, 0, 1, 2, 5, 9, 1 << 31, math.MaxInt, math.MinInt}
	case reflect.Uint32:
		raw = []any{uint32(0), uint32(1), uint32(2), uint32(3), uint32(4), uint32(5), uint32(6), uint32(7), uint32(100), uint32(math.MaxUint32)}
	case reflect.String:
		raw = []any{"", "font-family", "\x00", "invalid"}
	case reflect.Bool:
		raw = []any{false, true}
	case reflect.Float64:
		raw = []any{math.NaN(), math.Inf(1), math.Inf(-1), -1e308, 0.0, 5.5, 1e308}
	default:
		return nil
	}
	out := make([]reflect.Value, len(raw))
	for i, v := range raw {
		out[i] = reflect.ValueOf(v).Convert(t)
	}
	return out
}

func b3Product(sets [][]reflect.Value, emit func([]reflect.Value)) {
	args := make([]reflect.Value, len(sets))
	var rec func(int)
	rec = func(i int) {
		if i == len(sets) {
			emit(append([]reflect.Value(nil), args...))
			return
		}
		for _, v := range sets[i] {
			args[i] = v
			rec(i + 1)
		}
	}
	rec(0)
}

func b3Fmt(args []reflect.Value) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%v", a.Interface())
	}
	return strings.Join(parts, ",")
}

func TestSecB3_S08b_PlatformNodeHostileArguments(t *testing.T) {
	appContext := NewAppContext(gpuiutil.Some("app"))
	defer runtime.KeepAlive(appContext)
	handler := &recordingHandler{}
	bounds := NewWindowBounds(a11y.Rect{X0: 95, Y0: 75, X1: 505, Y1: 405}, a11y.Rect{X0: 100, Y0: 100, X1: 500, Y1: 400})
	context := newContext(appContext, consumer.NewTree(b3Update(), true), handler, bounds)
	appContext.pushAdapter(7, context)
	panics := map[string]string{}
	var lockLeaks []string
	calls := 0
	for local := a11y.NodeID(0); local <= 8; local++ {
		node := reflect.ValueOf(newPlatformNode(context, 7, nodeID(context, local)))
		typ := node.Type()
		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)
			ft := method.Type
			sets := make([][]reflect.Value, 0, ft.NumIn()-1)
			ok := true
			for i := 1; i < ft.NumIn(); i++ {
				vals := b3Values(ft.In(i))
				if vals == nil {
					ok = false
					break
				}
				sets = append(sets, vals)
			}
			if !ok {
				continue
			}
			b3Product(sets, func(args []reflect.Value) {
				calls++
				func() {
					defer func() {
						if r := recover(); r != nil {
							key := fmt.Sprintf("node %d %s(%s)", local, method.Name, b3Fmt(args))
							msg := fmt.Sprint(r)
							if _, seen := panics[method.Name+"|"+msg]; !seen {
								panics[method.Name+"|"+msg] = key
							}
							if context.treeLock.TryLock() {
								context.treeLock.Unlock()
							} else {
								lockLeaks = append(lockLeaks, key+": "+msg)
								context.treeLock = sync.RWMutex{}
							}
						}
					}()
					node.MethodByName(method.Name).Call(args)
				}()
			})
		}
	}
	done := make(chan struct{})
	go func() {
		_, release := context.writeTree()
		release()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		lockLeaks = append(lockLeaks, "tree write lock unobtainable after the run")
	}
	t.Logf("%d calls, %d action requests", calls, len(handler.requests))
	var lines []string
	for k, v := range panics {
		lines = append(lines, v+" -> "+strings.SplitN(k, "|", 2)[1])
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Logf("PANIC %s", l)
	}
	if len(lockLeaks) > 0 {
		t.Errorf("FAIL: panic left the tree lock held (UI-thread update would deadlock): %v", lockLeaks)
	}
	if len(panics) > 0 {
		t.Fatalf("FAIL: %d distinct panics reachable from AT-SPI client arguments", len(panics))
	}
}
