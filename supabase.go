package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// sbClient is a minimal Supabase REST client — no extra deps, plain net/http.
type sbClient struct {
	baseURL    string
	serviceKey string
	http       *http.Client
}

// newSupabaseClient builds a client from env vars.
// Required env vars: SUPABASE_URL, SUPABASE_SERVICE_KEY
func newSupabaseClient() (*sbClient, error) {
	u := os.Getenv("SUPABASE_URL")
	k := os.Getenv("SUPABASE_SERVICE_KEY")
	if u == "" || k == "" {
		return nil, fmt.Errorf("SUPABASE_URL or SUPABASE_SERVICE_KEY not set")
	}
	return &sbClient{
		baseURL:    u,
		serviceKey: k,
		http:       &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// do executes a request against the Supabase REST API and decodes the response.
func (c *sbClient) do(method, path string, body interface{}, headers map[string]string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+"/rest/v1"+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

// ── self-registration ─────────────────────────────────────────────────────────

// selfURL returns the public URL of this Replit instance, derived from
// REPLIT_DEV_DOMAIN. Falls back to a localhost address if the env var is absent
// so the tool still works outside Replit.
func selfURL() string {
	if domain := os.Getenv("REPLIT_DEV_DOMAIN"); domain != "" {
		return "https://" + domain
	}
	return "http://localhost:8080"
}

// RegisterSelf checks whether this instance's URL is already recorded in
// layer7_apis. If not, it inserts a new row. This is called once at startup.
func RegisterSelf(c *sbClient) error {
	url := selfURL()

	// ── check existing ────────────────────────────────────────────────────────
	raw, status, err := c.do("GET", "/layer7_apis?url=eq."+url+"&select=id", nil, nil)
	if err != nil {
		return fmt.Errorf("supabase check: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("supabase check returned %d: %s", status, raw)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("supabase decode: %w", err)
	}
	if len(rows) > 0 {
		log.Printf("[supabase] URL already registered: %s (id=%v)", url, rows[0]["id"])
		return nil
	}

	// ── insert ────────────────────────────────────────────────────────────────
	payload := map[string]string{"url": url}
	raw, status, err = c.do("POST", "/layer7_apis", payload, nil)
	if err != nil {
		return fmt.Errorf("supabase insert: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("supabase insert returned %d: %s", status, raw)
	}

	log.Printf("[supabase] registered new instance: %s", url)
	return nil
}
