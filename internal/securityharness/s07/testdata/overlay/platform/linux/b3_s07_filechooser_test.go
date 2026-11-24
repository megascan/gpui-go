//go:build securityharness && linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
)

const b3BusConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=%DIR%</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
`

func b3StartBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Fatal("harness: dbus-daemon not found")
	}
	dir, err := os.MkdirTemp("", "b3bus")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(strings.ReplaceAll(b3BusConfig, "%DIR%", dir)), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(daemon, "--config-file="+config, "--nofork", "--nopidfile", "--print-address=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(address)
}

func b3Connect(t *testing.T, address string) *godbus.Conn {
	t.Helper()
	conn, err := godbus.Connect(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

type b3Portal struct {
	conn    *godbus.Conn
	called  chan godbus.ObjectPath
	respond bool
}

func b3StartPortal(t *testing.T, address string, portal *b3Portal) {
	t.Helper()
	conn := b3Connect(t, address)
	portal.conn = conn
	open := func(sender godbus.Sender, parent, title string, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
		token, _ := options["handle_token"].Value().(string)
		handle := godbus.ObjectPath(fmt.Sprintf("%s/request/%s/%s", portalPath, strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_"), token))
		portal.called <- handle
		if portal.respond {
			go func() {
				time.Sleep(1500 * time.Millisecond)
				_ = conn.Emit(handle, portalRequestInterface+".Response", uint32(0), map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{"file:///home/user/report.txt"})})
			}()
		}
		return handle, nil
	}
	if err := conn.ExportMethodTable(map[string]any{"OpenFile": open, "SaveFile": open}, portalPath, fileChooserInterface); err != nil {
		t.Fatal(err)
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Get": func(iface, name string) (godbus.Variant, *godbus.Error) { return godbus.MakeVariant(uint32(4)), nil },
	}, portalPath, "org.freedesktop.DBus.Properties"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.RequestName(portalService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
}

func b3Request(address string, conn *godbus.Conn, method string) chan []string {
	result := make(chan []string, 1)
	go func() {
		uris, err := fileChooserRequest(func() (*godbus.Conn, error) { return conn, nil }, method, "", "Save File", map[string]godbus.Variant{})
		if err != nil {
			result <- []string{"error: " + err.Error()}
			return
		}
		result <- uris.UnwrapOrDefault()
	}()
	return result
}

func TestSecB3_S07_FileChooserResponseForgedByMonitoringPeer(t *testing.T) {
	address := b3StartBus(t)
	portal := &b3Portal{called: make(chan godbus.ObjectPath, 2), respond: true}
	b3StartPortal(t, address, portal)
	monitorConn := b3Connect(t, address)
	seen := make(chan *godbus.Message, 64)
	if err := monitorConn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, []string{"type='method_call',interface='" + fileChooserInterface + "'"}, uint32(0)).Err; err != nil {
		t.Fatalf("harness: BecomeMonitor: %v", err)
	}
	monitorConn.Eavesdrop(seen)
	attacker := b3Connect(t, address)
	app := b3Connect(t, address)
	result := b3Request(address, app, "SaveFile")
	var forgedPath godbus.ObjectPath
	deadline := time.After(10 * time.Second)
	for forgedPath == "" {
		select {
		case msg := <-seen:
			if msg.Type != godbus.TypeMethodCall || len(msg.Body) < 3 {
				continue
			}
			sender, _ := msg.Headers[godbus.FieldSender].Value().(string)
			options, _ := msg.Body[2].(map[string]godbus.Variant)
			token, _ := options["handle_token"].Value().(string)
			forgedPath = godbus.ObjectPath(fmt.Sprintf("%s/request/%s/%s", portalPath, strings.ReplaceAll(strings.TrimPrefix(sender, ":"), ".", "_"), token))
		case <-deadline:
			t.Fatal("harness: monitor saw no FileChooser call")
		}
	}
	<-portal.called
	evil := "file:///home/user/.bashrc"
	if err := attacker.Emit(forgedPath, portalRequestInterface+".Response", uint32(0), map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{evil})}); err != nil {
		t.Fatal(err)
	}
	select {
	case uris := <-result:
		if len(uris) == 1 && uris[0] == evil {
			path, _ := fileURIToPath(uris[0])
			t.Fatalf("FAIL: SaveFile result forged by non-portal peer %s (portal owner %s) after reading the handle_token via BecomeMonitor; app receives save path %q", attacker.Names()[0], portal.conn.Names()[0], path)
		}
		t.Logf("result %v", uris)
	case <-time.After(10 * time.Second):
		t.Fatal("harness: no result")
	}
}

func TestSecB3_S07_FileChooserResponseBlindSprayRejected(t *testing.T) {
	address := b3StartBus(t)
	portal := &b3Portal{called: make(chan godbus.ObjectPath, 2), respond: true}
	b3StartPortal(t, address, portal)
	attacker := b3Connect(t, address)
	app := b3Connect(t, address)
	result := b3Request(address, app, "OpenFile")
	<-portal.called
	sender := strings.ReplaceAll(strings.TrimPrefix(app.Names()[0], ":"), ".", "_")
	for n := range 256 {
		path := godbus.ObjectPath(fmt.Sprintf("%s/request/%s/gpui%016x", portalPath, sender, n))
		_ = attacker.Emit(path, portalRequestInterface+".Response", uint32(0), map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{"file:///etc/shadow"})})
	}
	select {
	case uris := <-result:
		if len(uris) == 1 && uris[0] == "file:///etc/shadow" {
			t.Fatalf("FAIL: blind forged Response accepted")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("harness: no result")
	}
}

func TestSecB3_S07_FileChooserNoResponseTimeout(t *testing.T) {
	address := b3StartBus(t)
	portal := &b3Portal{called: make(chan godbus.ObjectPath, 2)}
	b3StartPortal(t, address, portal)
	app := b3Connect(t, address)
	result := b3Request(address, app, "OpenFile")
	<-portal.called
	select {
	case uris := <-result:
		t.Logf("returned %v", uris)
	case <-time.After(5 * time.Second):
		t.Logf("INFO: fileChooserRequest still blocked 5s after a portal that never emits Response (no deadline; the background worker is held until the bus connection closes)")
	}
}
