package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Config is the single source of truth for all runtime settings.
// Loaded from config.json next to the binary at startup.
type Config struct {
	Supabase struct {
		URL        string `json:"url"`
		ServiceKey string `json:"service_key"`
	} `json:"supabase"`
	API struct {
		Port int `json:"port"`
	} `json:"api"`
	Defaults struct {
		Method    string `json:"method"`
		Workers   int    `json:"workers"`
		Duration  int    `json:"duration"`
		RateDelay int    `json:"rate_delay"`
		Verbose   bool   `json:"verbose"`
	} `json:"defaults"`
}

// cfg is the global loaded configuration.
var cfg Config

// loadConfig reads config.json from the same directory as the running binary.
// If the file is missing or malformed the process exits immediately — the tool
// cannot run safely without its configuration.
func loadConfig() {
	path := configPath()

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("[config] cannot open %s: %v", path, err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("[config] cannot parse %s: %v", path, err)
	}

	// apply built-in fallbacks so an incomplete file still works
	if cfg.API.Port == 0 {
		cfg.API.Port = 8080
	}
	if cfg.Defaults.Method == "" {
		cfg.Defaults.Method = "httpget"
	}
	if cfg.Defaults.Workers == 0 {
		cfg.Defaults.Workers = 512
	}
	if cfg.Defaults.Duration == 0 {
		cfg.Defaults.Duration = 30
	}

	log.Printf("[config] loaded from %s", path)
}

// configPath resolves the location of config.json.
// Priority: CONFIG_PATH env var → directory of the executable → current dir.
func configPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}

	// directory of the running binary
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(filename)
		candidate := filepath.Join(dir, "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// fallback: working directory
	return "config.json"
}
