//go:build linux

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

	"github.com/megascan/gpui-go"
)

const secBusConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
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

func secStartBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not found")
	}
	dir, err := os.MkdirTemp("", "gpuidbus")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(strings.ReplaceAll(secBusConfig, "%DIR%", dir)), 0o600); err != nil {
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

func secConnect(t *testing.T, address string) *godbus.Conn {
	t.Helper()
	conn, err := godbus.Connect(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestSecurity_SEC021_FileChooserResponseFromNonPortalIgnored(t *testing.T) {
	address := secStartBus(t)
	portal := secConnect(t, address)
	handles := make(chan godbus.ObjectPath, 1)
	open := func(sender godbus.Sender, parent, title string, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
		token, _ := options["handle_token"].Value().(string)
		handle := godbus.ObjectPath(fmt.Sprintf("%s/request/%s/%s", portalPath, strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_"), token))
		handles <- handle
		return handle, nil
	}
	if err := portal.ExportMethodTable(map[string]any{"OpenFile": open, "SaveFile": open}, portalPath, fileChooserInterface); err != nil {
		t.Fatal(err)
	}
	if err := portal.ExportMethodTable(map[string]any{
		"Get": func(iface, name string) (godbus.Variant, *godbus.Error) { return godbus.MakeVariant(uint32(4)), nil },
	}, portalPath, "org.freedesktop.DBus.Properties"); err != nil {
		t.Fatal(err)
	}
	if _, err := portal.RequestName(portalService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	app := secConnect(t, address)
	result := make(chan []string, 1)
	go func() {
		uris, err := fileChooserRequest(func() (*godbus.Conn, error) { return app, nil }, "SaveFile", "", "Save File", map[string]godbus.Variant{})
		if err != nil {
			result <- []string{"error: " + err.Error()}
			return
		}
		result <- uris.UnwrapOrDefault()
	}()
	var handle godbus.ObjectPath
	select {
	case handle = <-handles:
	case <-time.After(10 * time.Second):
		t.Fatal("portal was not called")
	}
	attacker := secConnect(t, address)
	forged := map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{"file:///home/user/.bashrc"})}
	if err := attacker.Emit(handle, portalRequestInterface+".Response", uint32(0), forged); err != nil {
		t.Fatal(err)
	}
	select {
	case uris := <-result:
		t.Fatalf("Response from non-portal %s accepted: %q", attacker.Names()[0], uris)
	case <-time.After(time.Second):
	}
	genuine := map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{"file:///home/user/report.txt"})}
	if err := portal.Emit(handle, portalRequestInterface+".Response", uint32(0), genuine); err != nil {
		t.Fatal(err)
	}
	select {
	case uris := <-result:
		if len(uris) != 1 || uris[0] != "file:///home/user/report.txt" {
			t.Fatalf("uris = %q", uris)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Response from the portal was not accepted")
	}
}

func TestSecurity_SEC017_OpenWithSystemNeutralisesLeadingDash(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	calls := filepath.Join(root, "calls")
	for _, dir := range []string{bin, calls} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := fmt.Sprintf("#!/bin/sh\nf=$(mktemp %q/call.XXXXXXXX)\nprintf '%%s\\n' \"$@\" >\"$f\"\nmv \"$f\" \"$f.done\"\n", calls)
	for _, name := range []string{"xdg-open", "dbus-launch"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(root, "no-bus"))
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	targets := []string{"--help", "-x", "--manual"}
	want := make(map[string]bool, len(targets))
	for _, target := range targets {
		if err := os.WriteFile(filepath.Join(cwd, target), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		want[filepath.Join(cwd, target)] = true
	}
	p := newLinuxPlatform(gpui.WindowingModesHeadless)
	defer p.Close()
	for _, target := range targets {
		p.OpenWithSystem(target)
	}
	var launched []string
	deadline := time.Now().Add(10 * time.Second)
	for len(launched) < len(targets) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		matches, err := filepath.Glob(filepath.Join(calls, "call.*.done"))
		if err != nil {
			t.Fatal(err)
		}
		launched = launched[:0]
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				t.Fatal(err)
			}
			launched = append(launched, strings.TrimSpace(string(data)))
		}
	}
	if len(launched) != len(targets) {
		t.Fatalf("launches = %q", launched)
	}
	for _, arg := range launched {
		if strings.HasPrefix(arg, "-") {
			t.Fatalf("xdg-open received option-like argument %q", arg)
		}
		if !want[arg] {
			t.Fatalf("xdg-open received %q, want one of the absolute paths %v", arg, want)
		}
		delete(want, arg)
	}
	if len(want) != 0 {
		t.Fatalf("targets never launched: %v", want)
	}
}
