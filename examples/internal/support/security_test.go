package support

import (
	"testing"
)

type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestSecurity_SEC013_support_client_body_capped(t *testing.T) {
	body, err := readBoundedBody(endlessReader{})
	if err == nil {
		t.Fatalf("endless body read to completion: %d bytes", len(body))
	}
	if len(body) != 0 {
		t.Fatalf("bounded read returned %d bytes with an error", len(body))
	}
}
