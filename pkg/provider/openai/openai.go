// Package openai implements the OpenAI provider
package openai

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

// Endpoints for OpenAI API
const (
	ChatCompletionsEndpoint = "https://api.openai.com/v1/chat/completions"
	CompletionsEndpoint     = "https://api.openai.com/v1/completions"
)

// Provider implements the LLMProvider interface for OpenAI
type Provider struct {
	client *transport.Client
	apiKey string
	logger *slog.Logger
}

// NewProvider creates a new OpenAI provider
func NewProvider(client *transport.Client, cfg config.Config) *Provider {
	return &Provider{
		client: client,
		apiKey: cfg.APIKey,
		logger: cfg.Logger,
	}
}

// Complete sends a request to OpenAI and returns the response
func (p *Provider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	// Prepare request payload based on model
	jsonData, err := p.prepareRequestPayload(req, false)
	if err != nil {
		return provider.Response{}, err
	}

	// Log the request payload (safely)
	if p.logger != nil {
		logsafe.DumpJSON(p.logger, "OpenAI request payload", jsonData)
	}

	// Determine the appropriate endpoint
	endpoint := p.determineEndpoint(req.Model)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return provider.Response{}, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return provider.Response{}, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// Parse the response
	text, err := p.parseResponse(body, req.Model)
	if err != nil {
		return provider.Response{}, err
	}

	return provider.Response{
		Text: text,
		Raw:  body,
	}, nil
}

// StreamComplete sends a request to OpenAI and streams the response
func (p *Provider) StreamComplete(ctx context.Context, req provider.Request) error {
	// Only supported for chat models
	if !strings.HasPrefix(req.Model, "gpt-") {
		return fmt.Errorf("%w: streaming not supported for model %s", provider.ErrUnsupportedModel, req.Model)
	}

	// Prepare request payload
	jsonData, err := p.prepareRequestPayload(req, true)
	if err != nil {
		return err
	}

	// Determine the appropriate endpoint
	endpoint := p.determineEndpoint(req.Model)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
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
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// Process SSE stream
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		event, data := parts[0], parts[1]
		if event != "data" {
			continue
		}

		if data == "[DONE]" {
			break
		}

		// Parse data as JSON
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			p.logger.Warn("error parsing chunk", "error", err, "data", data)
			continue
		}

		// Extract content from the chunk
		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fmt.Print(content)
				// Flush stdout to ensure tokens appear immediately
				os.Stdout.Sync()
			}

			// Check if we're done
			if chunk.Choices[0].FinishReason != "" {
				break
			}
		}
	}

	fmt.Println() // End with a newline
	return nil
}

// determineEndpoint returns the appropriate API endpoint for the given model
func (p *Provider) determineEndpoint(model string) string {
	// Chat models use the chat completions endpoint
	if strings.HasPrefix(model, "gpt-") {
		return ChatCompletionsEndpoint
	}
	// Legacy models use the completions endpoint
	return CompletionsEndpoint
}

// prepareRequestPayload creates the JSON payload for the OpenAI API request
func (p *Provider) prepareRequestPayload(req provider.Request, stream bool) ([]byte, error) {
	switch req.Model {
	case "gpt-4o":
		// Handle multimodal input for GPT-4o
		var messages []map[string]interface{}

		// Add system message if instruction is provided
		if req.Instruction != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": req.Instruction,
			})
		}

		// Build user content
		var userContent []map[string]interface{}

		// Add text input
		if req.Input != "" {
			userContent = append(userContent, map[string]interface{}{
				"type": "text",
				"text": req.Input,
			})
		}

		// Add image content
		if req.ImagePath != "" {
			imageURL, err := provider.LoadImageAsBase64(req.ImagePath)
			if err != nil {
				return nil, err
			}

			userContent = append(userContent, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]string{
					"url": imageURL,
				},
			})
		}

		// Add audio content
		if req.AudioPath != "" {
			audioData, err := provider.LoadAudioAsBase64(req.AudioPath)
			if err != nil {
				return nil, err
			}

			userContent = append(userContent, map[string]interface{}{
				"type": "audio",
				"audio": map[string]string{
					"data": audioData,
				},
			})
		}

		// Add the user message
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": userContent,
		})

		// Prepare JSON data
		requestBody := map[string]interface{}{
			"model":       req.Model,
			"messages":    messages,
			"temperature": req.Temperature,
			"stream":      stream,
		}

		return json.Marshal(requestBody)

	case "gpt-4", "gpt-4-0314", "gpt-4-32k", "gpt-4-32k-0314", "gpt-3.5-turbo":
		// Handle chat models (text only)
		messages := []map[string]string{
			{"role": "system", "content": req.Instruction},
			{"role": "user", "content": req.Input},
		}

		requestBody := map[string]interface{}{
			"model":       req.Model,
			"messages":    messages,
			"temperature": req.Temperature,
			"stream":      stream,
		}

		return json.Marshal(requestBody)

	case "text-davinci-003", "text-davinci-002", "text-curie-001", "text-babbage-001", "text-ada-001":
		// Handle legacy completions models
		prompt := req.Instruction + " " + req.Input
		requestBody := map[string]interface{}{
			"model":       req.Model,
			"prompt":      prompt,
			"temperature": req.Temperature,
			"stream":      stream,
		}

		return json.Marshal(requestBody)

	default:
		return nil, fmt.Errorf("%w: %s", provider.ErrUnsupportedModel, req.Model)
	}
}

// parseResponse extracts the text from the OpenAI API response
func (p *Provider) parseResponse(body []byte, model string) (string, error) {
	// For chat models
	if strings.HasPrefix(model, "gpt-") {
		var response struct {
			Choices []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		err := json.Unmarshal(body, &response)
		if err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		if len(response.Choices) == 0 {
			return "", provider.ErrNoResponseGenerated
		}

		return strings.TrimSpace(response.Choices[0].Message.Content), nil
	}

	// For legacy completions models
	var response struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", provider.ErrNoResponseGenerated
	}

	return strings.TrimSpace(response.Choices[0].Text), nil
}
