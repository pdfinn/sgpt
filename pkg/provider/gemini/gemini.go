// Package gemini implements the Google Gemini provider
package gemini

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

// Endpoint for Google Gemini API
const (
	APIEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
)

// Provider implements the LLMProvider interface for Google Gemini
type Provider struct {
	client *transport.Client
	apiKey string
	logger *slog.Logger
}

// NewProvider creates a new Google Gemini provider
func NewProvider(client *transport.Client, cfg config.Config) *Provider {
	return &Provider{
		client: client,
		apiKey: cfg.APIKey,
		logger: cfg.Logger,
	}
}

// Complete sends a request to Google Gemini and returns the response
func (p *Provider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	// Prepare request payload
	jsonData, err := p.prepareRequestPayload(req, false)
	if err != nil {
		return provider.Response{}, err
	}

	// Log the request payload (safely)
	if p.logger != nil {
		logsafe.DumpJSON(p.logger, "Google Gemini request payload", jsonData)
	}

	// Create API endpoint URL with model name
	endpoint := fmt.Sprintf(APIEndpoint, req.Model)

	// Add API key to URL
	endpoint = fmt.Sprintf("%s?key=%s", endpoint, p.apiKey)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

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
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
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

// StreamComplete sends a request to Google Gemini and streams the response
func (p *Provider) StreamComplete(ctx context.Context, req provider.Request) error {
	// Prepare request payload
	jsonData, err := p.prepareRequestPayload(req, true)
	if err != nil {
		return err
	}

	// Create API endpoint URL with model name
	endpoint := fmt.Sprintf(APIEndpoint, req.Model)

	// Add API key and streaming parameter to URL
	endpoint = fmt.Sprintf("%s?key=%s&alt=sse", endpoint, p.apiKey)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
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
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
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
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResponse struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &streamResponse); err != nil {
			p.logger.Warn("failed to parse stream response", "error", err)
			continue
		}

		if len(streamResponse.Candidates) > 0 &&
			len(streamResponse.Candidates[0].Content.Parts) > 0 {
			text := streamResponse.Candidates[0].Content.Parts[0].Text
			if text != "" {
				fmt.Print(text)
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

// prepareRequestPayload creates the JSON payload for the Google Gemini API request
func (p *Provider) prepareRequestPayload(req provider.Request, stream bool) ([]byte, error) {
	// Build the contents array for the request
	contents := []map[string]interface{}{
		{
			"role": "user",
			"parts": []map[string]interface{}{
				{
					"text": req.Input,
				},
			},
		},
	}

	// Add image if provided
	if req.ImagePath != "" {
		imageData, err := provider.LoadImageAsBase64(req.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load image: %w", err)
		}

		// If it's a URL, add it differently than base64 data
		var imagePart map[string]interface{}
		if strings.HasPrefix(imageData, "http") {
			imagePart = map[string]interface{}{
				"inline_data": map[string]interface{}{
					"mime_type": "image/jpeg",
					"url":       imageData,
				},
			}
		} else {
			// Remove the data URI prefix
			imageData = strings.TrimPrefix(imageData, "data:image/png;base64,")
			imagePart = map[string]interface{}{
				"inline_data": map[string]interface{}{
					"mime_type": "image/jpeg",
					"data":      imageData,
				},
			}
		}

		// Add the image part to the user's content
		contents[0]["parts"] = append(contents[0]["parts"].([]map[string]interface{}), imagePart)
	}

	// Add system instruction if provided
	if req.Instruction != "" {
		requestBody := map[string]interface{}{
			"contents": contents,
			"systemInstruction": map[string]interface{}{
				"parts": []map[string]interface{}{
					{
						"text": req.Instruction,
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature": req.Temperature,
			},
		}

		return json.Marshal(requestBody)
	}

	// Build request without system instruction
	requestBody := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature": req.Temperature,
		},
	}

	return json.Marshal(requestBody)
}

// parseResponse extracts the text from the Google Gemini API response
func (p *Provider) parseResponse(body []byte) (string, error) {
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 {
		return "", provider.ErrNoResponseGenerated
	}

	return response.Candidates[0].Content.Parts[0].Text, nil
}
