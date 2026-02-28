package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/guohao117/clashtui/internal/config"
)

// ClashLog represents a single Clash log entry.
type ClashLog struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time,omitempty"`
}

// ProxyGroup represents a Clash proxy group.
type ProxyGroup struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Now  string   `json:"now"`
	All  []string `json:"all"`
}

// TrafficSnapshot holds upload/download byte counts at a point in time.
type TrafficSnapshot struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// ConnectionsSummary holds aggregate connection data.
type ConnectionsSummary struct {
	ActiveCount int
	TotalBytes  int64
}

// Client wraps an http.Client tailored for the Clash RESTful API.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a Client from the given config.
func NewClient(cfg config.Config) *Client {
	return &Client{
		baseURL:   strings.TrimRight(cfg.ClashHost, "/"),
		authToken: cfg.AuthToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		},
	}
}

// newRequest creates an authorized HTTP request.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	return req, nil
}

// SwitchMode changes the Clash running mode (global / rule / direct).
func (c *Client) SwitchMode(ctx context.Context, mode string) error {
	body := fmt.Sprintf(`{"mode":"%s"}`, strings.ToLower(mode))
	req, err := c.newRequest(ctx, http.MethodPatch, "/configs", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// FetchProxyGroups returns all Selector-type proxy groups.
func (c *Client) FetchProxyGroups(ctx context.Context, mode string) ([]ProxyGroup, error) {
	if strings.EqualFold(mode, "direct") {
		return []ProxyGroup{{
			Name: "Direct",
			Type: "Direct",
			Now:  "DIRECT",
			All:  []string{"DIRECT"},
		}}, nil
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/proxies", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Proxies map[string]json.RawMessage `json:"proxies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	isGlobalMode := strings.EqualFold(mode, "global")
	groups := make([]ProxyGroup, 0, len(result.Proxies))

	for name, raw := range result.Proxies {
		isGlobal := strings.EqualFold(name, "GLOBAL")
		if isGlobalMode && !isGlobal {
			continue
		}
		if !isGlobalMode && isGlobal {
			continue
		}

		var proxy struct {
			Type string   `json:"type"`
			Now  string   `json:"now"`
			All  []string `json:"all"`
		}
		if err := json.Unmarshal(raw, &proxy); err != nil {
			continue
		}
		if proxy.Type != "Selector" {
			continue
		}

		groups = append(groups, ProxyGroup{
			Name: name,
			Type: proxy.Type,
			Now:  proxy.Now,
			All:  proxy.All,
		})
	}

	return groups, nil
}

// SwitchProxy selects a proxy within a group.
func (c *Client) SwitchProxy(ctx context.Context, group, proxy string) error {
	body := fmt.Sprintf(`{"name":"%s"}`, proxy)
	req, err := c.newRequest(ctx, http.MethodPut, "/proxies/"+group, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// FetchTraffic returns the latest traffic snapshot from the streaming endpoint.
func (c *Client) FetchTraffic(ctx context.Context) (TrafficSnapshot, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/traffic", nil)
	if err != nil {
		return TrafficSnapshot{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TrafficSnapshot{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var snap TrafficSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return TrafficSnapshot{}, fmt.Errorf("decode: %w", err)
	}
	return snap, nil
}

// FetchConnections returns a summary of current connections.
func (c *Client) FetchConnections(ctx context.Context) (ConnectionsSummary, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/connections", nil)
	if err != nil {
		return ConnectionsSummary{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ConnectionsSummary{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		DownloadTotal int64 `json:"downloadTotal"`
		UploadTotal   int64 `json:"uploadTotal"`
		Connections   []struct {
			ID string `json:"id"`
		} `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ConnectionsSummary{}, fmt.Errorf("decode: %w", err)
	}

	return ConnectionsSummary{
		ActiveCount: len(result.Connections),
		TotalBytes:  result.DownloadTotal + result.UploadTotal,
	}, nil
}

// StreamLogs reads logs from the streaming endpoint, sending batches
// to the provided channel. It blocks until ctx is cancelled or an error occurs.
func (c *Client) StreamLogs(ctx context.Context, ch chan<- []ClashLog) error {
	const (
		batchSize    = 10
		batchTimeout = 500 * time.Millisecond
	)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := c.streamOnce(ctx, ch, batchSize, batchTimeout); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				// retry
			}
		}
	}
}

func (c *Client) streamOnce(ctx context.Context, ch chan<- []ClashLog, batchSize int, batchTimeout time.Duration) error {
	// Use a separate client without global timeout for streaming.
	streamClient := &http.Client{
		Transport: c.httpClient.Transport,
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/logs", nil)
	if err != nil {
		return err
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	batch := make([]ClashLog, 0, batchSize)
	timer := time.NewTimer(batchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				ch <- batch
			}
			return ctx.Err()
		case <-timer.C:
			if len(batch) > 0 {
				ch <- batch
				batch = make([]ClashLog, 0, batchSize)
			}
			timer.Reset(batchTimeout)
		default:
			var log ClashLog
			if err := decoder.Decode(&log); err != nil {
				if len(batch) > 0 {
					ch <- batch
				}
				if err == io.EOF {
					return nil
				}
				return err
			}
			batch = append(batch, log)
			if len(batch) >= batchSize {
				ch <- batch
				batch = make([]ClashLog, 0, batchSize)
				timer.Reset(batchTimeout)
			}
		}
	}
}
