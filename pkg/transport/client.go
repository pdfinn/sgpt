// Package transport provides shared HTTP client functionality
package transport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client wraps a shared HTTP client with context support
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new HTTP client with reasonable defaults
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// Do performs an HTTP request with context support
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	c.logger.Debug("sending HTTP request", "method", req.Method, "url", req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err)
		return nil, err
	}

	c.logger.Debug("received HTTP response", "status", resp.Status, "content_type", resp.Header.Get("Content-Type"))
	return resp, nil
}

// ReadAll reads the entire body of the response with context awareness
func (c *Client) ReadAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	// Use io.ReadAll instead of ioutil.ReadAll
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err)
		return nil, err
	}

	return body, nil
}
