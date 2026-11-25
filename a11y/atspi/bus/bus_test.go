package bus

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

var privateBusAddress string

func TestMain(m *testing.M) {
	os.Unsetenv("AT_SPI_BUS_ADDRESS")
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/gpui-go-atspi-bus-test")
	daemon, dir, err := startPrivateDaemon()
	if err == nil {
		os.Setenv("DBUS_SESSION_BUS_ADDRESS", privateBusAddress)
	} else {
		fmt.Fprintf(os.Stderr, "bus tests: private dbus-daemon unavailable: %v\n", err)
	}
	code := m.Run()
	stopEventLoop()
	if daemon != nil {
		_ = daemon.Process.Kill()
		_ = daemon.Wait()
	}
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	os.Exit(code)
}

func startPrivateDaemon() (*exec.Cmd, string, error) {
	binary, err := exec.LookPath("dbus-daemon")
	if err != nil {
		return nil, "", err
	}
	dir, err := os.MkdirTemp("", "gpui-go-atspi-bus-")
	if err != nil {
		return nil, "", err
	}
	config := filepath.Join(dir, "bus.conf")
	configXML := `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN" "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=` + dir + `</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
`
	if err := os.WriteFile(config, []byte(configXML), 0o600); err != nil {
		return nil, dir, err
	}
	cmd := exec.Command(binary, "--config-file="+config, "--nofork", "--nopidfile", "--print-address=1")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, dir, err
	}
	if err := cmd.Start(); err != nil {
		return nil, dir, err
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, dir, err
	}
	privateBusAddress = strings.TrimSpace(line)
	return cmd, dir, nil
}

type fakeStatus struct {
	conn    *dbus.Conn
	mu      sync.Mutex
	enabled bool
}

func (s *fakeStatus) setEnabled(enabled bool) error {
	s.mu.Lock()
	s.enabled = enabled
	s.mu.Unlock()
	return s.conn.Emit(statusPath, propertiesInterface+".PropertiesChanged", statusInterface, map[string]dbus.Variant{"IsEnabled": dbus.MakeVariant(enabled)}, []string{})
}

type fakeRegistry struct {
	mu    sync.Mutex
	plugs []ownedObjectAddress
}

func startFakeServices(t *testing.T) (*fakeStatus, *fakeRegistry) {
	t.Helper()
	conn, err := dbus.Connect(privateBusAddress)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	status := &fakeStatus{conn: conn}
	registry := &fakeRegistry{}
	if err := conn.ExportMethodTable(map[string]any{
		"GetAddress": func() (string, *dbus.Error) { return privateBusAddress, nil },
	}, statusPath, "org.a11y.Bus"); err != nil {
		t.Fatal(err)
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Get": func(iface, name string) (dbus.Variant, *dbus.Error) {
			if iface != statusInterface || name != "IsEnabled" {
				return dbus.Variant{}, fdoError(errorUnknownProperty, name)
			}
			status.mu.Lock()
			defer status.mu.Unlock()
			return dbus.MakeVariant(status.enabled), nil
		},
		"GetAll": func(iface string) (map[string]dbus.Variant, *dbus.Error) {
			status.mu.Lock()
			defer status.mu.Unlock()
			return map[string]dbus.Variant{"IsEnabled": dbus.MakeVariant(status.enabled), "ScreenReaderEnabled": dbus.MakeVariant(false)}, nil
		},
	}, statusPath, propertiesInterface); err != nil {
		t.Fatal(err)
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Embed": func(plug ownedObjectAddress) (ownedObjectAddress, *dbus.Error) {
			registry.mu.Lock()
			registry.plugs = append(registry.plugs, plug)
			registry.mu.Unlock()
			return ownedObjectAddress{BusName: conn.Names()[0], Path: rootPath}, nil
		},
	}, rootPath, "org.a11y.atspi.Socket"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"org.a11y.Bus", "org.a11y.atspi.Registry"} {
		reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
		if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
			t.Fatalf("RequestName(%s) = %v, %v", name, reply, err)
		}
	}
	return status, registry
}

type testActivation struct {
	activated chan struct{}
}

func (h *testActivation) RequestInitialTree() (a11y.TreeUpdate, bool) {
	h.activated <- struct{}{}
	return busTree("Hello"), true
}

type testAction struct {
	requests chan a11y.ActionRequest
}

func (h *testAction) DoAction(request a11y.ActionRequest) {
	h.requests <- request
}

type testDeactivation struct {
	deactivated chan struct{}
}

func (h *testDeactivation) DeactivateAccessibility() {
	h.deactivated <- struct{}{}
}

func withLabel(node a11y.Node, label string) a11y.Node {
	node.SetLabel(label)
	return node
}

func busTree(label string) a11y.TreeUpdate {
	window := withChildren(a11y.RoleWindow, []a11y.NodeID{1})
	window.SetLabel(label)
	button := a11y.NewNode(a11y.RoleButton)
	button.SetLabel("Press")
	button.AddAction(a11y.ActionClick)
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: 0, Node: window},
			{ID: 1, Node: button},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(0)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  0,
	}
}

func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

func waitSignal(t *testing.T, ch <-chan *dbus.Signal, name string) *dbus.Signal {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case signal := <-ch:
			if signal.Name == name {
				return signal
			}
		case <-deadline:
			t.Fatalf("timed out waiting for signal %s", name)
			return nil
		}
	}
}

func waitSignals(t *testing.T, ch <-chan *dbus.Signal, names ...string) map[string]*dbus.Signal {
	t.Helper()
	found := map[string]*dbus.Signal{}
	deadline := time.After(10 * time.Second)
	for len(found) < len(names) {
		select {
		case signal := <-ch:
			for _, name := range names {
				if signal.Name == name && found[name] == nil {
					found[name] = signal
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for signals %v, got %v", names, found)
		}
	}
	return found
}

func callSignature(t *testing.T, object dbus.BusObject, method string, args ...any) *dbus.Call {
	t.Helper()
	call := object.Call(method, 0, args...)
	if call.Err != nil {
		t.Fatalf("%s: %v", method, call.Err)
	}
	return call
}

type signatureLog struct {
	mu      sync.Mutex
	reply   string
	signals map[string]string
}

func (l *signatureLog) intercept(msg *dbus.Message) {
	signature, _ := msg.Headers[dbus.FieldSignature].Value().(dbus.Signature)
	l.mu.Lock()
	defer l.mu.Unlock()
	switch msg.Type {
	case dbus.TypeMethodReply:
		l.reply = signature.String()
	case dbus.TypeSignal:
		iface, _ := msg.Headers[dbus.FieldInterface].Value().(string)
		member, _ := msg.Headers[dbus.FieldMember].Value().(string)
		l.signals[iface+"."+member] = signature.String()
	}
}

func (l *signatureLog) lastReply() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reply
}

func (l *signatureLog) signal(name string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.signals[name]
}

var replySignatures = &signatureLog{signals: map[string]string{}}

func wantSignature(t *testing.T, call *dbus.Call, want string) {
	t.Helper()
	if got := replySignatures.lastReply(); got != want {
		t.Fatalf("%s reply signature = %q, want %q", call.Method, got, want)
	}
}

func TestBus_AdapterOnPrivateBus(t *testing.T) {
	if privateBusAddress == "" {
		t.Skip("dbus-daemon is not available")
	}
	status, registry := startFakeServices(t)

	client, err := dbus.Connect(privateBusAddress, dbus.WithIncomingInterceptor(replySignatures.intercept))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	signals := make(chan *dbus.Signal, 64)
	client.Signal(signals)
	if err := client.AddMatchSignal(dbus.WithMatchPathNamespace("/org/a11y/atspi")); err != nil {
		t.Fatal(err)
	}

	activation := &testActivation{activated: make(chan struct{}, 4)}
	action := &testAction{requests: make(chan a11y.ActionRequest, 4)}
	deactivation := &testDeactivation{deactivated: make(chan struct{}, 4)}
	adapter := NewAdapter(activation, action, deactivation)
	defer adapter.Release()

	select {
	case <-activation.activated:
		t.Fatal("activated while the a11y bus is disabled")
	case <-time.After(100 * time.Millisecond):
	}

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
			if registry.plugs[0].Path != rootPath {
				t.Errorf("plug path = %s", registry.plugs[0].Path)
			}
		}
		registry.mu.Unlock()
		if appName == "" {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if appName == "" {
		t.Fatal("application never embedded itself in the registry")
	}

	root := client.Object(appName, rootPath)
	var children []ownedObjectAddress
	for time.Now().Before(deadline) {
		call := callSignature(t, root, accessibleInterfaceName+".GetChildren")
		wantSignature(t, call, "a(so)")
		if err := call.Store(&children); err != nil {
			t.Fatal(err)
		}
		if len(children) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(children) != 1 {
		t.Fatalf("root children = %v", children)
	}
	windowPath := children[0].Path
	if !strings.HasPrefix(string(windowPath), fmt.Sprintf("%s%d/", accessiblePathPrefix, adapter.id)) {
		t.Fatalf("window path = %s", windowPath)
	}

	window := client.Object(appName, windowPath)
	var windowChildren []ownedObjectAddress
	for time.Now().Before(deadline) {
		call := window.Call(accessibleInterfaceName+".GetChildren", 0)
		if call.Err == nil {
			if err := call.Store(&windowChildren); err != nil {
				t.Fatal(err)
			}
			if len(windowChildren) == 1 {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(windowChildren) != 1 {
		t.Fatalf("window children = %v", windowChildren)
	}
	name, err := window.GetProperty(accessibleInterfaceName + ".Name")
	if err != nil || name.Value() != "Hello" {
		t.Fatalf("window Name = %v, %v", name, err)
	}
	parent, err := root.GetProperty(accessibleInterfaceName + ".Parent")
	if err != nil || parent.Signature().String() != "(so)" {
		t.Fatalf("root Parent = %v, %v", parent, err)
	}

	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetRole"), "u")
	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetState"), "au")
	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetInterfaces"), "as")
	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetChildAtIndex", int32(0)), "(so)")
	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetRelationSet"), "a(ua(so))")
	wantSignature(t, callSignature(t, window, accessibleInterfaceName+".GetApplication"), "(so)")
	wantSignature(t, callSignature(t, window, componentInterfaceName+".GetExtents", uint32(0)), "(iiii)")
	wantSignature(t, callSignature(t, root, accessibleInterfaceName+".GetState"), "au")
	wantSignature(t, callSignature(t, client.Object(appName, cachePath()), cacheInterfaceName+".GetItems"), "a((so)(so)(so)iiassusau)")
	if call := window.Call(accessibleInterfaceName+".GetChildAtIndex", 0, int32(-1)); call.Err == nil {
		t.Fatal("negative index accepted")
	} else if dbusErr, ok := call.Err.(dbus.Error); !ok || dbusErr.Name != errorInvalidArgs {
		t.Fatalf("negative index error = %v", call.Err)
	}
	var xml string
	if err := window.Call(introspectInterface+".Introspect", 0).Store(&xml); err != nil || !strings.Contains(xml, accessibleInterfaceName) {
		t.Fatalf("Introspect = %q, %v", xml, err)
	}

	button := client.Object(appName, windowChildren[0].Path)
	var done bool
	if err := button.Call(actionInterfaceName+".DoAction", 0, int32(0)).Store(&done); err != nil || !done {
		t.Fatalf("DoAction = %v, %v", done, err)
	}
	request := receive(t, action.requests, "action request")
	if request.Action != a11y.ActionClick {
		t.Fatalf("action = %v", request.Action)
	}

	if err := root.SetProperty(applicationInterfaceName+".Id", dbus.MakeVariant(int32(7))); err != nil {
		t.Fatal(err)
	}
	id, err := root.GetProperty(applicationInterfaceName + ".Id")
	if err != nil || id.Value() != int32(7) {
		t.Fatalf("Id = %v, %v", id, err)
	}
	if err := root.SetProperty(applicationInterfaceName+".ToolkitName", dbus.MakeVariant("x")); err == nil {
		t.Fatal("read-only property accepted a value")
	}

	adapter.UpdateIfActive(func() a11y.TreeUpdate { return busTree("Renamed") })
	signal := waitSignal(t, signals, "org.a11y.atspi.Event.Object.PropertyChange")
	if got := replySignatures.signal(signal.Name); got != "siiva{sv}" {
		t.Fatalf("PropertyChange signature = %q", got)
	}
	if signal.Body[0] != "accessible-name" || signal.Body[3].(dbus.Variant).Value() != "Renamed" {
		t.Fatalf("PropertyChange body = %v", signal.Body)
	}

	adapter.UpdateWindowFocusState(true)
	signal = waitSignal(t, signals, "org.a11y.atspi.Event.Window.Activate")
	if signal.Path != windowPath || signal.Body[3].(dbus.Variant).Value() != "Renamed" {
		t.Fatalf("Activate = %v %v", signal.Path, signal.Body)
	}

	adapter.UpdateIfActive(func() a11y.TreeUpdate {
		return a11y.TreeUpdate{
			Nodes: []a11y.NodeEntry{
				{ID: 0, Node: withLabel(withChildren(a11y.RoleWindow, []a11y.NodeID{1, 2}), "Renamed")},
				{ID: 2, Node: a11y.NewNode(a11y.RoleButton)},
			},
			TreeID: a11y.TreeIDRoot(),
			Focus:  0,
		}
	})
	found := waitSignals(t, signals, "org.a11y.atspi.Cache.AddAccessible", "org.a11y.atspi.Event.Object.ChildrenChanged")
	if got := replySignatures.signal("org.a11y.atspi.Cache.AddAccessible"); got != "(so)(so)(so)iiassusau" {
		t.Fatalf("AddAccessible signature = %q", got)
	}
	signal = found["org.a11y.atspi.Event.Object.ChildrenChanged"]
	if signal.Body[0] != "add" || signal.Body[1] != int32(1) {
		t.Fatalf("ChildrenChanged add body = %v", signal.Body)
	}
	adapter.UpdateIfActive(func() a11y.TreeUpdate {
		return a11y.TreeUpdate{
			Nodes:  []a11y.NodeEntry{{ID: 0, Node: withLabel(withChildren(a11y.RoleWindow, []a11y.NodeID{1}), "Renamed")}},
			TreeID: a11y.TreeIDRoot(),
			Focus:  0,
		}
	})
	found = waitSignals(t, signals, "org.a11y.atspi.Cache.RemoveAccessible", "org.a11y.atspi.Event.Object.ChildrenChanged")
	if got := replySignatures.signal("org.a11y.atspi.Cache.RemoveAccessible"); got != "so" {
		t.Fatalf("RemoveAccessible signature = %q", got)
	}
	signal = found["org.a11y.atspi.Event.Object.ChildrenChanged"]
	if signal.Body[0] != "remove" || signal.Body[1] != int32(-1) {
		t.Fatalf("ChildrenChanged remove body = %v", signal.Body)
	}

	if err := status.setEnabled(false); err != nil {
		t.Fatal(err)
	}
	receive(t, deactivation.deactivated, "deactivation")
	adapter.state.mu.Lock()
	kind := adapter.state.kind
	adapter.state.mu.Unlock()
	if kind != adapterStateInactive {
		t.Fatalf("state after deactivation = %v", kind)
	}
}
