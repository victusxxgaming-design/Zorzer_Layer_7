package main

// AttackConfig holds all parameters for a single attack run.
// Mirrors the CLI flags so both surfaces share one definition.
type AttackConfig struct {
	Target    string `json:"target"`
	Method    string `json:"method"`
	Workers   int    `json:"workers"`
	Duration  int    `json:"duration"`
	ProxyFile string `json:"proxy_file"`
	Verbose   bool   `json:"verbose"`
	RateDelay int    `json:"rate_delay"`
}

// StartRequest is the JSON body expected by POST /api/attack/start.
type StartRequest struct {
	Target    string `json:"target"`
	Method    string `json:"method"`
	Workers   int    `json:"workers"`
	Duration  int    `json:"duration"`
	ProxyFile string `json:"proxy_file"`
	Verbose   bool   `json:"verbose"`
	RateDelay int    `json:"rate_delay"`
}

// StatusResponse is returned by GET /api/attack/status.
type StatusResponse struct {
	Running   bool         `json:"running"`
	Config    *AttackConfig `json:"config,omitempty"`
	Stats     *LiveStats   `json:"stats,omitempty"`
	StartedAt string       `json:"started_at,omitempty"`
	ElapsedS  float64      `json:"elapsed_seconds,omitempty"`
}

// LiveStats is the real-time counter snapshot embedded in StatusResponse.
type LiveStats struct {
	Sent    int64   `json:"sent"`
	Errors  int64   `json:"errors"`
	NonSucc int64   `json:"non_2xx"`
	RPS     float64 `json:"rps"`
}

// APIResponse is the standard JSON envelope for all API replies.
type APIResponse struct {
	OK      bool        `json:"ok"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
