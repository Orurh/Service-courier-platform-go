package ratelimit

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP_FallbackToRemoteAddr(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "http://example/", nil)
	r.RemoteAddr = "not-a-hostport"

	if got := clientIP(r); got != "not-a-hostport" {
		t.Fatalf("expected remote addr fallback, got %q", got)
	}
}

func TestClientIP_Unknown(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "http://example/", nil)
	r.RemoteAddr = ""

	if got := clientIP(r); got != "unknown" {
		t.Fatalf("expected unknown, got %q", got)
	}
}
