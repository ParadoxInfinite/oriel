// Package provider is the dormant extension seam for natural-language actions.
// The base ships NO model. If ORIEL_PROVIDER_URL is set, the base POSTs the
// user's text plus the available tools and live entities to that URL, and the
// provider (a separate process — rules, embeddings, or an LLM) returns a tool
// call. The returned call is always re-validated through the tool Registry
// before it executes, so a provider can never invoke an unknown tool or entity.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// EnvURL configures the provider endpoint. Unset means the seam is dormant.
const EnvURL = "ORIEL_PROVIDER_URL"

// ToolCall is what a provider must return: which tool to run with what args.
type ToolCall struct {
	Tool       string         `json:"tool"`
	Args       map[string]any `json:"args"`
	Confidence float64        `json:"confidence"`
}

// Request is the body POSTed to <url>/resolve.
type Request struct {
	Text     string              `json:"text"`
	Tools    any                 `json:"tools"`
	Entities map[string][]string `json:"entities"`
}

// Provider is an HTTP client for a configured resolver. The URL can be set from
// the environment at startup or swapped at runtime (Settings → AI), so the mutex
// guards it against concurrent request and config goroutines.
type Provider struct {
	mu   sync.RWMutex
	url  string
	http *http.Client
}

// New reads the endpoint from the environment. Resolve fails until configured.
func New() *Provider {
	return &Provider{
		url:  normalizeURL(os.Getenv(EnvURL)),
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func normalizeURL(u string) string { return strings.TrimRight(strings.TrimSpace(u), "/") }

// Enabled reports whether a provider URL is configured.
func (p *Provider) Enabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.url != ""
}

// URL returns the configured endpoint, empty when dormant.
func (p *Provider) URL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.url
}

// SetURL swaps the endpoint at runtime; an empty string returns to dormant.
func (p *Provider) SetURL(u string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.url = normalizeURL(u)
}

// Resolve asks the provider to map req.Text to a tool call.
func (p *Provider) Resolve(ctx context.Context, req Request) (ToolCall, error) {
	url := p.URL()
	if url == "" {
		return ToolCall{}, errors.New("no provider configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return ToolCall{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/resolve", bytes.NewReader(body))
	if err != nil {
		return ToolCall{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return ToolCall{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ToolCall{}, fmt.Errorf("provider returned %s", resp.Status)
	}
	var tc ToolCall
	if err := json.NewDecoder(resp.Body).Decode(&tc); err != nil {
		return ToolCall{}, fmt.Errorf("decode provider response: %w", err)
	}
	if tc.Tool == "" {
		return ToolCall{}, errors.New("provider returned no tool")
	}
	return tc, nil
}
