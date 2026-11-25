//go:build linux

package dbus

import (
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

func TestSystemNotifications_CallerActionIDsDoNotCollideWithTransportActionIDs(t *testing.T) {
	actionIDs := map[string]gpui.SharedString{
		"gpui-action-0": "default",
		"gpui-action-1": "__closed",
	}
	check := func(transportID string, want gpui.Option[gpui.SharedString], wantOK bool) {
		t.Helper()
		got, ok := responseActionID(transportID, actionIDs)
		if ok != wantOK || got != want {
			t.Fatalf("responseActionID(%q) = %v, %v; want %v, %v", transportID, got, ok, want, wantOK)
		}
	}
	check(defaultAction, gpui.None[gpui.SharedString](), true)
	check("gpui-action-0", gpui.Some[gpui.SharedString]("default"), true)
	check("gpui-action-1", gpui.Some[gpui.SharedString]("__closed"), true)
	check("unknown", gpui.None[gpui.SharedString](), false)
}

type fakeNotifications struct {
	calls chan []string
}

func (f *fakeNotifications) Notify(appName string, replacesID uint32, icon, summary, body string, actions []string, hints map[string]godbus.Variant, timeout int32) (uint32, *godbus.Error) {
	f.calls <- append([]string{appName, summary, body}, actions...)
	return 7, nil
}

func TestSystemNotifications_ShowForwardsActionInvoked(t *testing.T) {
	address := startTestBus(t)
	server := connectTestBus(t, address)
	fake := &fakeNotifications{calls: make(chan []string, 1)}
	if err := server.Export(fake, notificationsPath, notificationsInterface); err != nil {
		t.Fatal(err)
	}
	if _, err := server.RequestName(notificationsService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	state := NewSystemNotificationState(connectTestBus(t, address))
	responses := make(chan gpui.SystemNotificationResponse, 1)
	state.OnResponse(func(f func()) { f() }, func(r gpui.SystemNotificationResponse) { responses <- r })
	state.Show(gpui.Some("app"), gpui.SystemNotification{
		Tag: "tag", Title: "title", Body: "body",
		Actions: []gpui.SystemNotificationAction{{ID: "open", Label: "Open"}},
	})
	select {
	case call := <-fake.calls:
		want := []string{"app", "title", "body", "default", "default", "gpui-action-0", "Open"}
		if len(call) != len(want) {
			t.Fatalf("Notify args = %v", call)
		}
		for i := range want {
			if call[i] != want[i] {
				t.Fatalf("Notify args = %v, want %v", call, want)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Notify was not called")
	}
	if err := server.Emit(notificationsPath, notificationsInterface+".ActionInvoked", uint32(7), "gpui-action-0"); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-responses:
		if r.Tag != "tag" || r.ActionID != gpui.Some[gpui.SharedString]("open") {
			t.Fatalf("response = %+v", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no response")
	}
}
