//go:build securityharness && linux

package common

import (
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
)

func TestR2_S04_01_AllowlistCaseAndWhitespace(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	inputs := []string{"HTTP://127.0.0.1/a", "hTtPs://127.0.0.1/b", " http://127.0.0.1/c", "\thttp://127.0.0.1/d", "http://127.0.0.1/e\t", "FILE:///etc/passwd", "File://localhost/etc/passwd", "mailto:-x", "http:-x", "https:javascript:alert(1)", "http:/etc/passwd"}
	for _, uri := range inputs {
		OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	time.Sleep(time.Second)
	for _, call := range stubs.calls(t) {
		arg := call.args[len(call.args)-1]
		t.Logf("launched %s %q", call.program, call.args)
		lower := strings.ToLower(arg)
		if !(strings.HasPrefix(lower, "http:") || strings.HasPrefix(lower, "https:") || strings.HasPrefix(lower, "mailto:")) || strings.HasPrefix(arg, "-") || strings.ContainsAny(arg, " \t\n") {
			t.Errorf("non-allowlisted or option-like target reached %s: %q", call.program, call.args)
		}
	}
}

func TestR2_S04_02_MailtoAttachmentParams(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	hostile := []string{
		"mailto:victim@example.com?attach=/home/user/.ssh/id_rsa",
		"mailto:victim@example.com?attachment=file:///home/user/.gnupg/secring.gpg",
		"mailto:?to=a@example.com&attach=%2Fetc%2Fshadow&body=see%20attached",
	}
	for _, uri := range hostile {
		OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	calls := stubs.waitCalls(t, len(hostile), 10*time.Second)
	var reached []string
	for _, call := range calls {
		arg := call.args[len(call.args)-1]
		t.Logf("launched %s %q", call.program, call.args)
		_, query, _ := strings.Cut(arg, "?")
		if query == "" {
			continue
		}
		for field := range strings.SplitSeq(query, "&") {
			key, _, _ := strings.Cut(field, "=")
			name, err := url.QueryUnescape(key)
			if err != nil {
				reached = append(reached, arg)
				break
			}
			switch strings.ToLower(name) {
			case "to", "cc", "bcc", "subject", "body", "in-reply-to":
			default:
				reached = append(reached, arg)
			}
		}
	}
	if len(reached) > 0 {
		t.Errorf("mailto: passes the allowlist with attach/attachment or other non-allowlisted parameters intact, handed to the desktop mail handler: %q", reached)
	}
}
