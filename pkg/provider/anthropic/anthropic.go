// Package anthropic implements the Anthropic Claude provider
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"sgpt/pkg/config"
	"sgpt/pkg/logsafe"
	"sgpt/pkg/provider"
	"sgpt/pkg/transport"
)

// API endpoint for Anthropic Claude
const (
	MessagesEndpoint = "https://api.anthropic.com/v1/messages"
	StreamEvent      = "content_block_delta"
)

// Provider implements the LLMProvider interface for Anthropic Claude
type Provider struct {
	client *transport.Client
	apiKey string
	logger *slog.Logger
}

// NewProvider creates a new Anthropic provider
func NewProvider(client *transport.Client, cfg config.Config) *Provider {
	return &Provider{
		client: client,
		apiKey: cfg.APIKey,
		logger: cfg.Logger,
	}
}

// Complete sends a request to Anthropic and returns the response
func (p *Provider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	// Prepare request payload
	jsonData, err := p.prepareRequestPayload(req, false)
	if err != nil {
		return provider.Response{}, err
	}

	// Log the request payload (safely)
	if p.logger != nil {
		logsafe.DumpJSON(p.logger, "Anthropic request payload", jsonData)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", MessagesEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", p.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	// Send request
	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return provider.Response{}, fmt.Errorf("%w: %v", provider.ErrAPIRequestFailed, err)
	}

	// Read response body
	body, err := p.client.ReadAll(resp)
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return provider.Response{}, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return provider.Response{}, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// Parse the response
	text, err := p.parseResponse(body)
	if err != nil {
		return provider.Response{}, err
	}

	return provider.Response{
		Text: text,
		Raw:  body,
	}, nil
}

// StreamComplete sends a request to Anthropic and streams the response
func (p *Provider) StreamComplete(ctx context.Context, req provider.Request) error {
	// Prepare request payload
	jsonData, err := p.prepareRequestPayload(req, true)
	if err != nil {
		return err
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", MessagesEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", p.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", provider.ErrAPIRequestFailed, err)
	}
	defer resp.Body.Close()

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// Process SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "event: ") && !strings.HasPrefix(line, "data: ") {
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if eventType != StreamEvent {
				continue // Skip non-content events
			}
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var contentBlock struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}

			if err := json.Unmarshal([]byte(data), &contentBlock); err != nil {
				p.logger.Warn("failed to parse content block", "error", err)
				continue
			}

			if contentBlock.Type == "content_block_delta" && contentBlock.Delta.Text != "" {
				fmt.Print(contentBlock.Delta.Text)
				// Flush stdout to ensure tokens appear immediately
				os.Stdout.Sync()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	fmt.Println() // End with a newline
	return nil
}

// prepareRequestPayload creates the JSON payload for the Anthropic API request
func (p *Provider) prepareRequestPayload(req provider.Request, stream bool) ([]byte, error) {
	requestBody := map[string]interface{}{
		"model":       req.Model,
		"max_tokens":  1000,
		"temperature": req.Temperature,
		"stream":      stream,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": req.Input,
			},
		},
		"system": req.Instruction,
	}

	return json.Marshal(requestBody)
}

// parseResponse extracts the text from the Anthropic API response
func (p *Provider) parseResponse(body []byte) (string, error) {
	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	var textContent string
	for _, content := range response.Content {
		if content.Type == "text" {
			textContent += content.Text
		}
	}

	if textContent == "" {
		return "", provider.ErrNoResponseGenerated
	}

	return textContent, nil
}
