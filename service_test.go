package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestServiceEndpointsCachedAndFiltered(t *testing.T) {
	var hits atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/readyz" {
			w.WriteHeader(503)
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "healthy", "version": "0.37.0", "config": map[string]any{"secret": "must-not-escape"}, "checks": map[string]any{"upstream": map[string]any{"status": "healthy", "url": "https://private.example/?token=secret"}}, "tokens": map[string]any{"saved": 12}, "series": map[string]any{"daily": []any{map[string]any{"timestamp": "2026-09-16T00:00:00Z", "tokens_saved": 12, "by_model": map[string]any{"private-label": 1}}}}})
	}))
	defer s.Close()
	setup(t, s.URL)
	a := fetchService(settings.Load())
	fetchService(settings.Load())
	if hits.Load() != 5 {
		t.Fatalf("expected one read of each endpoint, got %d", hits.Load())
	}
	raw, _ := json.Marshal(a)
	for _, secret := range []string{"must-not-escape", "private.example", "private-label"} {
		if contains(string(raw), secret) {
			t.Fatal("sensitive metadata leaked")
		}
	}
	entries := a["endpoints"].([]map[string]any)
	if entries[1]["ok"] != false {
		t.Fatal("readiness failure hidden")
	}
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
