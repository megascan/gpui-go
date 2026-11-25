//go:build windows

package windows

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"unicode/utf16"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

func TestPlatform_ReadCredentialWithUsername(t *testing.T) {
	username, password := roundTripCredential(t, gpui.Some("alice"), []byte("secret"))
	if username != "alice" || string(password) != "secret" {
		t.Fatalf("got (%q, %q)", username, password)
	}
}

func TestPlatform_ReadCredentialWithoutUsername(t *testing.T) {
	username, password := roundTripCredential(t, gpui.None[string](), []byte("secret"))
	if username != "" || string(password) != "secret" {
		t.Fatalf("got (%q, %q)", username, password)
	}
}

func roundTripCredential(t *testing.T, username gpui.Option[string], secret []byte) (string, []byte) {
	t.Helper()
	targetName := append(utf16.Encode([]rune(fmt.Sprintf("zed-test-%d-%s", os.Getpid(), username.UnwrapOrDefault()))), 0)
	var usernameUTF16 []uint16
	if name, ok := username.Get(); ok {
		usernameUTF16 = append(utf16.Encode([]rune(name)), 0)
	}
	secret = append([]byte(nil), secret...)
	credential := win32.CREDENTIALW{
		Type:               win32.CRED_TYPE_GENERIC,
		TargetName:         &targetName[0],
		CredentialBlobSize: uint32(len(secret)),
		CredentialBlob:     &secret[0],
		Persist:            win32.CRED_PERSIST_SESSION,
	}
	if len(usernameUTF16) > 0 {
		credential.UserName = &usernameUTF16[0]
	}
	if err := win32.CredWriteW(&credential, 0); err != nil {
		t.Fatal(err)
	}
	credentials, err := win32.CredReadW(&targetName[0], win32.CRED_TYPE_GENERIC, 0)
	if err != nil {
		t.Fatal(err)
	}
	gotUsername, gotPassword, resultErr := usernameAndPassword(credentials)
	win32.CredFree(unsafe.Pointer(credentials))
	if err := win32.CredDeleteW(&targetName[0], win32.CRED_TYPE_GENERIC, 0); err != nil {
		t.Fatal(err)
	}
	if resultErr != nil {
		t.Fatal(resultErr)
	}
	return gotUsername, gotPassword
}

func TestPlatform_EncodeRestartArguments(t *testing.T) {
	if got := encodeRestartArguments(nil); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got, want := encodeRestartArguments([]string{"--user-data-dir", `C:\Zed Data`}), `"--user-data-dir" "C:\Zed Data"`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got, want := encodeRestartArguments([]string{`C:\`}), `"C:\\"`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlatform_Clipboard(t *testing.T) {
	if os.Getenv("GPUI_TEST_CLIPBOARD") != "1" {
		t.Skip("touches the system clipboard; set GPUI_TEST_CLIPBOARD=1 to run")
	}
	check := func(want gpui.ClipboardItem) {
		t.Helper()
		got, ok := readFromClipboard()
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("read %#v (%v), want %#v", got, ok, want)
		}
	}

	item := gpui.NewClipboardItemString("你好，我是张小白")
	writeToClipboard(item)
	check(item)

	item = gpui.NewClipboardItemString("12345")
	writeToClipboard(item)
	check(item)

	item = gpui.NewClipboardItemStringWithJSONMetadata("abcdef", []int{3, 4})
	writeToClipboard(item)
	check(item)

	writeToClipboard(gpui.NewClipboardItemStringWithJSONMetadata("before\x00after", []int{12}))
	check(gpui.NewClipboardItemStringWithJSONMetadata("before after", []int{12}))

	item = gpui.NewClipboardItemStringWithJSONMetadata("a\nb\nc", []int{5})
	writeToClipboard(item)
	if raw, ok := readRawClipboardText(); !ok || raw != "a\r\nb\r\nc" {
		t.Fatalf("raw %q (%v)", raw, ok)
	}
	check(item)

	writeToClipboard(gpui.NewClipboardItemStringWithJSONMetadata("a\r\nb\nc\rd", []int{6}))
	if raw, ok := readRawClipboardText(); !ok || raw != "a\r\nb\r\nc\rd" {
		t.Fatalf("raw %q (%v)", raw, ok)
	}
	check(gpui.NewClipboardItemStringWithJSONMetadata("a\nb\nc\rd", []int{6}))
}
