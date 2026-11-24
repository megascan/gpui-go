//go:build securityharness

package bus

import (
	"encoding/xml"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type b3Actions struct{ n atomic.Int64 }

func (h *b3Actions) DoAction(request a11y.ActionRequest) { h.n.Add(1) }

func b3Tree(variant int) a11y.TreeUpdate {
	window := a11y.NewNode(a11y.RoleWindow)
	window.SetLabel("W")
	children := []a11y.NodeID{1, 2}
	if variant%2 == 1 {
		children = []a11y.NodeID{1, 2, 5}
	}
	window.SetChildren(children)
	button := a11y.NewNode(a11y.RoleButton)
	button.SetLabel("Press")
	button.AddAction(a11y.ActionClick)
	input := a11y.NewNode(a11y.RoleTextInput)
	input.SetChildren([]a11y.NodeID{3})
	input.SetTextSelection(a11y.TextSelection{Anchor: a11y.TextPosition{Node: 3, CharacterIndex: 0}, Focus: a11y.TextPosition{Node: 3, CharacterIndex: 2}})
	run := a11y.NewNode(a11y.RoleTextRun)
	text := "hé\U0001F600x"
	lengths := []uint8{1, 2, 4, 1}
	if variant%3 == 2 {
		text = "ab"
		lengths = []uint8{1, 1}
	}
	run.SetValue(text)
	run.SetCharacterLengths(lengths)
	nodes := []a11y.NodeEntry{{ID: 0, Node: window}, {ID: 1, Node: button}, {ID: 2, Node: input}, {ID: 3, Node: run}}
	if variant%2 == 1 {
		slider := a11y.NewNode(a11y.RoleSlider)
		slider.SetNumericValue(float64(variant))
		slider.SetMinNumericValue(0)
		slider.SetMaxNumericValue(10)
		nodes = append(nodes, a11y.NodeEntry{ID: 5, Node: slider})
	}
	return a11y.TreeUpdate{Nodes: nodes, Tree: gpuiutil.Some(a11y.NewTree(0)), TreeID: a11y.TreeIDRoot(), Focus: 2}
}

func b3Args(sig string) [][]any {
	pool := map[byte][]any{
		'i': {int32(-1), int32(0), int32(2), int32(math.MaxInt32), int32(math.MinInt32)},
		'u': {uint32(0), uint32(2), uint32(7), uint32(math.MaxUint32)},
		's': {"", "font-family"},
		'b': {false, true},
		'd': {math.NaN(), math.Inf(1), 5.0},
		'v': {dbus.MakeVariant(math.NaN()), dbus.MakeVariant("x")},
	}
	out := [][]any{{}}
	for i := 0; i < len(sig); i++ {
		vals, ok := pool[sig[i]]
		if !ok {
			return nil
		}
		var next [][]any
		for _, prefix := range out {
			for _, v := range vals {
				next = append(next, append(append([]any(nil), prefix...), v))
			}
		}
		out = next
	}
	return out
}

func TestSecB3_S08a_EveryMethodHostileArgsDuringUpdates(t *testing.T) {
	if privateBusAddress == "" {
		t.Fatal("harness-error: private dbus-daemon unavailable")
	}
	status, registry := startFakeServices(t)
	client, err := dbus.Connect(privateBusAddress)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	activation := &testActivation{activated: make(chan struct{}, 4)}
	actions := &b3Actions{}
	adapter := NewAdapter(activation, actions, &testDeactivation{deactivated: make(chan struct{}, 4)})
	defer adapter.Release()
	if err := status.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	receive(t, activation.activated, "activation")
	var appName string
	deadline := time.Now().Add(10 * time.Second)
	for appName == "" && time.Now().Before(deadline) {
		registry.mu.Lock()
		if len(registry.plugs) > 0 {
			appName = registry.plugs[0].BusName
		}
		registry.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	if appName == "" {
		t.Fatal("harness: no embed")
	}
	adapter.UpdateIfActive(func() a11y.TreeUpdate { return b3Tree(1) })
	time.Sleep(200 * time.Millisecond)
	var items []any
	call := client.Object(appName, cachePath()).Call(cacheInterfaceName+".GetItems", 0)
	if call.Err != nil {
		t.Fatal(call.Err)
	}
	items = call.Body
	paths := map[dbus.ObjectPath]bool{rootPath: true}
	if list, ok := items[0].([][]any); ok {
		for _, item := range list {
			if ref, ok := item[0].([]any); ok && len(ref) == 2 {
				if p, ok := ref[1].(dbus.ObjectPath); ok {
					paths[p] = true
				}
			}
		}
	}
	t.Logf("objects: %d", len(paths))
	type target struct {
		path   dbus.ObjectPath
		method string
		sig    string
	}
	var targets []target
	for p := range paths {
		var raw string
		if err := client.Object(appName, p).Call(introspectInterface+".Introspect", 0).Store(&raw); err != nil {
			continue
		}
		var node introspect.Node
		if err := xml.Unmarshal([]byte(raw), &node); err != nil {
			t.Fatalf("harness: introspect xml: %v", err)
		}
		for _, iface := range node.Interfaces {
			if !strings.HasPrefix(iface.Name, "org.a11y.atspi.") {
				continue
			}
			for _, m := range iface.Methods {
				var sig strings.Builder
				for _, a := range m.Args {
					if a.Direction == "in" {
						sig.WriteString(a.Type)
					}
				}
				targets = append(targets, target{p, iface.Name + "." + m.Name, sig.String()})
			}
		}
	}
	t.Logf("methods across objects: %d", len(targets))
	stop := make(chan struct{})
	var updates sync.WaitGroup
	updates.Add(1)
	go func() {
		defer updates.Done()
		for v := 0; ; v++ {
			select {
			case <-stop:
				return
			default:
			}
			adapter.UpdateIfActive(func() a11y.TreeUpdate { return b3Tree(v) })
			time.Sleep(time.Millisecond)
		}
	}()
	var mu sync.Mutex
	panics := map[string]string{}
	calls := 0
	var workers sync.WaitGroup
	for w := 0; w < 4; w++ {
		workers.Add(1)
		go func(w int) {
			defer workers.Done()
			for n, tg := range targets {
				if n%4 != w {
					continue
				}
				for _, args := range b3Args(tg.sig) {
					c := client.Object(appName, tg.path).Call(tg.method, 0, args...)
					mu.Lock()
					calls++
					if c.Err != nil {
						if e, ok := c.Err.(dbus.Error); ok && e.Name == errorFailed && len(e.Body) > 0 {
							if msg, _ := e.Body[0].(string); strings.Contains(msg, "runtime error") || strings.Contains(msg, "panic") {
								panics[tg.method+" "+msg] = tg.sig
							}
						}
					}
					mu.Unlock()
				}
			}
		}(w)
	}
	workers.Wait()
	close(stop)
	updates.Wait()
	alive := make(chan error, 1)
	go func() {
		var children []ownedObjectAddress
		alive <- client.Object(appName, rootPath).Call(accessibleInterfaceName+".GetChildren", 0).Store(&children)
	}()
	select {
	case err := <-alive:
		if err != nil {
			t.Fatalf("FAIL: adapter returned %v after hostile calls", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("FAIL: adapter unresponsive after hostile calls")
	}
	t.Logf("%d calls, %d action requests delivered to the app", calls, actions.n.Load())
	for k := range panics {
		t.Logf("RECOVERED PANIC %s", k)
	}
	if len(panics) > 0 {
		t.Fatalf("FAIL: %d recovered panics reachable from bus clients", len(panics))
	}
}
