//go:build linux

package dbus

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

func TestSecurity_SEC020_NotificationActionFromNonOwnerIgnored(t *testing.T) {
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
	responses := make(chan gpui.SystemNotificationResponse, 4)
	state.OnResponse(func(f func()) { f() }, func(r gpui.SystemNotificationResponse) { responses <- r })
	state.Show(gpui.Some("app"), gpui.SystemNotification{
		Tag:     "tag",
		Title:   "title",
		Actions: []gpui.SystemNotificationAction{{ID: "approve", Label: "Approve"}},
	})
	select {
	case <-fake.calls:
	case <-time.After(10 * time.Second):
		t.Fatal("Notify was not called")
	}
	attacker := connectTestBus(t, address)
	if err := attacker.Emit(notificationsPath, notificationsInterface+".ActionInvoked", uint32(7), "gpui-action-0"); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-responses:
		t.Fatalf("ActionInvoked from non-owner %s delivered %+v", attacker.Names()[0], r)
	case <-time.After(time.Second):
	}
	if err := server.Emit(notificationsPath, notificationsInterface+".ActionInvoked", uint32(7), "gpui-action-0"); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-responses:
		if r.ActionID != gpui.Some[gpui.SharedString]("approve") {
			t.Fatalf("response = %+v", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ActionInvoked from the notification server was not delivered")
	}
}

type secPromptService struct {
	*fakeSecretService
	prompt godbus.ObjectPath
}

func (s secPromptService) Unlock(objects []godbus.ObjectPath) ([]godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
	return nil, s.prompt, nil
}

type secPrompt struct {
	called chan struct{}
}

func (p secPrompt) Prompt(windowID string) *godbus.Error {
	p.called <- struct{}{}
	return nil
}

func TestSecurity_SEC022_PromptCompletedFromNonOwnerIgnored(t *testing.T) {
	address := startTestBus(t)
	server := connectTestBus(t, address)
	prompt := godbus.ObjectPath("/org/freedesktop/secrets/prompt/u1")
	fake := &fakeSecretService{t: t, conn: server, sessions: map[godbus.ObjectPath]*keyring{}}
	if err := server.Export(secPromptService{fake, prompt}, secretsPath, secretServiceInterface); err != nil {
		t.Fatal(err)
	}
	promptObject := secPrompt{called: make(chan struct{}, 1)}
	if err := server.Export(promptObject, prompt, secretPromptIface); err != nil {
		t.Fatal(err)
	}
	if _, err := server.RequestName(secretsService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	app := connectTestBus(t, address)
	done := make(chan error, 1)
	go func() { done <- WriteCredentials(app, "https://example.invalid", "user", []byte("pw")) }()
	select {
	case <-promptObject.called:
	case err := <-done:
		t.Fatalf("WriteCredentials returned before prompting: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("Prompt was not called")
	}
	attacker := connectTestBus(t, address)
	stop := time.After(time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for forging := true; forging; {
		select {
		case err := <-done:
			t.Fatalf("Completed forged by non-owner %s ended the prompt: %v", attacker.Names()[0], err)
		case <-tick.C:
			_ = attacker.Emit(prompt, secretPromptIface+".Completed", false, godbus.MakeVariant(""))
		case <-stop:
			forging = false
		}
	}
	if err := server.Emit(prompt, secretPromptIface+".Completed", false, godbus.MakeVariant("")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Completed from the secret service was not accepted")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.items) != 1 {
		t.Fatalf("items = %+v", fake.items)
	}
}

type secCountingSecretService struct {
	*fakeSecretService
	plainSessions *atomic.Int64
}

func (s secCountingSecretService) OpenSession(algorithm string, input godbus.Variant) (godbus.Variant, godbus.ObjectPath, *godbus.Error) {
	if algorithm == algorithmPlain {
		s.plainSessions.Add(1)
	}
	return s.fakeSecretService.OpenSession(algorithm, input)
}

func TestSecurity_SEC023_PlainSecretSessionRefused(t *testing.T) {
	address := startTestBus(t)
	server := connectTestBus(t, address)
	fake := &fakeSecretService{t: t, conn: server, plainOnly: true, sessions: map[godbus.ObjectPath]*keyring{}}
	var plainSessions atomic.Int64
	if err := server.Export(secCountingSecretService{fake, &plainSessions}, secretsPath, secretServiceInterface); err != nil {
		t.Fatal(err)
	}
	if _, err := server.RequestName(secretsService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	app := connectTestBus(t, address)
	if err := WriteCredentials(app, "https://example.invalid", "user", []byte("pw")); !errors.Is(err, ErrPlainSessionRefused) {
		t.Fatalf("WriteCredentials = %v, want ErrPlainSessionRefused", err)
	}
	if _, _, _, err := ReadCredentials(app, "https://example.invalid"); !errors.Is(err, ErrPlainSessionRefused) {
		t.Fatalf("ReadCredentials = %v, want ErrPlainSessionRefused", err)
	}
	if n := plainSessions.Load(); n != 0 {
		t.Fatalf("client opened %d plain sessions", n)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.items) != 0 {
		t.Fatalf("items = %+v", fake.items)
	}
}
