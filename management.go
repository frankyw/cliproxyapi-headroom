package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed dashboard.html
var dashboard []byte

func managementRegistration() ([]byte, error) {
	return okEnvelope(map[string]any{"resources": []map[string]any{{"Path": "/stats", "Menu": "Headroom Stats", "Description": "CPA compression savings, latency and history"}, {"Path": "/stats-data"}, {"Path": "/service-data"}}, "routes": []map[string]any{{"Method": "GET", "Path": "/plugins/headroom/stats"}, {"Method": "GET", "Path": "/plugins/headroom/service-stats"}}})
}
func managementHandle(raw []byte) ([]byte, error) {
	var req struct{ Method, Path string }
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	status := 200
	contentType := "application/json"
	body := []byte(`{"error":"not found"}`)
	switch {
	case req.Method != "GET":
		status = 405
	case req.Path == "/v0/resource/plugins/headroom/stats":
		contentType = "text/html; charset=utf-8"
		body = dashboard
	case req.Path == "/v0/management/plugins/headroom/stats" || req.Path == "/v0/resource/plugins/headroom/stats-data":
		cfg := settings.Load()
		if cfg == nil || cfg.stats == nil {
			status = 503
			body = []byte(`{"error":"statistics unavailable"}`)
		} else {
			body, _ = json.Marshal(cfg.stats.snapshot())
		}
	case req.Path == "/v0/management/plugins/headroom/service-stats" || req.Path == "/v0/resource/plugins/headroom/service-data":
		cfg := settings.Load()
		if cfg == nil {
			status = 503
		} else {
			body, _ = json.Marshal(fetchService(cfg))
		}
	default:
		status = 404
	}
	headers := map[string][]string{"Content-Type": {contentType}, "Cache-Control": {"no-store"}, "X-Content-Type-Options": {"nosniff"}, "Referrer-Policy": {"no-referrer"}}
	if contentType == "text/html; charset=utf-8" {
		headers["Content-Security-Policy"] = []string{"default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'self'; base-uri 'none'; form-action 'none'"}
	}
	return okEnvelope(struct {
		StatusCode int
		Headers    map[string][]string
		Body       []byte
	}{status, headers, body})
}
