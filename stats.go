package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type counts struct {
	Requests     int64   `json:"requests"`
	Attempts     int64   `json:"attempts"`
	Compressed   int64   `json:"compressed"`
	Unchanged    int64   `json:"unchanged"`
	Skipped      int64   `json:"skipped"`
	Failures     int64   `json:"failures"`
	TokensBefore int64   `json:"tokens_before"`
	TokensAfter  int64   `json:"tokens_after"`
	TokenSamples int64   `json:"token_samples"`
	BytesBefore  int64   `json:"bytes_before"`
	BytesAfter   int64   `json:"bytes_after"`
	LatencyMS    float64 `json:"latency_ms"`
}
type metricEvent struct {
	At           time.Time `json:"at"`
	Model        string    `json:"model"`
	Status       string    `json:"status"`
	Reason       string    `json:"reason,omitempty"`
	Attempt      bool      `json:"-"`
	TokensBefore int64     `json:"tokens_before"`
	TokensAfter  int64     `json:"tokens_after"`
	TokenKnown   bool      `json:"token_known"`
	BytesBefore  int64     `json:"bytes_before"`
	BytesAfter   int64     `json:"bytes_after"`
	LatencyMS    float64   `json:"latency_ms"`
}
type statsState struct {
	Schema      int                `json:"schema"`
	Started     time.Time          `json:"started_at"`
	Updated     time.Time          `json:"updated_at"`
	Totals      counts             `json:"totals"`
	Models      map[string]*counts `json:"models"`
	Hours       map[string]*counts `json:"hours"`
	Recent      []metricEvent      `json:"recent"`
	LastAttempt *metricEvent       `json:"last_attempt,omitempty"`
}
type statsStore struct {
	mu        sync.Mutex
	state     statsState
	path      string
	dirty     bool
	saveError string
	lastSave  time.Time
	stop      chan struct{}
	done      chan struct{}
}

func newStatsStore(path string) (*statsStore, error) {
	s := &statsStore{path: path, stop: make(chan struct{}), done: make(chan struct{}), state: statsState{Schema: 1, Started: time.Now().UTC(), Models: map[string]*counts{}, Hours: map[string]*counts{}, Recent: []metricEvent{}}}
	if path != "" {
		raw, err := os.ReadFile(path)
		if err == nil {
			if len(raw) > 16<<20 {
				return nil, errors.New("stats file exceeds size limit")
			}
			if json.Unmarshal(raw, &s.state) != nil || s.state.Schema != 1 || s.state.Models == nil || s.state.Hours == nil {
				return nil, errors.New("invalid stats file; restore or choose a new stats_path")
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.New("cannot read stats file")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, errors.New("cannot create stats directory")
		}
		s.dirty = true
		if err := s.flush(); err != nil {
			return nil, errors.New("cannot write stats file")
		}
	}
	go func() {
		defer close(s.done)
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				s.flush()
			case <-s.stop:
				s.flush()
				return
			}
		}
	}()
	return s, nil
}
func (s *statsStore) close() { close(s.stop); <-s.done }
func (c *counts) add(e metricEvent) {
	c.Requests++
	if e.Attempt {
		c.Attempts++
		c.LatencyMS += e.LatencyMS
	}
	switch e.Status {
	case "compressed":
		c.Compressed++
		c.BytesBefore += e.BytesBefore
		c.BytesAfter += e.BytesAfter
		if e.TokenKnown {
			c.TokenSamples++
			c.TokensBefore += e.TokensBefore
			c.TokensAfter += e.TokensAfter
		}
	case "unchanged":
		c.Unchanged++
	case "fallback":
		c.Failures++
	default:
		c.Skipped++
	}
}
func (s *statsStore) record(e metricEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if len(e.Model) > 160 {
		e.Model = e.Model[:160]
	}
	if e.Model == "" {
		e.Model = "unknown"
	}
	if _, ok := s.state.Models[e.Model]; !ok && len(s.state.Models) >= 128 {
		e.Model = "other"
	}
	if e.Attempt || e.Status == "fallback" {
		copy := e
		s.state.LastAttempt = &copy
	}
	s.state.Updated = e.At
	s.state.Totals.add(e)
	if s.state.Models[e.Model] == nil {
		s.state.Models[e.Model] = &counts{}
	}
	s.state.Models[e.Model].add(e)
	hour := e.At.UTC().Truncate(time.Hour).Format(time.RFC3339)
	if s.state.Hours[hour] == nil {
		s.state.Hours[hour] = &counts{}
	}
	s.state.Hours[hour].add(e)
	cutoff := e.At.UTC().Add(-90 * 24 * time.Hour).Format(time.RFC3339)
	for k := range s.state.Hours {
		if k < cutoff {
			delete(s.state.Hours, k)
		}
	}
	s.state.Recent = append(s.state.Recent, e)
	if len(s.state.Recent) > 100 {
		s.state.Recent = s.state.Recent[len(s.state.Recent)-100:]
	}
	s.dirty = true
}
func (s *statsStore) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty || s.path == "" {
		return nil
	}
	raw, err := json.Marshal(s.state)
	if err == nil {
		var f *os.File
		f, err = os.CreateTemp(filepath.Dir(s.path), ".headroom-*.tmp")
		if err == nil {
			tmp := f.Name()
			defer os.Remove(tmp)
			_, err = f.Write(raw)
			if err == nil {
				err = f.Sync()
			}
			ce := f.Close()
			if err == nil {
				err = ce
			}
			if err == nil {
				err = os.Rename(tmp, s.path)
			}
		}
	}
	if err != nil {
		s.saveError = "Statistics could not be saved to disk"
		return err
	}
	s.saveError = ""
	s.dirty = false
	s.lastSave = time.Now().UTC()
	return nil
}
func (s *statsStore) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, _ := json.Marshal(s.state)
	var state statsState
	json.Unmarshal(raw, &state)
	keys := make([]string, 0, len(state.Hours))
	for k := range state.Hours {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	series := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		series = append(series, map[string]any{"at": k, "counts": state.Hours[k]})
	}
	return map[string]any{"version": version, "started_at": state.Started, "updated_at": state.Updated, "totals": state.Totals, "models": state.Models, "history": series, "recent": state.Recent, "last_attempt": state.LastAttempt, "persistence": map[string]any{"enabled": s.path != "", "error": s.saveError, "last_saved_at": s.lastSave}, "scope": "CPA Headroom plugin only"}
}

var statsConfigMu sync.Mutex
var activeStats *statsStore

func configureStats(path string) (*statsStore, error) {
	statsConfigMu.Lock()
	defer statsConfigMu.Unlock()
	if activeStats != nil && activeStats.path == path {
		return activeStats, nil
	}
	next, err := newStatsStore(path)
	if err != nil {
		return nil, err
	}
	if activeStats != nil {
		activeStats.close()
	}
	activeStats = next
	return next, nil
}
func flushStats() {
	statsConfigMu.Lock()
	defer statsConfigMu.Unlock()
	if activeStats != nil {
		activeStats.flush()
	}
}
