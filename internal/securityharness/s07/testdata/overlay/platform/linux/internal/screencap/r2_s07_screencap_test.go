//go:build securityharness && linux

package screencap

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

const r2BusConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
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

func r2StartBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("harness-error: dbus-daemon not found")
	}
	dir, err := os.MkdirTemp("", "r2b")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(strings.ReplaceAll(r2BusConfig, "%DIR%", dir)), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(daemon, "--config-file="+config, "--nofork", "--print-address=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(line)
}

func r2Connect(t *testing.T, address string) *godbus.Conn {
	t.Helper()
	conn, err := godbus.Connect(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

type r2SilentPortal struct{ called chan string }

func (p *r2SilentPortal) CreateSession(sender godbus.Sender, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
	token, _ := options["handle_token"].Value().(string)
	s := strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_")
	p.called <- token
	return godbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", s, token)), nil
}

func TestR2_S07_ScreencapResponseForgedByMonitoringPeer(t *testing.T) {
	address := r2StartBus(t)
	portal := r2Connect(t, address)
	fake := &r2SilentPortal{called: make(chan string, 1)}
	if err := portal.Export(fake, portalDesktopPath, portalScreenCast); err != nil {
		t.Fatal(err)
	}
	if _, err := portal.RequestName(portalDesktopService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	victim := r2Connect(t, address)
	attacker := r2Connect(t, address)
	p := NewScreenCastPortal(victim)
	victimPath := strings.ReplaceAll(strings.TrimPrefix(victim.Names()[0], ":"), ".", "_")
	type result struct {
		handle godbus.ObjectPath
		err    error
	}
	done := make(chan result, 1)
	go func() {
		h, err := p.createSession()
		done <- result{h, err}
	}()
	var token string
	select {
	case token = <-fake.called:
	case <-time.After(5 * time.Second):
		t.Fatal("harness-error: CreateSession not called")
	}
	msg := &godbus.Message{Type: godbus.TypeSignal, Headers: map[godbus.HeaderField]godbus.Variant{
		godbus.FieldPath:        godbus.MakeVariant(godbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", victimPath, token))),
		godbus.FieldInterface:   godbus.MakeVariant(portalRequest),
		godbus.FieldMember:      godbus.MakeVariant("Response"),
		godbus.FieldDestination: godbus.MakeVariant(victim.Names()[0]),
		godbus.FieldSignature:   godbus.MakeVariant(godbus.SignatureOf(uint32(0), map[string]godbus.Variant{})),
	}, Body: []any{uint32(0), map[string]godbus.Variant{"session_handle": godbus.MakeVariant("/attacker/chosen/session")}}}
	if call := attacker.Send(msg, nil); call.Err != nil {
		t.Fatalf("harness-error: send: %v", call.Err)
	}
	select {
	case r := <-done:
		t.Logf("token %q (one of 65536); one forged unicast Response from %s (portal owner is %s): createSession returned handle=%q err=%v", token, attacker.Names()[0], portal.Names()[0], r.handle, r.err)
		if r.err == nil && r.handle == "/attacker/chosen/session" {
			t.Errorf("FORGED: ScreenCast Request.Response from a non-portal peer accepted; reqWaiter.wait checks only the path, and the match rule's sender filter does not stop unicast signals")
		}
	case <-time.After(15 * time.Second):
		t.Logf("no forged response accepted")
	}
}

func TestR2_S07_ScreencapJunkSignalsExhaustWait(t *testing.T) {
	address := r2StartBus(t)
	portal := r2Connect(t, address)
	fake := &r2SilentPortal{called: make(chan string, 1)}
	if err := portal.Export(fake, portalDesktopPath, portalScreenCast); err != nil {
		t.Fatal(err)
	}
	if _, err := portal.RequestName(portalDesktopService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	victim := r2Connect(t, address)
	attacker := r2Connect(t, address)
	p := NewScreenCastPortal(victim)
	done := make(chan error, 1)
	go func() {
		_, err := p.createSession()
		done <- err
	}()
	<-fake.called
	for i := range 120 {
		msg := &godbus.Message{Type: godbus.TypeSignal, Headers: map[godbus.HeaderField]godbus.Variant{
			godbus.FieldPath:        godbus.MakeVariant(godbus.ObjectPath(fmt.Sprintf("/junk/%d", i))),
			godbus.FieldInterface:   godbus.MakeVariant("org.example.Junk"),
			godbus.FieldMember:      godbus.MakeVariant("Ping"),
			godbus.FieldDestination: godbus.MakeVariant(victim.Names()[0]),
		}}
		_ = attacker.Send(msg, nil)
	}
	time.Sleep(200 * time.Millisecond)
	start := time.Now()
	path := godbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", strings.ReplaceAll(strings.TrimPrefix(victim.Names()[0], ":"), ".", "_"), p.token))
	_ = portal.Emit(path, portalRequest+".Response", uint32(0), map[string]godbus.Variant{"session_handle": godbus.MakeVariant("/real/session")})
	select {
	case err := <-done:
		t.Logf("after 120 unrelated unicast signals, the genuine portal Response gave err=%v (%s)", err, time.Since(start).Round(time.Millisecond))
		if err != nil {
			t.Errorf("DOS: unrelated signals consume reqWaiter.wait's 100-iteration budget, so the genuine portal Response is ignored: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Errorf("createSession hung")
	}
}
