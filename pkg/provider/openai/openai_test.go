package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"sgpt/pkg/config"
	"sgpt/pkg/provider"
	"sgpt/pkg/transport"
)

// TestURLOverrider allows overriding URLs for testing
type testURL struct {
	ChatCompletionsURL string
	CompletionsURL     string
}

func TestOpenAIProvider_Complete(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization: Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		// Verify request is properly formed
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Check the endpoint
		switch r.URL.Path {
		case "/v1/chat/completions":
			// For chat completions
			messages, ok := requestBody["messages"].([]interface{})
			if !ok || len(messages) < 2 {
				t.Errorf("Expected at least 2 messages, got %v", messages)
			}

			// Return a mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"id": "mock-id",
				"object": "chat.completion",
				"created": 1677858242,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "This is a test response"
						},
						"finish_reason": "stop",
						"index": 0
					}
				],
				"usage": {
					"prompt_tokens": 13,
					"completion_tokens": 7,
					"total_tokens": 20
				}
			}`
			_, _ = w.Write([]byte(response))

		case "/v1/completions":
			// For legacy completions
			prompt, ok := requestBody["prompt"].(string)
			if !ok || prompt == "" {
				t.Errorf("Expected prompt, got %v", prompt)
			}

			// Return a mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"id": "mock-id",
				"object": "text_completion",
				"created": 1677858242,
				"model": "text-davinci-003",
				"choices": [
					{
						"text": "This is a legacy model response",
						"finish_reason": "stop",
						"index": 0
					}
				],
				"usage": {
					"prompt_tokens": 13,
					"completion_tokens": 7,
					"total_tokens": 20
				}
			}`
			_, _ = w.Write([]byte(response))

		default:
			t.Errorf("Unexpected endpoint: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors during tests
	}))

	// Create a transport client for testing
	client := transport.NewClient(logger)

	// Create a config for testing
	cfg := config.Config{
		APIKey: "test-key",
		Logger: logger,
	}

	// Create the provider using NewProvider
	p := NewProvider(client, cfg)

	// Create a custom function to handle test requests with the mock server URL
	testComplete := func(ctx context.Context, req provider.Request) (provider.Response, error) {
		// Create a test request
		var jsonData []byte
		var err error
		var endpoint string

		// Determine endpoint based on model
		if req.Model == "text-davinci-003" {
			endpoint = server.URL + "/v1/completions"
		} else {
			endpoint = server.URL + "/v1/chat/completions"
		}

		// Prepare the request payload
		jsonData, err = p.prepareRequestPayload(req, false)
		if err != nil {
			return provider.Response{}, err
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
		if err != nil {
			return provider.Response{}, err
		}

		// Set headers
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		// Send request
		resp, err := p.client.Do(ctx, httpReq)
		if err != nil {
			return provider.Response{}, err
		}

		// Read response body
		body, err := p.client.ReadAll(resp)
		if err != nil {
			return provider.Response{}, err
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

	// Test cases
	testCases := []struct {
		name        string
		request     provider.Request
		expectError bool
		expected    string
	}{
		{
			name: "Chat model request",
			request: provider.Request{
				Model:       "gpt-3.5-turbo",
				Instruction: "You are a test assistant",
				Input:       "This is a test",
				Temperature: 0.5,
			},
			expectError: false,
			expected:    "This is a test response",
		},
		{
			name: "Legacy model request",
			request: provider.Request{
				Model:       "text-davinci-003",
				Instruction: "You are a test assistant",
				Input:       "This is a test",
				Temperature: 0.5,
			},
			expectError: false,
			expected:    "This is a legacy model response",
		},
		{
			name: "Invalid model",
			request: provider.Request{
				Model:       "invalid-model",
				Instruction: "You are a test assistant",
				Input:       "This is a test",
				Temperature: 0.5,
			},
			expectError: true,
			expected:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp provider.Response
			var err error

			if tc.request.Model == "invalid-model" {
				// For invalid model test, use the real Complete method which will validate models
				resp, err = p.Complete(context.Background(), tc.request)
			} else {
				// For valid models in tests, use our mock server
				resp, err = testComplete(context.Background(), tc.request)
			}

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if resp.Text != tc.expected {
					t.Errorf("Expected response %q, got %q", tc.expected, resp.Text)
				}
			}
		})
	}
}
