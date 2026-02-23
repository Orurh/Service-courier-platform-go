package pprofserver

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthOrLocalOnly_AllowsLoopbackWithoutAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := authOrLocalOnly(next, Config{User: "", Pass: ""})
	req := httptest.NewRequest(http.MethodGet, "http://example/debug/pprof/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected %d, got %d", http.StatusTeapot, rr.Code)
	}
}

func TestAuthOrLocalOnly_NonLoopback_EmptyCreds_Unauthorized(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	})

	h := authOrLocalOnly(next, Config{User: "", Pass: ""})
	req := httptest.NewRequest(http.MethodGet, "http://example/debug/pprof/", nil)
	req.RemoteAddr = "8.8.8.8:54444"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("expected WWW-Authenticate header to be set")
	}
}

func TestAuthOrLocalOnly_NonLoopback_WrongCreds_Unauthorized(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	})

	h := authOrLocalOnly(next, Config{User: "u", Pass: "p"})
	req := httptest.NewRequest(http.MethodGet, "http://example/debug/pprof/", nil)
	req.RemoteAddr = "8.8.8.8:54444"
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("u:WRONG")))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("expected WWW-Authenticate header to be set")
	}
}

func TestAuthOrLocalOnly_NonLoopback_CorrectCreds_Allows(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	h := authOrLocalOnly(next, Config{User: "u", Pass: "p"})
	req := httptest.NewRequest(http.MethodGet, "http://example/debug/pprof/", nil)
	req.RemoteAddr = "8.8.8.8:54444"
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("u:p")))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected %d, got %d", http.StatusTeapot, rr.Code)
	}
}

func TestIsLoopback(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"127.0.0.1:123", true},
		{"127.0.0.1", true},
		{" 127.0.0.1 ", true},
		{"[::1]:123", true},
		{"8.8.8.8:1", false},
		{"not-an-ip:1", false},
	}
	for _, tc := range cases {
		got := isLoopback(tc.in)
		if got != tc.want {
			t.Fatalf("isLoopback(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSecureEq(t *testing.T) {
	if secureEq("a", "ab") {
		t.Fatal("expected false for different lengths")
	}
	if !secureEq("abc", "abc") {
		t.Fatal("expected true for equal strings")
	}
	if secureEq("abc", "abd") {
		t.Fatal("expected false for different strings")
	}
}
