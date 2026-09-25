package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStatsPersistenceAndConcurrency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	s, err := newStatsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.record(metricEvent{Model: "model-a", Status: "compressed", Attempt: true, TokensBefore: 100, TokensAfter: 30, TokenKnown: true, BytesBefore: 400, BytesAfter: 100, LatencyMS: 12})
		}()
	}
	wg.Wait()
	s.close()
	restored, err := newStatsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.close()
	c := restored.state.Totals
	if c.Requests != 50 || c.Compressed != 50 || c.TokensBefore != 5000 || c.TokensAfter != 1500 || c.BytesAfter != 5000 {
		t.Fatalf("lost counts: %+v", c)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe file mode %v", info.Mode())
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("content")) || bytes.Contains(raw, []byte("Authorization")) {
		t.Fatal("unexpected content persisted")
	}
}
func TestStatsOutcomeAccounting(t *testing.T) {
	s, e := newStatsStore("")
	if e != nil {
		t.Fatal(e)
	}
	defer s.close()
	for _, e := range []metricEvent{{Status: "skipped"}, {Status: "fallback", Attempt: true}, {Status: "unchanged", Attempt: true}, {Status: "compressed", Attempt: true, TokensBefore: 100, TokensAfter: 1, TokenKnown: false, BytesBefore: 1000, BytesAfter: 20}} {
		s.record(e)
	}
	c := s.state.Totals
	if c.Requests != 4 || c.Attempts != 3 || c.Failures != 1 || c.Unchanged != 1 || c.Skipped != 1 || c.Compressed != 1 || c.TokensBefore != 0 {
		t.Fatalf("incorrect outcomes: %+v", c)
	}
}
func TestStatsRetentionAndCorruption(t *testing.T) {
	s, _ := newStatsStore("")
	defer s.close()
	s.record(metricEvent{At: time.Now().Add(-91 * 24 * time.Hour), Status: "skipped"})
	for i := 0; i < 110; i++ {
		s.record(metricEvent{Status: "skipped"})
	}
	if len(s.state.Recent) != 100 || len(s.state.Hours) != 1 || s.state.Totals.Requests != 111 {
		t.Fatal("retention damaged lifetime totals")
	}
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte("invalid"), 0600)
	if _, err := newStatsStore(p); err == nil {
		t.Fatal("corrupt stats must not be overwritten")
	}
	raw, _ := os.ReadFile(p)
	if string(raw) != "invalid" {
		t.Fatal("corrupt data overwritten")
	}
}
func TestManagementRoutesAndStaticPage(t *testing.T) {
	raw, e := managementRegistration()
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(raw, []byte("Headroom Stats")) || !bytes.Contains(raw, []byte("/plugins/headroom/stats")) || !bytes.Contains(raw, []byte("/stats-data")) || !bytes.Contains(raw, []byte("/service-data")) {
		t.Fatal("missing routes")
	}
	for _, test := range []struct {
		path     string
		code     int
		contains string
	}{{"/v0/resource/plugins/headroom/stats", 200, "<!doctype html>"}, {"/v0/resource/plugins/headroom/secret", 404, "not found"}} {
		r, _ := json.Marshal(map[string]string{"Method": "GET", "Path": test.path})
		raw, e := managementHandle(r)
		if e != nil {
			t.Fatal(e)
		}
		var response struct {
			Result struct {
				StatusCode int
				Body       []byte
			}
		}
		json.Unmarshal(raw, &response)
		if response.Result.StatusCode != test.code || !bytes.Contains(response.Result.Body, []byte(test.contains)) {
			t.Fatalf("invalid route: %s", raw)
		}
	}
}
