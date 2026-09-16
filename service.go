package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type serviceCache struct {
	mu   sync.Mutex
	at   time.Time
	data map[string]any
}

func pick(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}
func servicePayload(path string, m map[string]any) map[string]any {
	switch path {
	case "/livez", "/readyz", "/health":
		out := pick(m, "service", "status", "alive", "ready", "version", "timestamp", "uptime_seconds", "rust_core")
		if checks, ok := m["checks"].(map[string]any); ok {
			safe := map[string]any{}
			for k, v := range checks {
				if x, ok := v.(map[string]any); ok {
					safe[k] = pick(x, "enabled", "ready", "status", "optional")
				}
			}
			out["checks"] = safe
		}
		return out
	case "/stats":
		return pick(m, "requests", "tokens", "latency", "overhead", "ttfb", "compressions_by_strategy", "tokens_saved_by_strategy", "display_session")
	case "/stats-history":
		out := pick(m, "generated_at", "lifetime", "display_session", "history_summary", "retention")
		if series, ok := m["series"].(map[string]any); ok {
			safe := map[string]any{}
			for _, key := range []string{"hourly", "daily", "weekly", "monthly"} {
				if rows, ok := series[key].([]any); ok {
					limit := 720
					if key == "daily" {
						limit = 90
					}
					if len(rows) > limit {
						rows = rows[len(rows)-limit:]
					}
					clean := []any{}
					for _, row := range rows {
						if r, ok := row.(map[string]any); ok {
							clean = append(clean, pick(r, "timestamp", "tokens_saved", "total_tokens_saved", "total_input_tokens_delta", "compression_savings_usd_delta"))
						}
					}
					safe[key] = clean
				}
			}
			out["series"] = safe
		}
		return out
	}
	return map[string]any{}
}
func fetchService(cfg *runtimeConfig) map[string]any {
	cfg.service.mu.Lock()
	defer cfg.service.mu.Unlock()
	if cfg.service.data != nil && time.Since(cfg.service.at) < 10*time.Second {
		return cfg.service.data
	}
	paths := []string{"/livez", "/readyz", "/health", "/stats", "/stats-history"}
	results := make([]map[string]any, len(paths))
	var wg sync.WaitGroup
	base, _ := url.Parse(cfg.Endpoint)
	base.Path = ""
	base.RawQuery = ""
	base.Fragment = ""
	if cfg.ServiceURL != "" {
		base, _ = url.Parse(cfg.ServiceURL)
	}
	for i, path := range paths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			entry := map[string]any{"path": path, "ok": false}
			results[i] = entry
			u := *base
			u.Path = base.Path + path
			u.RawQuery = ""
			u.Fragment = ""
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			r, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			if err != nil {
				entry["error"] = "Invalid service URL"
				return
			}
			r.Header.Set("User-Agent", "cliproxyapi-headroom/"+version)
			if cfg.TokenEnv != "" {
				token := os.Getenv(cfg.TokenEnv)
				if token == "" {
					entry["error"] = "Headroom token is unavailable"
					return
				}
				r.Header.Set("Authorization", "Bearer "+token)
			}
			response, err := cfg.client.Do(r)
			if err != nil {
				entry["error"] = "Endpoint unreachable or timed out"
				return
			}
			defer response.Body.Close()
			entry["http_status"] = response.StatusCode
			raw, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
			if err != nil || len(raw) > 8<<20 {
				entry["error"] = "Invalid or oversized response"
				return
			}
			var payload map[string]any
			if json.Unmarshal(raw, &payload) != nil {
				entry["error"] = "Endpoint did not return JSON"
				return
			}
			entry["data"] = servicePayload(path, payload)
			entry["ok"] = response.StatusCode >= 200 && response.StatusCode < 300
		}(i, path)
	}
	wg.Wait()
	out := map[string]any{"fetched_at": time.Now().UTC(), "scope": "All traffic through this Headroom service; not limited to CPA", "endpoints": results}
	cfg.service.at = time.Now()
	cfg.service.data = out
	return out
}
