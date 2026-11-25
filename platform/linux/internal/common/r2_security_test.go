//go:build linux

package common

import "testing"

func TestSecurity_R2_10_MailtoKeepsOnlySafeFields(t *testing.T) {
	got := filterMailtoQuery("to=a@example.com&attach=%2Fetc%2Fshadow&Subject=hi&ATTACHMENT=x&%61ttach=y&body=b&cc=c&bcc=d")
	want := "to=a@example.com&Subject=hi&body=b&cc=c&bcc=d"
	if got != want {
		t.Fatalf("filterMailtoQuery = %q, want %q", got, want)
	}
}
