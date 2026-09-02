package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func New(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	original := p.Director
	p.Director = func(r *http.Request) {
		original(r)
		r.Host = u.Host
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	}
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, `{"error":{"code":"upstream","message":"service unavailable"}}`, http.StatusBadGateway)
	}
	return p
}

func StripAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		next.ServeHTTP(w, r)
	})
}
