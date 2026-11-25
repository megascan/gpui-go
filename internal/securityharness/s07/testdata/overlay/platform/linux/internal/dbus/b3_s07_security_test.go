//go:build securityharness && linux

package dbus

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

type b3Notifications struct {
	mu    sync.Mutex
	next  uint32
	calls chan uint32
}

func (f *b3Notifications) Notify(appName string, replacesID uint32, icon, summary, body string, actions []string, hints map[string]godbus.Variant, timeout int32) (uint32, *godbus.Error) {
	f.mu.Lock()
	f.next++
	id := f.next
	f.mu.Unlock()
	f.calls <- id
	return id, nil
}

func TestSecB3_S07_NotificationActionInvokedBlindSpoof(t *testing.T) {
	address := startTestBus(t)
	server := connectTestBus(t, address)
	fake := &b3Notifications{next: 22, calls: make(chan uint32, 4)}
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
		Tag: "transfer-42", Title: "Confirm transfer", Body: "Approve sending funds?",
		Actions: []gpui.SystemNotificationAction{{ID: "approve", Label: "Approve"}, {ID: "reject", Label: "Reject"}},
	})
	select {
	case <-fake.calls:
	case <-time.After(10 * time.Second):
		t.Fatal("harness: Notify was not called")
	}
	attacker := connectTestBus(t, address)
	for id := uint32(1); id <= 64; id++ {
		if err := attacker.Emit(notificationsPath, notificationsInterface+".ActionInvoked", id, "gpui-action-0"); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case r := <-responses:
		t.Fatalf("FAIL: forged ActionInvoked from non-owner %s (blind id spray 1..64, no eavesdrop) delivered response %+v to the app; notification daemon is %s", attacker.Names()[0], r, server.Names()[0])
	case <-time.After(2 * time.Second):
	}
}

type b3SecretValue struct {
	Session     godbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

type b3SecretService struct {
	t            *testing.T
	conn         *godbus.Conn
	allowCrypto  bool
	promptPath   godbus.ObjectPath
	secret       b3SecretValue
	created      chan b3SecretValue
	createdCount int
	mu           sync.Mutex
}

const (
	b3Collection = godbus.ObjectPath("/org/freedesktop/secrets/collection/login")
	b3Item       = godbus.ObjectPath("/org/freedesktop/secrets/collection/login/1")
	b3Session    = godbus.ObjectPath("/org/freedesktop/secrets/session/s1")
)

func startB3SecretService(t *testing.T, address string, svc *b3SecretService) {
	t.Helper()
	conn := connectTestBus(t, address)
	svc.conn = conn
	svc.t = t
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(conn.ExportMethodTable(map[string]any{
		"OpenSession": func(algorithm string, input godbus.Variant) (godbus.Variant, godbus.ObjectPath, *godbus.Error) {
			if algorithm == algorithmEncrypted {
				if !svc.allowCrypto {
					return godbus.Variant{}, "", godbus.NewError("org.freedesktop.DBus.Error.NotSupported", []any{"no"})
				}
				return godbus.MakeVariant([]byte{1}), b3Session, nil
			}
			return godbus.MakeVariant(""), b3Session, nil
		},
		"ReadAlias": func(name string) (godbus.ObjectPath, *godbus.Error) { return b3Collection, nil },
		"Unlock": func(paths []godbus.ObjectPath) ([]godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
			if svc.promptPath != "" {
				return nil, svc.promptPath, nil
			}
			return paths, noPrompt, nil
		},
	}, secretsPath, secretServiceInterface))
	must(conn.ExportMethodTable(map[string]any{
		"CreateItem": func(props map[string]godbus.Variant, secret b3SecretValue, replace bool) (godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
			svc.mu.Lock()
			svc.createdCount++
			svc.mu.Unlock()
			svc.created <- secret
			return b3Item, noPrompt, nil
		},
		"SearchItems": func(attrs map[string]string) ([]godbus.ObjectPath, *godbus.Error) {
			return []godbus.ObjectPath{b3Item}, nil
		},
	}, b3Collection, secretCollectionIface))
	must(conn.ExportMethodTable(map[string]any{
		"Get": func(iface, name string) (godbus.Variant, *godbus.Error) {
			switch name {
			case "Label":
				return godbus.MakeVariant(KeyringLabel), nil
			case "Attributes":
				return godbus.MakeVariant(map[string]string{"url": "https://x", "username": "u"}), nil
			}
			return godbus.Variant{}, godbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{name})
		},
	}, b3Item, "org.freedesktop.DBus.Properties"))
	must(conn.ExportMethodTable(map[string]any{
		"GetSecret": func(session godbus.ObjectPath) (b3SecretValue, *godbus.Error) { return svc.secret, nil },
	}, b3Item, secretItemIface))
	must(conn.ExportMethodTable(map[string]any{
		"Close": func() *godbus.Error { return nil },
	}, b3Session, "org.freedesktop.Secret.Session"))
	if svc.promptPath != "" {
		must(conn.ExportMethodTable(map[string]any{
			"Prompt": func(windowID string) *godbus.Error { return nil },
		}, svc.promptPath, secretPromptIface))
	}
	if _, err := conn.RequestName(secretsService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
}

func b3Monitor(t *testing.T, address string, rules []string) chan *godbus.Message {
	t.Helper()
	conn := connectTestBus(t, address)
	ch := make(chan *godbus.Message, 256)
	if err := conn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, uint32(0)).Err; err != nil {
		t.Fatalf("harness: BecomeMonitor: %v", err)
	}
	conn.Eavesdrop(ch)
	return ch
}

func TestSecB3_S07_SecretServicePlainFallbackLeaksPassword(t *testing.T) {
	address := startTestBus(t)
	svc := &b3SecretService{created: make(chan b3SecretValue, 1)}
	startB3SecretService(t, address, svc)
	monitor := b3Monitor(t, address, []string{"type='method_call',member='CreateItem'"})
	app := connectTestBus(t, address)
	password := []byte("hunter2-plaintext-canary")
	if err := WriteCredentials(app, "https://x", "u", password); err != nil {
		t.Logf("WriteCredentials refused without an encrypted session: %v", err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg := <-monitor:
			if len(msg.Body) >= 2 {
				if fields, ok := msg.Body[1].([]any); ok && len(fields) >= 3 {
					if value, ok := fields[2].([]byte); ok && bytes.Equal(value, password) {
						t.Fatalf("FAIL: secret service refused %s; client fell back to %q and a same-uid bus monitor read the password in cleartext: %q", algorithmEncrypted, algorithmPlain, value)
					}
				}
			}
		case <-deadline:
			return
		}
	}
}

func b3Key() []byte {
	common := big.NewInt(1).Bytes()
	ikm := make([]byte, 128-len(common), 128)
	ikm = append(ikm, common...)
	key, _ := hkdf.Key(sha256.New, ikm, nil, "", 16)
	return key
}

func b3Encrypt(plain, iv []byte) []byte {
	block, _ := aes.NewCipher(b3Key())
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, plain)
	return out
}

func TestSecB3_S07_SecretServiceMalformedEncryptedReplies(t *testing.T) {
	iv := bytes.Repeat([]byte{7}, 16)
	cases := map[string]b3SecretValue{
		"short-iv":        {Session: b3Session, Parameters: []byte{1, 2, 3}, Value: make([]byte, 16)},
		"empty-iv":        {Session: b3Session, Parameters: nil, Value: make([]byte, 16)},
		"empty-value":     {Session: b3Session, Parameters: iv, Value: nil},
		"unaligned-value": {Session: b3Session, Parameters: iv, Value: make([]byte, 17)},
		"pad-zero":        {Session: b3Session, Parameters: iv, Value: b3Encrypt(append(bytes.Repeat([]byte{'a'}, 15), 0), iv)},
		"pad-17":          {Session: b3Session, Parameters: iv, Value: b3Encrypt(append(bytes.Repeat([]byte{'a'}, 15), 17), iv)},
		"pad-255":         {Session: b3Session, Parameters: iv, Value: b3Encrypt(append(bytes.Repeat([]byte{'a'}, 15), 255), iv)},
		"pad-mismatch":    {Session: b3Session, Parameters: iv, Value: b3Encrypt(append(bytes.Repeat([]byte{'a'}, 14), 3, 2), iv)},
		"pad-full-block":  {Session: b3Session, Parameters: iv, Value: b3Encrypt(bytes.Repeat([]byte{16}, 16), iv)},
		"huge-iv":         {Session: b3Session, Parameters: make([]byte, 1<<20), Value: make([]byte, 16)},
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			address := startTestBus(t)
			svc := &b3SecretService{allowCrypto: true, secret: value, created: make(chan b3SecretValue, 1)}
			startB3SecretService(t, address, svc)
			app := connectTestBus(t, address)
			var panicked any
			var user string
			var secret []byte
			var ok bool
			var err error
			func() {
				defer func() { panicked = recover() }()
				user, secret, ok, err = ReadCredentials(app, "https://x")
			}()
			if panicked != nil {
				t.Fatalf("FAIL: ReadCredentials panicked on %s: %v", name, panicked)
			}
			if name == "pad-full-block" {
				if err != nil || !ok || len(secret) != 0 {
					t.Fatalf("pad-full-block: got %q %q %v %v, want empty secret", user, secret, ok, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("FAIL: malformed %s accepted: user=%q secret=%q ok=%v", name, user, secret, ok)
			}
			t.Logf("%s -> %v", name, err)
		})
	}
}

func TestSecB3_S07_SecretPromptCompletedForgedByPeer(t *testing.T) {
	address := startTestBus(t)
	prompt := godbus.ObjectPath("/org/freedesktop/secrets/prompt/u1")
	svc := &b3SecretService{allowCrypto: true, promptPath: prompt, created: make(chan b3SecretValue, 1)}
	startB3SecretService(t, address, svc)
	app := connectTestBus(t, address)
	done := make(chan error, 1)
	go func() { done <- WriteCredentials(app, "https://x", "u", []byte("pw")) }()
	attacker := connectTestBus(t, address)
	stop := time.After(3 * time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			svc.mu.Lock()
			created := svc.createdCount
			svc.mu.Unlock()
			if err == nil && created > 0 {
				t.Fatalf("FAIL: Prompt.Completed forged by non-owner %s (service is %s) at predictable path %s ended the unlock prompt; WriteCredentials proceeded to CreateItem without the user answering the prompt", attacker.Names()[0], svc.conn.Names()[0], prompt)
			}
			if errors.Is(err, ErrPromptDismissed) {
				t.Fatalf("FAIL: forged Completed(dismissed) accepted")
			}
			t.Logf("WriteCredentials returned %v created=%d", err, created)
			return
		case <-tick.C:
			for n := 1; n <= 4; n++ {
				path := godbus.ObjectPath(fmt.Sprintf("/org/freedesktop/secrets/prompt/u%d", n))
				_ = attacker.Emit(path, secretPromptIface+".Completed", false, godbus.MakeVariant(""))
			}
		case <-stop:
			return
		}
	}
}
