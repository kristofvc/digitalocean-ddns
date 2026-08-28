package digitalocean

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindAndUpdate(t *testing.T) {
	var updated bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing auth")
		}
		if r.Method == http.MethodGet {
			if got := r.URL.Query().Get("name"); got != "home.example.com" {
				t.Errorf("name filter=%q, want %q", got, "home.example.com")
			}
			json.NewEncoder(w).Encode(map[string]any{"domain_records": []Record{{ID: 3, Type: "A", Name: "home", Data: "192.0.2.1", TTL: 300}}})
			return
		}
		updated = true
		var b Record
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatal(err)
		}
		if b.Data != "203.0.113.8" {
			t.Errorf("data=%s", b.Data)
		}
		w.Write([]byte("{}"))
	}))
	defer s.Close()
	c := Client{HTTP: s.Client(), BaseURL: s.URL, Token: "secret"}
	r, err := c.FindARecord(context.Background(), "example.com", "home")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateRecord(context.Background(), "example.com", r, "203.0.113.8"); err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("not updated")
	}
}

func TestFindARecordQueryName(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{name: "@", want: "example.com"},
		{name: "home", want: "home.example.com"},
		{name: "nested.home", want: "nested.home.example.com"},
		{name: "home.example.com", want: "home.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("name"); got != tt.want {
					t.Errorf("name filter=%q, want %q", got, tt.want)
				}
				json.NewEncoder(w).Encode(map[string]any{"domain_records": []Record{{ID: 1, Type: "A", Name: tt.name}}})
			}))
			defer s.Close()
			c := Client{HTTP: s.Client(), BaseURL: s.URL, Token: "secret"}
			if _, err := c.FindARecord(context.Background(), "example.com", tt.name); err != nil {
				t.Fatal(err)
			}
		})
	}
}
