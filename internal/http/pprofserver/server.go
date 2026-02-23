package pprofserver

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
)

// Config stores pprof server settings.
type Config struct {
	User string
	Pass string
}

// Handler returns pprof handlers.
func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	return authOrLocalOnly(mux, cfg)
}

func authOrLocalOnly(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isLoopback(r.RemoteAddr) {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.User == "" || cfg.Pass == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok || !secureEq(u, cfg.User) || !secureEq(p, cfg.Pass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureEq(u, s string) bool {
	if len(u) != len(s) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(u), []byte(s)) == 1
}

func isLoopback(remoteAddr string) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	host = strings.TrimSpace(host)

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
