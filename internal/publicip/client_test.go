package publicip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFallback(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 500) }))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("203.0.113.7\n")) }))
	defer good.Close()
	ip, err := (Client{HTTP: good.Client(), Providers: []string{bad.URL, good.URL}}).Get(context.Background())
	if err != nil || ip.String() != "203.0.113.7" {
		t.Fatalf("got %v, %v", ip, err)
	}
}
func TestRejectsIPv6(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("2001:db8::1")) }))
	defer s.Close()
	if _, err := (Client{HTTP: s.Client(), Providers: []string{s.URL}}).Get(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
